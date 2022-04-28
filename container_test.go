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

func TestInvoke(t *testing.T) {

	c := NewContainer()

	type A struct {
		Name string
	}

	c.Register(
		Identity("song"),
		func(c *Container) interface{} {
			return A{Name: "yo"}
		})

	c.Invoke(func(a A) {
		if a.Name != "yo" {
			t.Error("invoke with wrong arg")
		}
	})
}

func TestRegister(t *testing.T) {

	c := NewContainer()

	// a function scope variable for test usage. It will be append with string literal
	// when the definition's builder is called.
	var loadedDefs string

	configName := Identity("config")
	configBuild := func(c *Container) interface{} {
		loadedDefs += "config"
		return "config"
	}

	dbName := Identity("db")
	dbBuild := func(c *Container) interface{} {
		loadedDefs += "db"
		return "db"
	}

	err := c.Register(configName, configBuild)
	if err != nil {
		t.Error("failed to register config in container")
	}

	err = c.Register(configName, configBuild)
	if _, ok := err.(AlreadyRegisteredError); !ok {
		t.Error("failed to detect duplicated registration")
	}

	err = c.Register(dbName, dbBuild)
	if err != nil {
		t.Error("failed to register config in container")
	}

	// if loadedDefs != "" {
	// 	t.Error("register should be a lazy action")
	// }

}

func TestGetAndLazyCreate(t *testing.T) {

	c := NewContainer()

	// a function scope variable for test usage. It will be append with string literal
	// when the definition's builder is called.
	var loadedDefs string

	configName := Identity("config")
	configBuild := func(c *Container) interface{} {
		loadedDefs += "config"
		return "config"
	}

	dbName := Identity("db")
	dbBuild := func(c *Container) interface{} {
		loadedDefs += "db"
		return "db"
	}

	c.Register(configName, configBuild)
	c.Register(dbName, dbBuild)

	config, err := c.Get(Identity("config"))
	if err != nil || config.(string) != "config" {
		t.Error("failed to get config from container")
	}

	db, err := c.Get(Identity("db"))
	if err != nil || db.(string) != "db" {
		t.Error("failed to db config from container")
	}

	if loadedDefs != "configdb" {
		t.Error("config db is not built")
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
	configName := Identity("config")
	configBuild := func(c *Container) interface{} {
		return "config"
	}

	c.Register(configName, configBuild)
	c.Unregister(Identity("config"))

	if len(c.store) != 0 && len(c.builders) != 0 {
		t.Error("failed to unregisterd")
	}

}
