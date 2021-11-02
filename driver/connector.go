package driver

import (
	"context"
	"database/sql/driver"
)

type connectorWrapper struct {
	driver driver.Driver
	driver.Connector
}

func (w *connectorWrapper) Driver() driver.Driver {
	return w.driver
}

func (w *connectorWrapper) Connect(ctx context.Context) (c driver.Conn, err error) {
	orig, err := w.Connector.Connect(ctx)
	if err == nil {
		c = &connContextWrapper{
			connContext: orig.(connContext),
			builder:     w.builder(),
		}
	}
	return
}

func (w *connectorWrapper) builder() *builder {
	if cb, ok := w.driver.(*driverContextWrapper); ok {
		return cb.builder
	}

	panic("no builder in connector")
}
