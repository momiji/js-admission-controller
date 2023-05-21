package store

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestCache_Base(t *testing.T) {
	cache := NewCache()

	// test 0 items
	if len(cache.Find("a", "b")) != 0 {
		t.Fatalf("failed")
	}

	// test 1 item
	cache.Add("a", "b", "c1", &unstructured.Unstructured{})
	if len(cache.Find("a", "b")) != 1 {
		t.Fatalf("failed")
	}

	// test 2 items
	cache.Add("a", "b", "c2", &unstructured.Unstructured{})
	if len(cache.Find("a", "b")) != 2 {
		t.Fatalf("failed")
	}

	// test remove
	cache.Remove("a", "b", "c1")
	if len(cache.Find("a", "b")) != 1 {
		t.Fatalf("failed")
	}

}

func TestCache_Find(t *testing.T) {
	cache := NewCache()

	cache.Add("a", "", "a1", &unstructured.Unstructured{})
	cache.Add("a", "", "a2", &unstructured.Unstructured{})
	cache.Add("b", "ns1", "b1", &unstructured.Unstructured{})
	cache.Add("b", "ns2", "b2", &unstructured.Unstructured{})

	// check cluster wide Find
	if len(cache.Find("a", "")) != 2 {
		t.Fatalf("failed")
	}
	if len(cache.Find("b", "")) != 2 {
		t.Fatalf("failed")
	}

	// check namespace Find
	if len(cache.Find("a", "ns1")) != 0 {
		t.Fatalf("failed")
	}
	if len(cache.Find("b", "ns1")) != 1 {
		t.Fatalf("failed")
	}
}
