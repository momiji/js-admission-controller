package admission

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestAdmissions_Find(t *testing.T) {
	adm := NewAdmissions()
	var code *AdmissionCode

	code, _ = adm.Upsert(&Admission{
		Namespace:  "",
		Name:       "1",
		Resources:  []string{"pods"},
		Javascript: "",
	})
	code.IsValid = true

	code, _ = adm.Upsert(&Admission{
		Namespace:  "",
		Name:       "2",
		Resources:  []string{"pods"},
		Javascript: "",
	})
	code.IsValid = true

	code, _ = adm.Upsert(&Admission{
		Namespace:  "ns",
		Name:       "3",
		Resources:  []string{"pods"},
		Javascript: "",
	})
	code.IsValid = true

	if len(adm.Find("pods", "ns")) != 3 {
		t.Fatalf("failed")
	}

	if len(adm.Find("pods", "ns2")) != 2 {
		t.Fatalf("failed")
	}

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
