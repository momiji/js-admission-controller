package admission

import (
	"fmt"
	"github.com/momiji/js-admissions-controller/logs"
	"reflect"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	undefined = goja.Undefined()
)

type JsContext struct {
	mux         *sync.Mutex
	Program     *ast.Program
	Compiled    *goja.Program
	Runtime     *goja.Runtime
	State       map[string]interface{}
	JsaInit     *JsFunction
	JsaMutate   *JsFunction
	JsaValidate *JsFunction
	JsaCreated  *JsFunction
	JsaUpdated  *JsFunction
	JsaDeleted  *JsFunction
}

type JsFunction struct {
	Func   goja.Callable
	Params map[string]int
}

func NewJsContext(name string, js string) (*JsContext, error) {
	// compile code
	prg, err := goja.Parse("", js, parser.WithDisableSourceMaps)
	if err != nil {
		return nil, err
	}

	ast, err := goja.CompileAST(prg, false)
	if err != nil {
		return nil, err
	}

	// create runtime
	runtime := goja.New()
	runtime.RunProgram(ast)

	// create context
	context := JsContext{
		mux:         &sync.Mutex{},
		Program:     prg,
		Compiled:    ast,
		Runtime:     runtime,
		State:       make(map[string]interface{}),
		JsaInit:     analyseFunction(runtime, prg, "jsa_init", "state"),
		JsaMutate:   analyseFunction(runtime, prg, "jsa_mutate", "state", "sync", "obj", "op"),
		JsaValidate: analyseFunction(runtime, prg, "jsa_validate", "state", "sync", "obj", "op"),
		JsaCreated:  analyseFunction(runtime, prg, "jsa_created", "state", "sync", "obj"),
		JsaUpdated:  analyseFunction(runtime, prg, "jsa_updated", "state", "sync", "obj", "old"),
		JsaDeleted:  analyseFunction(runtime, prg, "jsa_deleted", "state", "sync", "obj"),
	}

	// add runtime utils
	runtime.Set("log", func(a ...interface{}) {
		logs.Infof("js(%s) %s\n", name, fmt.Sprint(a...))
	})
	runtime.Set("logf", func(f string, a ...interface{}) {
		logs.Infof("js(%s) %s\n", name, fmt.Sprintf(f, a...))
	})
	runtime.Set("debug", func(a ...interface{}) {
		logs.Debugf("js(%s) %s\n", name, fmt.Sprint(a...))
	})
	runtime.Set("debugf", func(f string, a ...interface{}) {
		logs.Debugf("js(%s) %s\n", name, fmt.Sprintf(f, a...))
	})

	// return
	return &context, nil
}

func analyseFunction(runtime *goja.Runtime, prg *ast.Program, name string, parameters ...string) *JsFunction {
	params := make(map[string]int)
	for _, param := range parameters {
		params[param] = 0
	}
	for _, stmt := range prg.Body {
		if decl, ok := stmt.(*ast.FunctionDeclaration); ok {
			if decl.Function.Name.Name.String() == name {
				// check this is a function
				if fn, ok := goja.AssertFunction(runtime.Get(name)); ok {
					for index, binding := range decl.Function.ParameterList.List {
						if identifier, ok := binding.Target.(*ast.Identifier); ok {
							param := identifier.Name.String()
							if _, ok = params[param]; ok {
								params[param] = index + 1
							}
						}
					}
					return &JsFunction{
						Func:   fn,
						Params: params,
					}
				}
			}
		}
	}
	return nil
}

func (c *JsContext) Call(fn *JsFunction, forceSync bool, values map[string]interface{}) (goja.Value, error) {
	if fn == nil {
		return nil, nil
	}

	// build args
	var stateSource *map[string]interface{}
	var stateObject goja.Value
	args := []goja.Value{undefined, undefined, undefined, undefined, undefined}
	for n, v := range values {
		if n == "state" {
			// special case with state, we keep ptr to the map
			// and keep object for later export
			forceSync = true
			stateSource = v.(*map[string]interface{})
			stateObject = ToGojaObject(c.Runtime, *stateSource)
			args[fn.Params[n]] = stateObject
		} else {
			args[fn.Params[n]] = ToGojaObject(c.Runtime, v)
		}
	}
	args[0] = undefined

	// lock if sync is needed
	_, withSync := fn.Params["sync"]
	withSync = withSync || forceSync
	if withSync {
		c.mux.Lock()
	}
	defer func() {
		if withSync {
			c.mux.Unlock()
		}
	}()

	// call javascript func
	res, err := fn.Func(goja.Undefined(), args[1:]...)
	if err != nil {
		return nil, err
	}

	// restore state if it was present
	if stateSource != nil {
		*stateSource = stateObject.Export().(map[string]interface{})
	}
	return res, nil
}

func ToMap(obj interface{}) map[string]interface{} {
	if res, ok := obj.(map[string]interface{}); ok {
		return res
	}
	return nil
}

func ToUnstructured(obj interface{}) *unstructured.Unstructured {
	m := ToMap(obj)
	if m == nil {
		return nil
	}
	if _, ok := m["kind"]; !ok {
		m["kind"] = ""
	}

	return &unstructured.Unstructured{Object: m}
}

func ToGojaObject(r *goja.Runtime, value any) goja.Value {
	if value == nil {
		return r.ToValue(value)
	}
	t := reflect.TypeOf(value)
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		v := reflect.ValueOf(value)
		a := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			a[i] = ToGojaObject(r, v.Index(i).Interface())
		}
		return r.NewArray(a...)
	case reflect.Map:
		v := reflect.ValueOf(value)
		o := r.NewObject()
		iter := v.MapRange()
		for iter.Next() {
			//key := iter.Key().Interface().(string)
			//val := iter.Value().Interface()
			//println(key, val)
			_ = o.Set(iter.Key().Interface().(string), ToGojaObject(r, iter.Value().Interface()))
		}
		return o
	}
	return r.ToValue(value)
}
