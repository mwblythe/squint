package driver

import (
	"database/sql"
	"database/sql/driver"

	"github.com/mwblythe/squint"
)

// Options for Register
type Options struct {
	driver  driver.Driver
	builder *squint.Builder
}

func (o *Options) set(options ...Option) {
	for _, opt := range options {
		opt(o)
	}
}

// Option is a functional option
type Option func(*Options)

// Builder is the squint builder to use
func Builder(b *squint.Builder) Option {
	return func(o *Options) {
		o.builder = b
	}
}

// To is the name of the destination sql driver
func To(name string) Option {
	return func(o *Options) {
		db, err := sql.Open(name, "")
		if err != nil {
			panic(err)
		}
		defer db.Close()
		o.driver = db.Driver()
	}
}

// ToDriver is the destination sql driver
func ToDriver(driver driver.Driver) Option {
	return func(o *Options) {
		o.driver = driver
	}
}
