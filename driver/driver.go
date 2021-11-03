package driver

import (
	"database/sql"
	"database/sql/driver"
	"strings"

	"github.com/mwblythe/squint"
)

// Register a squint wrapped sql driver
func Register(name string, o ...Option) {
	var opt Options
	opt.set(o...)

	if opt.driver == nil {
		panic("squint/driver: must specify To or ToDriver")
	}

	if opt.builder == nil {
		opt.builder = squint.NewBuilder()
	}

	sql.Register(name, wrap(opt.driver, opt.builder))
}

// driverWrapper is a wrapper for basic drivers
type driverWrapper struct {
	driver.Driver // the driver being wrapped
	builder       *builder
}

// Open a connection
func (d *driverWrapper) Open(name string) (c driver.Conn, err error) {
	orig, err := d.Driver.Open(name)
	if err == nil {
		c = &connWrapper{
			conn:    orig.(conn),
			builder: d.builder,
		}
	}
	return
}

// wrapped driver implementing DriverContext
type driverContext interface {
	driver.Driver
	driver.DriverContext
}

// driverContextWrapper is a wrapper for driverContextWrapper drivers
type driverContextWrapper struct { // nolint
	driverContext
	builder *builder
}

// Open a connection
func (d *driverContextWrapper) Open(name string) (c driver.Conn, err error) {
	orig, err := d.driverContext.Open(name)
	if err == nil {
		c = wrapConnContext(orig, d.builder)
	}

	return
}

// OpenConnector opens a connector
func (d *driverContextWrapper) OpenConnector(name string) (c driver.Connector, err error) {
	orig, err := d.driverContext.OpenConnector(name)
	if err == nil {
		c = &connectorWrapper{
			driver:    d,
			Connector: orig,
		}
	}
	return
}

// wrap the provided Driver
func wrap(orig driver.Driver, build *squint.Builder) driver.Driver {
	if _, ok := orig.(driver.DriverContext); ok {
		return &driverContextWrapper{
			driverContext: orig.(driverContext),
			builder:       newBuilder(build),
		}
	}

	return &driverWrapper{
		Driver:  orig,
		builder: newBuilder(build),
	}
}

// does the query have bind placeholders?
// TODO: detect more than mysql
func hasPlaceholders(query string) bool {
	return strings.Contains(query, "?")
}

// convert from []Value to []NamedValue
func valsToNamed(vals []driver.Value) (named []driver.NamedValue) {
	named = make([]driver.NamedValue, len(vals))

	for n := range vals {
		named[n] = driver.NamedValue{
			Ordinal: n + 1,
			Value:   vals[n],
		}
	}

	return
}

// convert from []NamedValue to []Value
func namedToVals(named []driver.NamedValue) (vals []driver.Value) {
	vals = make([]driver.Value, len(named))

	for n := range named {
		vals[n] = named[n].Value
	}

	return
}
