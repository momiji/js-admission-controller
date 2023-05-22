package admission

import "testing"

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
