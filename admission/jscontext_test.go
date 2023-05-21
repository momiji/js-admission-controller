package admission

import (
	"github.com/dop251/goja"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestJsContext_Goja(t *testing.T) {
	compile, err := goja.Compile("", `
		function test(i) {
			if (i==0) return;
			if (i==1) return undefined;
			if (i==2) return null;
			if (i==3) return "test";
			if (i==4) return {};
			if (i==5) return { x: 1 };
		}
	`, false)
	if err != nil {
		t.Fatalf("failed")
	}
	runtime := goja.New()
	runtime.RunProgram(compile)
	fn, _ := goja.AssertFunction(runtime.Get("test"))
	var value goja.Value
	// test 0
	value, err = fn(goja.Undefined(), runtime.ToValue(0))
	if err != nil || value.Export() != nil {
		t.Fatalf("failed")
	}
	// test 1
	value, err = fn(goja.Undefined(), runtime.ToValue(1))
	if err != nil || value.Export() != nil {
		t.Fatalf("failed")
	}
	// test 2
	value, err = fn(goja.Undefined(), runtime.ToValue(2))
	if err != nil || value.Export() != nil {
		t.Fatalf("failed")
	}
	// test 3
	value, err = fn(goja.Undefined(), runtime.ToValue(3))
	if err != nil || toMap(value.Export()) != nil {
		t.Fatalf("failed")
	}
	// test 4
	value, err = fn(goja.Undefined(), runtime.ToValue(4))
	if err != nil || toMap(value.Export()) == nil {
		t.Fatalf("failed")
	}
	// test 5
	value, err = fn(goja.Undefined(), runtime.ToValue(5))
	if err != nil || toMap(value.Export()) == nil {
		t.Fatalf("failed")
	}
	m := toMap(value.Export())
	i, f, err := unstructured.NestedInt64(m, "x")
	if err != nil || !f || i != 1 {
		t.Fatalf("failed")
	}
}

func toMap(obj interface{}) map[string]interface{} {
	if res, ok := obj.(map[string]interface{}); ok {
		return res
	}
	return nil
}
