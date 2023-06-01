package admission

import (
	"context"
	"fmt"
	"github.com/momiji/js-admissions-controller/logs"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"reflect"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
	"github.com/jolestar/go-commons-pool/v2"
)

const (
	JsaInit     = "jsa_init"
	JsaMutate   = "jsa_mutate"
	JsaValidate = "jsa_validate"
	JsaCreated  = "jsa_created"
	JsaUpdated  = "jsa_updated"
	JsaDeleted  = "jsa_deleted"
)

var (
	undefined = goja.Undefined()
)

type JsContext struct {
	mux      *sync.RWMutex
	program  *ast.Program
	compiled *goja.Program
	State    map[string]interface{}
	pool     *pool.ObjectPool
	timeout  int
}

type JsRuntime struct {
	Runtime *goja.Runtime
	Methods map[string]*JsFunction
}

type JsFunction struct {
	Func   goja.Callable
	Params map[string]int
}

func NewJsContext(name string, js string, timeout int) (*JsContext, error) {
	// compile code
	program, err := goja.Parse("", js, parser.WithDisableSourceMaps)
	if err != nil {
		return nil, err
	}

	compiled, err := goja.CompileAST(program, false)
	if err != nil {
		return nil, err
	}

	// create pool factory
	factory := pool.NewPooledObjectFactorySimple(func(ctx context.Context) (interface{}, error) {
		// create runtime
		runtime := goja.New()
		_, err := runtime.RunProgram(compiled)
		if err != nil {
			return nil, err
		}

		// add runtime utils
		err = runtime.Set("log", func(a ...interface{}) {
			logs.Infof("js(%s) %s\n", name, fmt.Sprint(a...))
		})
		if err != nil {
			return nil, err
		}
		err = runtime.Set("logf", func(f string, a ...interface{}) {
			logs.Infof("js(%s) %s\n", name, fmt.Sprintf(f, a...))
		})
		if err != nil {
			return nil, err
		}
		err = runtime.Set("debug", func(a ...interface{}) {
			logs.Debugf("js(%s) %s\n", name, fmt.Sprint(a...))
		})
		if err != nil {
			return nil, err
		}
		err = runtime.Set("debugf", func(f string, a ...interface{}) {
			logs.Debugf("js(%s) %s\n", name, fmt.Sprintf(f, a...))
		})
		if err != nil {
			return nil, err
		}

		// return
		// TODO potential optimization? compute params in JsContext, so only Get(function_name) remains
		return &JsRuntime{
			Runtime: runtime,
			Methods: map[string]*JsFunction{
				JsaInit:     analyseFunction(runtime, program, JsaInit, "state"),
				JsaMutate:   analyseFunction(runtime, program, JsaMutate, "state", "sync", "obj", "op"),
				JsaValidate: analyseFunction(runtime, program, JsaValidate, "state", "sync", "obj", "op"),
				JsaCreated:  analyseFunction(runtime, program, JsaCreated, "state", "sync", "obj"),
				JsaUpdated:  analyseFunction(runtime, program, JsaUpdated, "state", "sync", "obj", "old"),
				JsaDeleted:  analyseFunction(runtime, program, JsaDeleted, "state", "sync", "obj"),
			},
		}, nil
	})

	// create pool
	p := pool.NewObjectPoolWithDefaultConfig(context.Background(), factory)

	// create context
	ctx := JsContext{
		mux:      &sync.RWMutex{},
		program:  program,
		compiled: compiled,
		State:    make(map[string]interface{}),
		pool:     p,
		timeout:  timeout,
	}

	// return
	return &ctx, nil
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

func (c *JsContext) Call(method string, forceSync bool, values map[string]interface{}) (goja.Value, error) {
	background := context.Background()
	object, err := c.pool.BorrowObject(background)
	if err != nil {
		return nil, err
	}
	defer func(pool *pool.ObjectPool, ctx context.Context, object interface{}) {
		_ = pool.ReturnObject(ctx, object)
	}(c.pool, background, object)
	runtime := object.(*JsRuntime)
	return c.call(runtime.Runtime, runtime.Methods[method], forceSync, values)
}

func (c *JsContext) call(runtime *goja.Runtime, fn *JsFunction, forceSync bool, values map[string]interface{}) (goja.Value, error) {
	if fn == nil {
		return nil, nil
	}

	// build args
	var stateSource *map[string]interface{}
	var stateObject goja.Value
	withState := false
	args := []goja.Value{undefined, undefined, undefined, undefined, undefined}
	for n, v := range values {
		if n == "state" {
			withState = true
			// special case with state, we keep ptr to the map
			// and keep object for later export
			stateSource = v.(*map[string]interface{})
			stateObject = ToGojaObject(runtime, *stateSource)
			args[fn.Params[n]] = stateObject
		} else {
			args[fn.Params[n]] = ToGojaObject(runtime, v)
		}
	}
	args[0] = undefined

	// lock RW (if sync || forceSync), R (if state)
	_, withSync := fn.Params["sync"]
	withSync = withSync || forceSync
	if withSync {
		c.mux.Lock()
		defer c.mux.Unlock()
	} else if withState {
		c.mux.RLock()
		defer c.mux.RUnlock()
	}

	// call javascript func
	timer := time.AfterFunc(time.Second*time.Duration(c.timeout), func() {
		runtime.Interrupt(nil)
	})
	res, err := fn.Func(goja.Undefined(), args[1:]...)
	timer.Stop()
	if err != nil {
		return nil, err
	}

	// restore state if it was present
	if withState {
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
