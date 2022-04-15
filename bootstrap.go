package objectcommander

import (
	"sync"
)

// Manager handles the resource's initialization and release
type Manager interface {
	ID() Identity
	Start(c *Container) error
	Close(c *Container) error
}

// NewBootstrap creates a bootstrap instance
func NewBootstrap(c *Container) *Bootstrap {
	if c == nil {
		c = NewContainer()
	}

	return &Bootstrap{
		container:             c,
		successful_procedures: make([]Manager, 0, 10),
	}
}

// Bootstrap describes a series of procedures to be executed
// before running the main function
type Bootstrap struct {
	container             *Container
	successful_procedures []Manager
	sync.RWMutex
}

func (b *Bootstrap) GetContainer() *Container {
	return b.container
}

// Release releases the resources which collected by the procedures
func (b *Bootstrap) Release() {
	for _, p := range b.successful_procedures {
		p.Close(b.container)
	}

	b.container.FlushALL()
	b.successful_procedures = []Manager{}

}

// Boot executes the series of procedures
// If lazy is true, the definition in the each procedures will only executed when
// calling the container's get method.
//
// That means the resources will only be initialized when accessing the resource stored in the container.
// In normal situation, lazy is set to false because we want the resources are prepared before spawning our
// main application.
func (b *Bootstrap) Boot(procedures []Manager, lazy bool) *Bootstrap {

	for _, p := range procedures {
		err := p.Start(b.container)

		if err == nil {
			if !lazy {
				if _, err := b.container.Get(p.ID()); err != nil {
					panic(err)
				}

			}
			b.successful_procedures = append(b.successful_procedures, p)
			continue
		}

		if _, ok := err.(AlreadRegisteredError); ok {
			continue
		} else {
			b.Release()
			panic(err)
		}

	}

	return b
}

// Run performs the specify function after Booting the procedures
// In addition, this will release the resources after executing the function
func (b *Bootstrap) Run(f func()) {
	if len(b.successful_procedures) != 0 {
		f()
		b.Release()
	}
}
