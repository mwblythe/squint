package driver

import (
	"database/sql"
	"database/sql/driver"
	"log"
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
func (d *driverWrapper) Open(name string) (driver.Conn, error) {
	log.Println("OPEN")

	orig, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}

	return &connWrapper{
		conn:    orig.(conn),
		builder: d.builder,
	}, nil
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
func (d *driverContextWrapper) Open(name string) (driver.Conn, error) {
	log.Println("Open")

	orig, err := d.driverContext.Open(name)
	if err != nil {
		return nil, err
	}

	return &connContextWrapper{
		connContext: orig.(connContext),
		builder:     d.builder,
	}, nil
}

// OpenConnector opens a connector
func (d *driverContextWrapper) OpenConnector(name string) (driver.Connector, error) {
	log.Println("OpenConnector")

	wrapped, err := d.driverContext.OpenConnector(name)
	if err != nil {
		return nil, err
	}

	return &connectorWrapper{
		driver:    d,
		Connector: wrapped,
	}, nil
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
