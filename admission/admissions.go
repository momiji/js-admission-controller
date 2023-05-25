package admission

import (
	"sort"
	"sync"

	admission "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Admissions struct {
	mux sync.RWMutex
	//clustered  *AdmissionList
	namespaces map[string]*AdmissionList
}

type Admission struct {
	Namespace  string
	Name       string
	Resources  []string
	Javascript string
}

type AdmissionList struct {
	admissions map[string]*AdmissionCode
}
type AdmissionCode struct {
	Admission *Admission
	Context   *JsContext
	IsValid   bool
}

func NewAdmissions() *Admissions {
	return &Admissions{
		mux: sync.RWMutex{},
		//clustered:  newAdmissionList(),
		namespaces: make(map[string]*AdmissionList),
	}
}

func newAdmissionList() *AdmissionList {
	return &AdmissionList{
		admissions: make(map[string]*AdmissionCode),
	}
}

func newAdmissionCode(adm *Admission) (*AdmissionCode, error) {
	name := adm.Name
	if adm.Namespace != "" {
		name = adm.Namespace + "/" + adm.Name
	}
	js, err := NewJsContext(name, adm.Javascript)
	if err != nil {
		return nil, err
	}
	return &AdmissionCode{
		Admission: adm,
		Context:   js,
		IsValid:   false,
	}, nil
}

func (a *Admissions) Upsert(adm *Admission) (*AdmissionCode, error) {
	a.mux.Lock()
	defer a.mux.Unlock()

	// get list
	var ok bool
	list, ok := a.namespaces[adm.Namespace]
	if !ok {
		list = newAdmissionList()
		a.namespaces[adm.Namespace] = list
	}

	// create code
	delete(list.admissions, adm.Name)
	code, err := newAdmissionCode(adm)
	if err != nil {
		return nil, err
	}

	// add code
	list.admissions[adm.Name] = code
	return code, nil
}

func (a *Admissions) Remove(namespace string, name string) {
	a.mux.Lock()
	defer a.mux.Unlock()

	// get list
	list, ok := a.namespaces[namespace]
	if !ok {
		return
	}

	// delete code
	delete(list.admissions, name)
}

// Find return admissions for current namespace and cluster if namespace != ""
func (a *Admissions) Find(resource string, namespace string) []*AdmissionCode {
	//TODO optimize or put a cache in place
	a.mux.RLock()
	defer a.mux.RUnlock()

	// list all namespace
	codes := make([]*AdmissionCode, 0)
	if list, ok := a.namespaces[namespace]; ok {
		for _, code := range list.admissions {
			if code.IsValid {
				for _, r := range code.Admission.Resources {
					if r == resource {
						codes = append(codes, code)
						break
					}
				}
			}
		}
	}
	if namespace != "" {
		if list, ok := a.namespaces[""]; ok {
			for _, code := range list.admissions {
				if code.IsValid {
					for _, r := range code.Admission.Resources {
						if r == resource {
							codes = append(codes, code)
							break
						}
					}
				}
			}
		}
	}

	// sort to have namespace then cluster, and sort by name
	sort.Slice(codes, func(i int, j int) bool {
		if codes[i].Admission.Namespace != codes[j].Admission.Namespace {
			return codes[i].Admission.Namespace > codes[j].Admission.Namespace
		}
		return codes[i].Admission.Name <= codes[j].Admission.Name
	})

	return codes
}

func (c *AdmissionCode) Init() error {
	ctx := c.Context
	_, err := ctx.Call(c.Context.JsaInit, true, map[string]interface{}{"state": &ctx.State})
	if err != nil {
		return err
	}
	return nil
}

func (c *AdmissionCode) Created(obj *unstructured.Unstructured) error {
	ctx := c.Context
	_, err := ctx.Call(c.Context.JsaCreated, false, map[string]interface{}{"state": &ctx.State, "sync": true, "obj": obj.Object})
	if err != nil {
		return err
	}
	return nil
}

func (c *AdmissionCode) Updated(obj *unstructured.Unstructured, old *unstructured.Unstructured) error {
	ctx := c.Context
	_, err := ctx.Call(c.Context.JsaUpdated, false, map[string]interface{}{"state": &ctx.State, "sync": true, "obj": obj.Object, "old": old.Object})
	if err != nil {
		return err
	}
	return nil
}

func (c *AdmissionCode) Deleted(obj *unstructured.Unstructured) error {
	ctx := c.Context
	_, err := ctx.Call(c.Context.JsaDeleted, false, map[string]interface{}{"state": &ctx.State, "sync": true, "obj": obj.Object})
	if err != nil {
		return err
	}
	return nil
}

func (c *AdmissionCode) Validate(operation admission.Operation, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	ctx := c.Context
	res, err := ctx.Call(c.Context.JsaValidate, false, map[string]interface{}{"state": &ctx.State, "sync": true, "obj": obj.Object, "op": operation})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return ToUnstructured(res.Export()), nil
}

func (c *AdmissionCode) Mutate(operation admission.Operation, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	ctx := c.Context
	res, err := ctx.Call(c.Context.JsaMutate, false, map[string]interface{}{"state": &ctx.State, "sync": true, "obj": obj.Object, "op": operation})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return ToUnstructured(res.Export()), nil
}
