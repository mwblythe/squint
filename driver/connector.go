package driver

import (
	"context"
	"database/sql/driver"
)

type connectorWrapper struct {
	driver driver.Driver
	driver.Connector
}

func (c *connectorWrapper) Driver() driver.Driver {
	return c.driver
}

// TODO
func (c *connectorWrapper) Connect(context.Context) (driver.Conn, error) {
	return nil, nil
}
