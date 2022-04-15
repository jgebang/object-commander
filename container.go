package objectcommander

import (
	"fmt"
	"sync"
)

// Identity is a unique name for container resource and bootstrap
type Identity string

// AlreadRegisteredError is an error for reregisteration
type AlreadRegisteredError struct {
	msg string
}

// Error returns the error message
func (a AlreadRegisteredError) Error() string {
	return a.msg
}

// Builder generate the resrouce
type Builder func(c *Container) interface{}

// Definition defines the
type Definition struct {
	Name  Identity
	Build Builder
}

// NewContainer creates a new container
func NewContainer() *Container {
	return &Container{
		builders: make(map[Identity]Builder),
		store:    make(map[Identity]interface{}),
	}
}

// Container is global object accessor and can be used as dependency injection
type Container struct {
	builders map[Identity]Builder
	store    map[Identity]interface{}
	sync.RWMutex
}

// Register adds the definition to builders
func (c *Container) register(def *Definition, overwrite bool) error {
	c.Lock()
	defer c.Unlock()

	if _, exists := c.builders[def.Name]; exists && !overwrite {
		return AlreadRegisteredError{
			msg: fmt.Sprintf("%s was already registered", def.Name),
		}
	}

	c.builders[def.Name] = def.Build

	return nil
}

// Register add the definition to builders
func (c *Container) Register(def *Definition) error {
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
