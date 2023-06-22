package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/momiji/js-admissions-controller/logs"
	"github.com/momiji/js-admissions-controller/utils"
	"github.com/snorwin/jsonpatch"
	admission "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func serveMutate(w http.ResponseWriter, r *http.Request) {
	serve(w, r, mutate)
}
func serveValidate(w http.ResponseWriter, r *http.Request) {
	serve(w, r, validate)
}

type admitv1Func func(*admission.AdmissionReview) *admission.AdmissionResponse

// serve handles the http portion of a request prior to handing to an admit function
func serve(w http.ResponseWriter, r *http.Request, admit admitv1Func) {
	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		logs.Errorf("Invalid contentType %s, expect application/json", contentType)
		return
	}

	// get body content
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	//logs.Infof("handling request: %s", body)

	// deserialize json content
	var responseObj runtime.Object
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		logs.Errorf(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	requestedAdmissionReview, ok := obj.(*admission.AdmissionReview)
	if !ok {
		logs.Errorf("Expected v1.AdmissionReview but got: %T", obj)
		return
	}

	// call admission mutate/validate func
	responseAdmissionReview := &admission.AdmissionReview{}
	responseAdmissionReview.SetGroupVersionKind(*gvk)
	responseAdmissionReview.Response = admit(requestedAdmissionReview)
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	responseObj = responseAdmissionReview

	// send response
	//logs.Infof("sending response: %v", responseObj)
	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		logs.Errorf("Unable to marshal admission response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(respBytes); err != nil {
		logs.Errorf("Unable to send admission response: %v", err)
	}
}

func mutate(ar *admission.AdmissionReview) *admission.AdmissionResponse {
	// skip if no admissions
	adms := admissions.Find(utils.GVK1ToString(ar.Request.Kind), ar.Request.Namespace)
	if len(adms) == 0 {
		return &admission.AdmissionResponse{Allowed: true}
	}

	// log
	name := ar.Request.Name
	logs.Tracef("Mutate: %s %s ns=%s name=%s", ar.Request.Operation, utils.GVK1ToString(ar.Request.Kind), ar.Request.Namespace, name)
	showLog := func(show bool, res string) {
		if show {
			logs.Infof("Mutate: %s %s ns=%s name=%s - %s", ar.Request.Operation, utils.GVK1ToString(ar.Request.Kind), ar.Request.Namespace, name, res)
		}
	}

	// dcode object
	raw := ar.Request.Object.Raw
	if ar.Request.Operation == "DELETE" {
		raw = ar.Request.OldObject.Raw
	}
	obj, _, _ := unstructured.UnstructuredJSONScheme.Decode(raw, nil, nil)
	uObj, ok := obj.(*unstructured.Unstructured)
	if obj != nil && !ok {
		showLog(true, "Error")
		logs.Errorf("Mutate failed, item is not *unstructured.Unstructured: %s", ar.Request.Object.Raw)
		return &admission.AdmissionResponse{Allowed: true}
	}
	newUObj := &unstructured.Unstructured{Object: uObj.Object}
	changed := false
	mutations := make([]string, 0)

	if ar.Request.Operation == "CREATE" && name == "" {
		name = uObj.GetGenerateName() + "???"
	}

	for _, code := range adms {
		res, err := code.Mutate(ar.Request.Operation, newUObj.DeepCopy())
		if err != nil {
			showLog(true, "Error")
			logs.Errorf("Error in mutate: %v", err)
			return &admission.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		if res != nil {
			allowed, b, e := unstructured.NestedBool(res.Object, "Allowed")
			if (b && e == nil) && !allowed {
				message, _, _ := unstructured.NestedString(res.Object, "Message")
				showLog(true, "Forbidden")
				return &admission.AdmissionResponse{
					Result: &metav1.Status{
						Message: message,
					},
				}
			}
			result, _, _ := unstructured.NestedMap(res.Object, "Result")
			if result != nil {
				newUObj.Object = result
				changed = true
				mutations = append(mutations, code.Admission.FullName())
			}
		}
	}

	if changed {
		_ = unstructured.SetNestedField(newUObj.Object, strings.Join(mutations, ","), "metadata", "annotations", "jsadmissions.momiji.com/mutate")
		patch, err := jsonpatch.CreateJSONPatch(newUObj.Object, uObj.Object)
		if err != nil {
			return &admission.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		// success
		showLog(true, fmt.Sprintf("Patch: %s", patch))
		patchType := admission.PatchTypeJSONPatch
		return &admission.AdmissionResponse{Allowed: true, PatchType: &patchType, Patch: patch.Raw()}
	}

	// success
	return &admission.AdmissionResponse{Allowed: true}
}

func validate(ar *admission.AdmissionReview) *admission.AdmissionResponse {
	// skip if no admissions
	adms := admissions.Find(utils.GVK1ToString(ar.Request.Kind), ar.Request.Namespace)
	if len(adms) == 0 {
		return &admission.AdmissionResponse{Allowed: true}
	}

	// log
	logs.Tracef("Validate: %s %s ns=%s name=%s", ar.Request.Operation, utils.GVK1ToString(ar.Request.Kind), ar.Request.Namespace, ar.Request.Name)
	doLog := false
	showLog := func(show bool, res string) {
		if show {
			logs.Infof("Validate: %s %s ns=%s name=%s - %s", ar.Request.Operation, utils.GVK1ToString(ar.Request.Kind), ar.Request.Namespace, ar.Request.Name, res)
		}
	}

	// dcode object
	raw := ar.Request.Object.Raw
	if ar.Request.Operation == "DELETE" {
		raw = ar.Request.OldObject.Raw
	}
	obj, _, _ := unstructured.UnstructuredJSONScheme.Decode(raw, nil, nil)
	uObj, ok := obj.(*unstructured.Unstructured)
	if obj != nil && !ok {
		showLog(true, "Error")
		logs.Errorf("Validate failed, item is not *unstructured.Unstructured: %s", ar.Request.Object.Raw)
		return &admission.AdmissionResponse{Allowed: true}
	}

	for _, code := range adms {
		res, err := code.Validate(ar.Request.Operation, uObj)
		if err != nil {
			showLog(true, "Error")
			logs.Errorf("Error in validate: %v", err)
			return &admission.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		if res != nil {
			allowed, b, e := unstructured.NestedBool(res.Object, "Allowed")
			doLog = doLog || (b && e == nil)
			if (b && e == nil) && !allowed {
				message, _, _ := unstructured.NestedString(res.Object, "Message")
				showLog(true, "Forbidden")
				return &admission.AdmissionResponse{
					Result: &metav1.Status{
						Message: message,
					},
				}
			}
		}
	}

	// success
	showLog(doLog, "Allowed")
	return &admission.AdmissionResponse{Allowed: true}
}
