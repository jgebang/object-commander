package objectcommander

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewContainer(t *testing.T) {

	c := NewContainer()

	if c.defs == nil || c.typeToIdentity == nil || c.store == nil {
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
		func() A {
			return A{Name: "yo"}
		})

	c.Invoke(func(a A) {
		if a.Name != "yo" {
			t.Error("invoke with wrong arg")
		}
	})
}

func TestAssign(t *testing.T) {

	c := NewContainer()

	type Person struct{ Age int }

	c.Register(Identity("p"), func() Person {
		return Person{Age: 10}
	})

	var pp Person
	c.Assign(&pp)

	if pp.Age != 10 {
		t.Error("failed to assign")
	}

}

func TestInvokeWithSpecifiedIdentity(t *testing.T) {
	// we register two different instance of struct A

	c := NewContainer()
	type A struct {
		Name string
	}

	c.Register(
		Identity("alice"),
		func() A {
			return A{Name: "I am Alice"}
		})

	c.Register(
		Identity("bob"),
		func() A {
			return A{Name: "I am Bob"}
		})

	c.Invoke(func(a A) {
		if a.Name != "I am Alice" {
			t.Errorf("By default, it should get the first registered instance if there is no identity specified. but we get %s", a.Name)
		}
	})

	c.Invoke(func(a A) {
		if a.Name != "I am Bob" {
			t.Errorf("we should get the indicated instance instead of %s", a.Name)
		}
	}, Identity("bob"))

}

func TestRegister(t *testing.T) {

	c := NewContainer()

	// a function scope variable for test usage. It will be append with string literal
	// when the definition's builder is called.
	var loadedDefs string

	configName := Identity("config")
	configBuild := func() string {
		loadedDefs += "config"
		return "config"
	}

	dbName := Identity("db")
	dbBuild := func() string {
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

	if loadedDefs != "" {
		t.Error("register should be a lazy action")
	}

}

func TestGet(t *testing.T) {

	c := NewContainer()
	id := Identity("config")

	c.Register(id, func() string {
		return "config"
	})

	instance, err := c.Create(id)
	if err != nil {
		t.Error(err)
	}

	if instance.(string) != "config" {
		t.Error("get an unexpected instance")
	}

	// trying to get an unregistered instance
	_, err = c.Create(Identity("non"))
	if err == nil {
		t.Error("we should get an error with non registered instance here")
	}

}

func TestGetWithDependencies(t *testing.T) {

	c := NewContainer()
	configName := Identity("config")
	configBuild := func() string {
		return "config"
	}

	type DB struct{ Name string }

	dbName := Identity("db")
	dbBuild := func(config string) DB {
		if config != "config" {
			t.Errorf("doesn't get the config from container")
		}
		return DB{Name: "sql"}
	}

	c.Register(configName, configBuild)
	c.Register(dbName, dbBuild)

	db, err := c.Get(Identity("db"))
	if err != nil {
		t.Error(err.Error())
	}

	if d, ok := db.(DB); !ok {
		t.Errorf("get error instance: %v", d)
	}

}

func TestGetAndLazyCreate(t *testing.T) {

	c := NewContainer()

	// a function scope variable for test usage. It will be append with string literal
	// when the definition's builder is called.
	var loadedDefs string

	configName := Identity("config")
	configBuild := func() string {
		loadedDefs += "config"
		return "config"
	}

	dbName := Identity("db")

	type DB struct{ Name string }

	dbBuild := func() DB {
		loadedDefs += "db"
		return DB{Name: "sql"}
	}

	c.Register(configName, configBuild)
	c.Register(dbName, dbBuild)

	config, err := c.Get(Identity("config"))
	if err != nil || config.(string) != "config" {
		t.Error("failed to get config from container")
	}

	db, err := c.Get(Identity("db"))
	if err != nil || db.(DB).Name != "sql" {
		t.Error("failed to db config from container")
	}

	if loadedDefs != "configdb" {
		t.Error("config db is not built")
	}

	if len(c.store) != 2 {
		t.Error("definitions are not correctly stored")
	}
}

func TestGetNonRegisteredDef(t *testing.T) {
	c := NewContainer()

	_, err := c.Get(Identity("nop"))
	if !strings.Contains(err.Error(), "was not registered") {
		t.Error("failed to detect nop is not registered")
	}

	_, err = c.GetByType(reflect.TypeOf("hello"))
	if !strings.Contains(err.Error(), "there is no instance") {
		t.Error("failed to detect non registered instance with specified type")
	}
}

func TestMustGet(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected to get a panic")
		}
	}()

	c := NewContainer()
	_ = c.MustGet(Identity("nop"))

}

func TestUnregistered(t *testing.T) {

	c := NewContainer()
	configName := Identity("config")
	configBuild := func(c *Container) interface{} {
		return "config"
	}

	c.Register(configName, configBuild)
	c.Unregister(Identity("config"))

	if len(c.store) != 0 && len(c.defs) != 0 {
		t.Error("failed to unregistered")
	}

}
