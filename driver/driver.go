package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"

	"github.com/mwblythe/squint"
)

// compile-time interface checks
var (
	_ driver.Driver = (*sqDriver)(nil)
)

// Register a sql driver to produce a squint-enabled version.
//
// toDriver is the original sql driver, e.g. "mysql"
//
// Options include:
//
// Name(string) : name to use for the squint driver. (Default "squint-" + toDriver)
// Builder(*Builder) : squint Builder to use. (Default is Builder with no options)
//
func Register(toDriver string, o ...Option) {
	var drv sqDriver
	drv.toDriver = toDriver

	// defaults
	drv.set(
		Name("squint-"+toDriver),
		Builder(squint.NewBuilder()),
	)

	// overrides
	drv.set(o...)

	sql.Register(drv.name, &drv)
}

// sqDriver is the squint proxy driver
type sqDriver struct {
	name     string
	toDriver string
	builder  *builder
	db       sync.Map
}

func (d *sqDriver) Open(dsn string) (c driver.Conn, err error) {
	var db *sql.DB
	if i, ok := d.db.Load(dsn); ok {
		db = i.(*sql.DB)
	} else {
		db, err = sql.Open(d.toDriver, dsn)
		if err != nil {
			return
		}

		d.db.Store(dsn, db)
	}

	// build and return conn
	conn, err := db.Conn(context.Background())
	if err == nil {
		c = newConn(conn, d.builder)
	}

	return
}
