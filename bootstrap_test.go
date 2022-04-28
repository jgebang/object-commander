package objectcommander

import (
	"fmt"
	"strings"
	"testing"
)

// a global variable for testing usage
var loadtracker string

type dbManager struct {
}

func (dbm *dbManager) ID() Identity {
	return Identity("db")
}

func (dbm *dbManager) Start(c *Container) error {

	c.Register(dbm.ID(), func(c *Container) interface{} {
		loadtracker += "db"
		return "db"
	})

	return nil
}

func (dbm *dbManager) Close(c *Container) error {

	db, err := c.Get(dbm.ID())
	if err != nil {
		return err
	}

	loadtracker = strings.Replace(loadtracker, "db", "", 1) // represent db resource is released
	fmt.Printf("%s is closed\n", db.(string))
	return nil
}

type logManager struct{}

func (l *logManager) ID() Identity {
	return Identity("log")
}

func (l *logManager) Start(c *Container) error {
	c.Register(l.ID(), func(c *Container) interface{} {
		loadtracker += "log"
		return "log"
	})

	return nil
}

func (l *logManager) Close(c *Container) error {

	log, err := c.Get(l.ID())
	if err != nil {
		return err
	}

	loadtracker = strings.Replace(loadtracker, "log", "", 1) // represent log resource is released
	fmt.Printf("%s is closed\n", log.(string))
	return nil
}

func TestNewBootstrap(t *testing.T) {

	b := NewBootstrap(nil)

	if b.successful_procedures == nil {
		t.Error("failed to new a boostrap instance")
	}

}

func TestBoot(t *testing.T) {

	b := NewBootstrap(nil)
	steps := []Manager{
		&dbManager{},
		&logManager{},
	}

	b.Boot(steps).Run(func() {

		if loadtracker != "dblog" {
			t.Error("steps were not executed")
		}

	})

	if loadtracker != "" {
		t.Error("resources were not released")
	}
}
