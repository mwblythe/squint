package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"log"
	"strings"

	"github.com/mwblythe/squint"
)

// driverWrapper is a wrapper for basic drivers
type driverWrapper struct {
	driver.Driver // the driver being wrapped
}

// Open a connection
func (d *driverWrapper) Open(name string) (driver.Conn, error) {
	log.Println("OPEN")

	orig, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}

	return &connWrapper{orig.(conn)}, nil
}

// wrapped driver implementing DriverContext
type driverContext interface {
	driver.Driver
	driver.DriverContext
}

// driverContextWrapper is a wrapper for driverContextWrapper drivers
type driverContextWrapper struct { // nolint
	driverContext
}

// Open a connection
func (d *driverContextWrapper) Open(name string) (driver.Conn, error) {
	log.Println("Open")

	orig, err := d.driverContext.Open(name)
	if err != nil {
		return nil, err
	}

	return &connContextWrapper{orig.(connContext)}, nil
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

// WrapByName wraps the named sql driver
func WrapByName(name string) (driver.Driver, error) {
	db, err := sql.Open(name, "")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return Wrap(db.Driver()), nil
}

// Wrap the provided Driver
func Wrap(orig driver.Driver) driver.Driver {
	if _, ok := orig.(driver.DriverContext); ok {
		return &driverContextWrapper{orig.(driverContext)}
	}

	return &driverWrapper{orig}
}

// does the query have bind placeholders?
// TODO: detect more than mysql
func hasPlaceholders(query string) bool {
	return strings.Contains(query, "?")
}

func build(query string, inVals []driver.Value) (string, []driver.Value) {
	bits := make([]interface{}, len(inVals)+1)
	bits[0] = query
	for n := range inVals {
		bits[n+1] = inVals[n]
	}

	query, binds := squint.NewBuilder().Build(bits...)

	outVals := make([]driver.Value, len(binds))
	for n := range binds {
		outVals[n] = driver.Value(binds[n])
	}

	return query, outVals
}

func buildNamed(ctx context.Context, query string, inVals []driver.NamedValue) (context.Context, string, []driver.NamedValue) {
	bits := make([]interface{}, len(inVals)+1)
	bits[0] = query
	for n := range inVals {
		bits[n+1] = inVals[n].Value
	}

	query, binds := squint.NewBuilder().Build(bits...)

	outVals := make([]driver.NamedValue, len(binds))
	for n := range binds {
		outVals[n] = driver.NamedValue{
			Ordinal: n + 1,
			Value:   driver.Value(binds[n]),
		}
	}

	return ctx, query, outVals
}
