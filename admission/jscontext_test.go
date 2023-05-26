package admission

import (
	"encoding/json"
	"github.com/dop251/goja"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestJsContext_Objects(t *testing.T) {
	compile, err := goja.Compile("", `
function test(obj, state) {
	obj.a.push(3);
	obj.b[0] = 3
	obj.c = [1,2]
	obj.c.push(3)
	state.a = {}
	state.a.a = [1,2]
	state.b = [1,2]
	state.c = [1,2]
	state.a.a.push(3)
	state.b.push(3)
	state.c[0] = 3
	return obj
}
`, false)
	if err != nil {
		t.Fatalf("failed")
	}
	runtime := goja.New()
	_, err = runtime.RunProgram(compile)
	if err != nil {
		t.Fatalf("failed")
	}
	fn, _ := goja.AssertFunction(runtime.Get("test"))
	var value goja.Value

	// test output is a recognizable object
	obj := make(map[string]interface{})
	obj["a"] = []int{1, 2}
	obj["b"] = []int{1, 2}
	state := ToGojaObject(runtime, make(map[string]interface{}))

	value, err = fn(goja.Undefined(), ToGojaObject(runtime, obj), state)
	if err != nil || value.Export() == nil {
		t.Fatalf("failed")
	}
	jsonValue, _ := json.Marshal(value.Export())
	if string(jsonValue) != `{"a":[1,2,3],"b":[3,2],"c":[1,2,3]}` {
		t.Fatalf("failed")
	}

	// test input has not been modified, as expected now we have ToGojaObject
	jsonValue, _ = json.Marshal(state)
	if string(jsonValue) != `{"a":{"a":[1,2,3]},"b":[1,2,3],"c":[3,2]}` {
		t.Fatalf("failed")
	}
	jsonValue, _ = json.Marshal(state.Export())
	if string(jsonValue) != `{"a":{"a":[1,2,3]},"b":[1,2,3],"c":[3,2]}` {
		t.Fatalf("failed")
	}
}
func TestJsContext_Return(t *testing.T) {
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
	_, err = runtime.RunProgram(compile)
	if err != nil {
		t.Fatalf("failed")
	}
	fn, _ := goja.AssertFunction(runtime.Get("test"))
	var value goja.Value
	// test 0
	value, err = fn(goja.Undefined(), ToGojaObject(runtime, 0))
	if err != nil || value.Export() != nil {
		t.Fatalf("failed")
	}
	// test 1
	value, err = fn(goja.Undefined(), ToGojaObject(runtime, 1))
	if err != nil || value.Export() != nil {
		t.Fatalf("failed")
	}
	// test 2
	value, err = fn(goja.Undefined(), ToGojaObject(runtime, 2))
	if err != nil || value.Export() != nil {
		t.Fatalf("failed")
	}
	// test 3
	value, err = fn(goja.Undefined(), ToGojaObject(runtime, 3))
	if err != nil || ToMap(value.Export()) != nil {
		t.Fatalf("failed")
	}
	// test 4
	value, err = fn(goja.Undefined(), ToGojaObject(runtime, 4))
	if err != nil || ToMap(value.Export()) == nil {
		t.Fatalf("failed")
	}
	// test 5
	value, err = fn(goja.Undefined(), ToGojaObject(runtime, 5))
	if err != nil || ToMap(value.Export()) == nil {
		t.Fatalf("failed")
	}
	m := ToMap(value.Export())
	i, f, err := unstructured.NestedInt64(m, "x")
	if err != nil || !f || i != 1 {
		t.Fatalf("failed")
	}
}
