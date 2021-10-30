package driver

import (
	"context"
	"database/sql/driver"
	"log"
)

type connectorWrapper struct {
	driver driver.Driver
	driver.Connector
}

func (c *connectorWrapper) Driver() driver.Driver {
	return c.driver
}

func (c *connectorWrapper) Connect(ctx context.Context) (driver.Conn, error) {
	log.Println("connector.Connect")

	orig, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &connContextWrapper{
		connContext: orig.(connContext),
		builder:     c.builder(),
	}, nil
}

func (c *connectorWrapper) builder() *builder {
	if cb, ok := c.driver.(*driverContextWrapper); ok {
		return cb.builder
	}

	panic("no builder in connector")
}
