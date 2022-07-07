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

func (c *Container) bind(b Builder) (*reflect.Value, error) {
	ftype := reflect.TypeOf(b)
	if ftype == nil {
		return nil, errors.New("can't invoke nil type")
	}

	if ftype.Kind() != reflect.Func {
		return nil, fmt.Errorf("can't invoke non-function: %v(type:%s)", b, ftype)
	}

	if ftype.NumOut() != 1 {
		return nil, fmt.Errorf("expect builder function returns one value")
	}

	args := buildParams(ftype, c)
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

// Unregister removes the definition from the builders
func (c *Container) Unregister(name Identity) {
	c.Lock()
	defer c.Unlock()

	delete(c.defs, name)
	delete(c.store, name)
}

// FlushALL clears all registered builders
func (c *Container) FlushALL() {
	for key := range c.defs {
		c.Unregister(key)
	}

	c.typeToIdentity = make(map[reflect.Type][]Identity)
}

// Get to get a singleton resource
func (c *Container) Get(name Identity) (interface{}, error) {
	c.RLock()

	if obj, exists := c.store[name]; exists {
		c.RUnlock()
		return obj, nil
	}
	c.RUnlock()

	c.Lock()
	defer c.Unlock()

	ret, err := c.create(name)
	if err != nil {
		return nil, err
	}

	obj := ret.Interface()

	c.store[name] = obj

	return obj, nil
}

func (c *Container) create(name Identity) (*reflect.Value, error) {
	builder, exists := c.defs[name]

	if !exists {
		return nil, fmt.Errorf("%s was not registered", name)
	}

	// how to perform builder here, I think I need to rely on the invoke
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
	valueType := reflect.TypeOf(value)
	if valueType == nil {
		return errors.New("input value should not be nil")
	}

	et := valueType.Elem()
	if len(ids) > 0 {
		result = c.store[ids[0]]
	} else {
		id := c.typeToIdentity[et][0]
		result = c.store[id]
	}

	reflect.ValueOf(value).Elem().Set(reflect.ValueOf(result))

	return nil
}

// Invoke makes the input function to be called with args provided from the container
func (c *Container) Invoke(function interface{}, ids ...Identity) error {
	ftype := reflect.TypeOf(function)
	if ftype == nil {
		return errors.New("can't invoke nil type")
	}

	if ftype.Kind() != reflect.Func {
		return fmt.Errorf("can't invoke non-function: %v(type:%s)", function, ftype)
	}

	// how to collect the args
	args := buildParams(ftype, c, ids...)

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
func buildParams(fn reflect.Type, c *Container, ids ...Identity) []reflect.Value {
	args := []reflect.Value{}
	numArgs := fn.NumIn()

	// currently do not consider to support variadic arguments
	if fn.IsVariadic() {
		numArgs--
	}

	for i := 0; i < numArgs; i++ {
		argType := fn.In(i)
		var arg interface{}
		// try to get the arg from the container with argType?
		if len(ids) > 0 {
			arg, _ = c.Get(ids[i])
		} else {
			arg, _ = c.Get(c.typeToIdentity[argType][0])
		}

		what := reflect.ValueOf(arg)
		args = append(args, what)
	}

	return args
}

func invoker(fn reflect.Value, args []reflect.Value) []reflect.Value {
	return fn.Call(args)
}
