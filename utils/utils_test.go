package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func TestGVKToString(t *testing.T) {
	//
	if GVKToString(schema.GroupVersionKind{Group: "group", Version: "version", Kind: "kind"}) != "group/version/kind" {
		t.Fatalf("failed")
	}
	if GVKToString(schema.GroupVersionKind{Group: "", Version: "version", Kind: "kind"}) != "version/kind" {
		t.Fatalf("failed")
	}
	//
	if GVRToString(schema.GroupVersionResource{Group: "group", Version: "version", Resource: "kind"}) != "group/version/kind" {
		t.Fatalf("failed")
	}
	if GVRToString(schema.GroupVersionResource{Group: "", Version: "version", Resource: "kind"}) != "version/kind" {
		t.Fatalf("failed")
	}
	//
	if GVR1ToString(metav1.GroupVersionResource{Group: "group", Version: "version", Resource: "kind"}) != "group/version/kind" {
		t.Fatalf("failed")
	}
	if GVR1ToString(metav1.GroupVersionResource{Group: "", Version: "version", Resource: "kind"}) != "version/kind" {
		t.Fatalf("failed")
	}
}
