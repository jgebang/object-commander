package objectcommander

import (
	"strings"
	"testing"
)

func TestNewContainer(t *testing.T) {

	c := NewContainer()

	if c.builders == nil || c.store == nil {
		t.Error("failed to new a container instance")
	}

}

func TestRegister(t *testing.T) {

	c := NewContainer()

	// a function scope variable for test usage. It will be append with string literal
	// when the definition's builder is called.
	var loadedDefs string

	var configDef Definition = Definition{
		Name: Identity("config"),
		Build: func(c *Container) interface{} {
			loadedDefs += "config"
			return "config"
		},
	}

	var dbDef Definition = Definition{
		Name: Identity("db"),
		Build: func(c *Container) interface{} {
			loadedDefs += "db"
			return "db"
		},
	}

	err := c.Register(&configDef)
	if err != nil {
		t.Error("failed to register config in container")
	}

	err = c.Register(&configDef)
	if _, ok := err.(AlreadRegisteredError); !ok {
		t.Error("failed to detect duplicated registration")
	}

	err = c.Register(&dbDef)
	if err != nil {
		t.Error("failed to register config in container")
	}

	if loadedDefs != "" {
		t.Error("register should be a lazy action")
	}

}

func TestGetAndLazyCreate(t *testing.T) {

	c := NewContainer()

	// a function scope variable for test usage. It will be append with string literal
	// when the definition's builder is called.
	var loadedDefs string

	var configDef Definition = Definition{
		Name: Identity("config"),
		Build: func(c *Container) interface{} {
			loadedDefs += "config"
			return "config"
		},
	}

	var dbDef Definition = Definition{
		Name: Identity("db"),
		Build: func(c *Container) interface{} {
			loadedDefs += "db"
			return "db"
		},
	}

	c.Register(&configDef)
	c.Register(&dbDef)

	config, err := c.Get(Identity("config"))
	if err != nil || config.(string) != "config" {
		t.Error("failed to get config from container")
	}

	if loadedDefs != "config" {
		t.Error("config is not built")
	}

	db, err := c.Get(Identity("db"))
	if err != nil || db.(string) != "db" {
		t.Error("failed to db config from container")
	}

	if loadedDefs != "configdb" {
		t.Error("db is not built")
	}

	if len(c.store) != 2 {
		t.Error("definitions are not correctly stored")
	}
}

func TestGetNonRegisterdDef(t *testing.T) {

	c := NewContainer()

	_, err := c.Get(Identity("nop"))
	if !strings.Contains(err.Error(), "was not registered") {
		t.Error("failed to detect nop is not registered")
	}
}

func TestUnregisterd(t *testing.T) {

	c := NewContainer()
	var configDef Definition = Definition{
		Name: Identity("config"),
		Build: func(c *Container) interface{} {
			return "config"
		},
	}

	c.Register(&configDef)
	c.Unregister(Identity("config"))

	if len(c.store) != 0 && len(c.builders) != 0 {
		t.Error("failed to unregisterd")
	}

}
