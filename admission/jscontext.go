package admission

import (
	"fmt"
	"github.com/momiji/js-admissions-controller/logs"
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
	args := []goja.Value{undefined, undefined, undefined, undefined, undefined}
	for n, v := range values {
		args[fn.Params[n]] = c.Runtime.ToValue(v)
	}
	args[0] = undefined

	// lock if sync is needed
	_, sync := fn.Params["sync"]
	sync = sync || forceSync
	if sync {
		c.mux.Lock()
		defer c.mux.Unlock()
	}

	// call javascript func
	res, err := fn.Func(goja.Undefined(), args[1:]...)
	if err != nil {
		return nil, err
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
