package admission

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
	"testing"
)

func TestAdmissions_Find(t *testing.T) {
	adm := NewAdmissions()
	var code *AdmissionCode

	add := func(ns string, name string) {
		code, _ = adm.Upsert(&Admission{
			Namespace:  ns,
			Name:       name,
			Resources:  []string{"pods"},
			Javascript: "",
		})
		code.IsValid = true
	}
	add("", "1")
	add("", "2")
	add("ns", "3")
	add("ns2", "4")

	// check Find(ns) returns 3 items (namespace="ns" or "")
	if len(adm.Find("pods", "ns")) != 3 {
		t.Fatalf("failed")
	}

	// check Find(other) returns 2 items (only namespace="")
	if len(adm.Find("pods", "other")) != 2 {
		t.Fatalf("failed")
	}

	// check Find(*) returns 2 items (only namespace="")
	if len(adm.Find("pods", "")) != 2 {
		t.Fatalf("failed")
	}
}

func TestAdmission_Annotation(t *testing.T) {
	obj := map[string]interface{}{
		"kind": "k",
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"x": "1",
			},
		},
	}
	s, found, err := unstructured.NestedString(obj, "metadata", "annotations", "x")
	if err != nil || !found || s != "1" {
		t.Fatalf("failed")
	}
	err = unstructured.SetNestedField(obj, "2", "metadata", "annotations", "y")
	if err != nil {
		t.Fatalf("failed")
	}
	s, found, err = unstructured.NestedString(obj, "metadata", "annotations", "y")
	if err != nil || !found || s != "2" {
		t.Fatalf("failed")
	}
}

func TestAdmissions_Order(t *testing.T) {
	adm := NewAdmissions()
	var code *AdmissionCode

	add := func(ns string, name string) {
		code, _ = adm.Upsert(&Admission{
			Namespace:  ns,
			Name:       name,
			Resources:  []string{"pods"},
			Javascript: "",
		})
		code.IsValid = true
	}
	add("ns", "n3")
	add("", "c3")
	add("", "c1")
	add("ns", "n1")
	add("ns", "n2")
	add("", "c2")

	// check Find(ns) returns namespaced admissions before clustered admissions, in name order
	var names []string
	admissions := adm.Find("pods", "ns")
	for _, a := range admissions {
		names = append(names, a.Admission.Name)
	}
	if strings.Join(names, " ") != "n1 n2 n3 c1 c2 c3" {
		println(strings.Join(names, " "))
		t.Fatalf("failed")
	}
}
