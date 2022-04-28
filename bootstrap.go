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
func (b *Bootstrap) Boot(procedures []Manager) *Bootstrap {

	for _, p := range procedures {
		err := p.Start(b.container)

		if err == nil {
			b.successful_procedures = append(b.successful_procedures, p)
			continue
		}

		if _, ok := err.(AlreadyRegisteredError); ok {
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
