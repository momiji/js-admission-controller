package main

import (
	"fmt"
	"testing"

	"github.com/snorwin/jsonpatch"
)

func TestJsonPatch(t *testing.T) {
	type Person struct {
		Name string  `json:"name"`
		Age  float64 `json:"age"`
	}

	original := &Person{
		Name: "John Doe",
		Age:  42,
	}
	updated := &Person{
		Name: "Jane Doe",
		Age:  21,
	}

	patch, err := jsonpatch.CreateJSONPatch(updated, original)
	if err != nil {
		t.Fatal("failed")
	}
	fmt.Println(patch.String())
}
