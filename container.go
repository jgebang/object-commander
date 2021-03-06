package objectcommander

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Identity is a unique name for container resource and bootstrap
type Identity string

// AlreadyRegisteredError is an error for reregisteration
type AlreadyRegisteredError struct {
	msg string
}

// Error returns the error message
func (a AlreadyRegisteredError) Error() string {
	return a.msg
}

// Builder is a function to generate the resrouce
type Builder interface{}

// NewContainer creates a new container
func NewContainer() *Container {
	return &Container{
		store:          make(map[Identity]interface{}),
		defs:           make(map[Identity]Builder),
		typeToIdentity: make(map[reflect.Type][]Identity),
	}
}

// Container is global object accessor and can be used as dependency injection
type Container struct {
	defs           map[Identity]Builder
	typeToIdentity map[reflect.Type][]Identity
	store          map[Identity]interface{}
	sync.RWMutex
}

// checkBuilderSignature is an helper function to check the interface if matched the format
// for generating the resource.
func checkBuilderSignature(ftype reflect.Type) error {

	if ftype == nil {
		return errors.New("can't invoke nil type")
	}

	if ftype.Kind() != reflect.Func {
		return fmt.Errorf("can't invoke non-function: %s", ftype)
	}

	if ftype.NumOut() != 1 {
		return fmt.Errorf("expect builder function returns one value")
	}

	return nil
}

func (c *Container) bind(b Builder) (*reflect.Value, error) {
	ftype := reflect.TypeOf(b)

	if err := checkBuilderSignature(ftype); err != nil {
		return nil, err
	}

	args, err := buildParams(ftype, c)
	if err != nil {
		return nil, err
	}
	ret := invoker(reflect.ValueOf(b), args)
	return &ret[0], nil
}

// Register add the definition to builders
func (c *Container) Register(name Identity, build Builder) error {

	c.RLock()
	if _, exists := c.defs[name]; exists {

		c.RUnlock()
		return AlreadyRegisteredError{
			msg: fmt.Sprintf("%s was already registered", name),
		}
	}
	c.RUnlock()

	c.Lock()
	fn := reflect.TypeOf((build))
	retType := fn.Out(0)

	c.defs[name] = build
	c.typeToIdentity[retType] = append(
		c.typeToIdentity[retType],
		name)

	c.Unlock()

	return nil
}

func pop(source []Identity, target Identity) []Identity {
	var index = 0

	for i, value := range source {
		if value == target {
			index = i
			break
		}
	}

	return append(source[:index], source[index+1:]...)
}

// Unregister removes the definition from the builders
func (c *Container) Unregister(name Identity) {
	c.Lock()
	defer c.Unlock()

	builder := c.defs[name]
	fn := reflect.TypeOf(builder)
	retType := fn.Out(0)

	pop(c.typeToIdentity[retType], name)
	delete(c.defs, name)
	delete(c.store, name)
}

// FlushALL clears all registered builders
func (c *Container) FlushALL() {
	c.defs = make(map[Identity]Builder)
	c.store = make(map[Identity]interface{})
	c.typeToIdentity = make(map[reflect.Type][]Identity)
}

// GetByType works like get but instead of getting instance by the identity,
// this will allow you give a type and automatically induct the identity
// for you
func (c *Container) GetByType(t reflect.Type) (interface{}, error) {

	if len(c.typeToIdentity[t]) == 0 {
		return nil, fmt.Errorf("there is no instance registered with type: %s", t)
	}

	id := c.typeToIdentity[t][0]

	return c.Get(id)
}

// MustGet is an helper for Get without returning error. It will
// panic once if there is an error happens so pleasure ensure you
// are knowing the instance is actually registered.
func (c *Container) MustGet(name Identity) interface{} {
	result, err := c.Get(name)
	if err != nil {
		panic(err)
	}

	return result
}

// Get to get a singleton resource
func (c *Container) Get(name Identity) (interface{}, error) {
	c.RLock()

	if obj, exists := c.store[name]; exists {
		c.RUnlock()
		return obj, nil
	}
	c.RUnlock()

	ret, err := c.create(name)
	if err != nil {
		return nil, err
	}

	obj := ret.Interface()

	c.Lock()
	defer c.Unlock()
	c.store[name] = obj

	return obj, nil
}

func (c *Container) create(name Identity) (*reflect.Value, error) {
	builder, exists := c.defs[name]

	if !exists {
		return nil, fmt.Errorf("%s was not registered", name)
	}

	ret, err := c.bind(builder)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// Create to create a new resource from the builder definition
func (c *Container) Create(name Identity) (interface{}, error) {
	c.Lock()
	defer c.Unlock()

	ret, err := c.create(name)
	if err != nil {
		return nil, err
	}

	return ret.Interface(), nil
}

// Assign is similar to Get instead returning an interface.
// this will assign the value taken from the container to the arg and
// you can specify the identity to indicate which instance you want to
// be assigned.
func (c *Container) Assign(value interface{}, ids ...Identity) error {
	var result interface{}
	var err error

	valueType := reflect.TypeOf(value)
	if valueType == nil {
		return errors.New("input value should not be nil")
	}

	et := valueType.Elem()
	if len(ids) > 0 {
		if result, err = c.Get(ids[0]); err != nil {
			return err
		}
	} else {
		if result, err = c.GetByType(et); err != nil {
			return err
		}
	}

	reflect.ValueOf(value).Elem().Set(reflect.ValueOf(result))

	return nil
}

func maybeError(ret []reflect.Value) error {
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

// checkCallee is an helper to check the basic required function signature
func checkCallee(ftype reflect.Type) error {
	if ftype == nil {
		return errors.New("can't invoke nil type")
	}

	if ftype.Kind() != reflect.Func {
		return fmt.Errorf("can't invoke non-function: %s", ftype)
	}

	return nil
}

// Invoke makes the input function to be called with args provided from the container
func (c *Container) Invoke(function interface{}, ids ...Identity) error {
	ftype := reflect.TypeOf(function)

	if err := checkCallee(ftype); err != nil {
		return err
	}

	// how to collect the args
	args, err := buildParams(ftype, c, ids...)
	if err != nil {
		return err
	}

	return maybeError(invoker(reflect.ValueOf(function), args))
}

// grabe the args from the fn and build them from the container
func buildParams(fn reflect.Type, c *Container, ids ...Identity) ([]reflect.Value, error) {
	args := []reflect.Value{}
	var arg interface{}
	var err error
	numArgs := fn.NumIn()

	// currently do not consider to support variadic arguments
	if fn.IsVariadic() {
		numArgs--
	}

	for i := 0; i < numArgs; i++ {
		argType := fn.In(i)
		// try to get the arg from the container with argType?
		if len(ids) > 0 {
			if arg, err = c.Get(ids[i]); err != nil {
				return nil, err
			}
		} else {

			if arg, err = c.GetByType(argType); err != nil {
				return nil, err
			}

		}

		what := reflect.ValueOf(arg)
		args = append(args, what)
	}

	return args, nil
}

func invoker(fn reflect.Value, args []reflect.Value) []reflect.Value {
	return fn.Call(args)
}
