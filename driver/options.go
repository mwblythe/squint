package driver

import (
	"github.com/mwblythe/squint"
)

func (d *sqDriver) set(options ...Option) {
	for _, opt := range options {
		opt(d)
	}
}

// Option is a functional option
type Option func(*sqDriver)

// Builder is the squint builder to use
func Builder(b *squint.Builder) Option {
	return func(d *sqDriver) {
		d.builder = newBuilder(b)
	}
}

// Name of the new squint driver
func Name(name string) Option {
	return func(d *sqDriver) {
		d.name = name
	}
}
