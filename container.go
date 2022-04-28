package objectcommander

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Identity is a unique name for container resource and bootstrap
type Identity string

// AlreadRegisteredError is an error for reregisteration
type AlreadyRegisteredError struct {
	msg string
}

// Error returns the error message
func (a AlreadyRegisteredError) Error() string {
	return a.msg
}

// Builder generate the resrouce
type Builder func(c *Container) interface{}

// Definition defines the
type Definition struct {
	Name  Identity
	Type  reflect.Type
	Build Builder
}

// NewContainer creates a new container
func NewContainer() *Container {
	return &Container{
		builders: make(map[Identity]Builder),
		store:    make(map[Identity]interface{}),
		defs:     make(map[reflect.Type]*Definition),
	}
}

// Container is global object accessor and can be used as dependency injection
type Container struct {
	builders map[Identity]Builder
	defs     map[reflect.Type]*Definition
	store    map[Identity]interface{}
	sync.RWMutex
}

// Register adds the definition to builders
func (c *Container) register(def *Definition, overwrite bool) error {
	c.Lock()
	defer c.Unlock()

	if _, exists := c.builders[def.Name]; exists && !overwrite {
		return AlreadyRegisteredError{
			msg: fmt.Sprintf("%s was already registered", def.Name),
		}
	}

	// In order to get the returned arg's type, we have to invoke the build
	// TODO: think a better way to handle this
	ret := def.Build(c)
	ftype := reflect.TypeOf(ret)
	def.Type = ftype
	c.builders[def.Name] = def.Build
	c.defs[ftype] = def
	c.store[def.Name] = ret

	return nil
}

// Register add the definition to builders
func (c *Container) Register(name Identity, build Builder) error {

	def := &Definition{
		Name:  name,
		Build: build,
	}

	return c.register(def, false)
}

// Unregister removes the definition from the builders
func (c *Container) Unregister(name Identity) {
	c.Lock()
	defer c.Unlock()

	delete(c.builders, name)
	delete(c.store, name)
}

// FlushALL clears all registered builders
func (c *Container) FlushALL() {
	for key := range c.builders {
		c.Unregister(key)
	}
}

// Get to get a singleton resource
func (c *Container) Get(name Identity) (interface{}, error) {
	c.RLock()

	if obj, exists := c.store[name]; exists {
		c.RUnlock()
		return obj, nil
	}
	c.RUnlock()

	obj, err := c.Create(name)
	if err != nil {
		return nil, err
	}

	c.Lock()
	defer c.Unlock()
	c.store[name] = obj
	return obj, nil
}

// Create to create a new resource from the builder definition
func (c *Container) Create(name Identity) (interface{}, error) {
	c.RLock()
	defer c.RUnlock()

	builder, exists := c.builders[name]

	if !exists {
		return nil, fmt.Errorf("%s was not registered", name)
	}

	return builder(c), nil
}

// Replace to replace the registered definition
func (c *Container) Replace(def *Definition) error {
	return c.register(def, true)
}

// Invoke to invokes the input function with  args provided from the container
func (c *Container) Invoke(function interface{}) error {
	ftype := reflect.TypeOf(function)
	if ftype == nil {
		return errors.New("can't invoke nil type")
	}

	if ftype.Kind() != reflect.Func {
		return fmt.Errorf("can't invoke non-function: %v(type:%s)", function, ftype)
	}

	// how to collect the args
	args := buildParams(ftype, c)

	ret := invoker(reflect.ValueOf(function), args)
	if len(ret) == 0 {
		return nil
	}

	// check whether there is an error or not.
	lastRet := ret[len(ret)-1]

	err, ok := lastRet.Interface().(error)
	if ok {
		return err
	}

	return nil
}

// grabe the args from the fn and build them from the container
func buildParams(fn reflect.Type, c *Container) []reflect.Value {
	args := []reflect.Value{}
	numArgs := fn.NumIn()

	// currently do not consider to support variadic arguments
	if fn.IsVariadic() {
		numArgs--
	}

	for i := 0; i < numArgs; i++ {
		argType := fn.In(i)
		// try to get the arg from the container with argType?
		arg, _ := c.Get(c.defs[argType].Name)
		what := reflect.ValueOf(arg)
		args = append(args, what)
	}

	return args
}

func invoker(fn reflect.Value, args []reflect.Value) []reflect.Value {
	return fn.Call(args)
}
