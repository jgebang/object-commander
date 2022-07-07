package objectcommander

import (
	"fmt"
	"strings"
	"testing"
)

// a global variable for testing usage
var loadtracker string

var dbManager = Manager{
	ID: Identity("db"),
	Start: func() string {
		loadtracker += "db"
		return "db"
	},
	Close: func(c *Container) error {
		db, err := c.Get(Identity("db"))
		if err != nil {
			return err
		}

		loadtracker = strings.Replace(loadtracker, "db", "", 1) // represent db resource is released
		fmt.Printf("%s is closed\n", db.(string))
		return nil

	},
}

var logManager = Manager{
	ID: Identity("log"),
	Start: func() string {
		loadtracker += "log"
		return "log"
	},
	Close: func(c *Container) error {
		log, err := c.Get(Identity("log"))
		if err != nil {
			return err
		}

		loadtracker = strings.Replace(loadtracker, "log", "", 1) // represent log resource is released
		fmt.Printf("%s is closed\n", log.(string))
		return nil

	},
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
		dbManager,
		logManager,
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
