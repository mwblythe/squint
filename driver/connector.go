package driver

import (
	"context"
	"database/sql/driver"
	"io"
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
		c = wrapConnContext(orig, w.builder())
	}
	return
}

func (w *connectorWrapper) builder() *builder {
	if cb, ok := w.driver.(*driverContextWrapper); ok {
		return cb.builder
	}

	panic("no builder in connector")
}

func (w *connectorWrapper) Close() error {
	if c, ok := w.Connector.(io.Closer); ok {
		return c.Close()
	}

	return nil
}
