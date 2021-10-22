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

func (c *connectorWrapper) Connect(ctx context.Context) (driver.Conn, error) {
	orig, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return connContextWrapper{orig.(connContext)}, nil
}
