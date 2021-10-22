package driver

import (
	"database/sql"
	"database/sql/driver"
	"log"
)

// driverWrapper is a wrapper for basic drivers
type driverWrapper struct {
	driver.Driver // the driver being wrapped
}

// Open a connection
func (d driverWrapper) Open(name string) (driver.Conn, error) {
	log.Println("OPEN")

	orig, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}

	return connWrapper{orig.(conn)}, nil
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
