// Package driver enables the use of squint Build() syntax in standard sql/database query functions.
package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"time"

	"github.com/mwblythe/squint"
)

// compile-time interface checks
var (
	_ driver.Driver = (*sqDriver)(nil)
)

var ErrNoDB = errors.New("no database connection")

// Register a sql driver to produce a squint-enabled version.
//
// toDriver is the original sql driver, e.g. "mysql"
//
// Options include:
//
// Name(string) : name to use for the squint driver. (Default "squint-" + toDriver)
// Builder(*Builder) : squint Builder to use. (Default is Builder with no options)
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
	db, err := d.pool(dsn)
	if err != nil {
		return
	}

	if db == nil {
		err = ErrNoDB
		return
	}

	// build conn
	conn, err := db.Conn(context.Background())
	if err == nil && conn != nil {
		c = newConn(conn, d.builder, dsn)
	}

	return
}

func (d *sqDriver) pool(dsn string) (db *sql.DB, err error) {
	if v, ok := d.db.Load(dsn); ok {
		db = v.(*sql.DB)
	} else {
		db, err = sql.Open(d.toDriver, dsn)
		if err == nil {
			d.db.Store(dsn, db)
		}
	}

	return
}

func SetConnMaxIdleTime(outerDB *sql.DB, d time.Duration) {
	inDB, err := innerDB(outerDB)
	if err == nil && inDB != nil {
		outerDB.SetConnMaxIdleTime(d)
		inDB.SetConnMaxIdleTime(d)
	}
}

func SetConnMaxLifetime(outerDB *sql.DB, d time.Duration) {
	inDB, err := innerDB(outerDB)
	if err == nil && inDB != nil {
		outerDB.SetConnMaxLifetime(d)
		inDB.SetConnMaxLifetime(d)
	}
}

func innerDB(outerDB *sql.DB) (*sql.DB, error) {
	driver, ok := outerDB.Driver().(*sqDriver)
	if !ok {
		return nil, errors.New("not a squint driver")
	}

	conn, err := outerDB.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var outDB *sql.DB
	err = conn.Raw(func(driverConn interface{}) error {
		sqConn, ok := driverConn.(*sqConn)
		if !ok {
			return errors.New("not a squint connection")
		}

		outDB, err = driver.pool(sqConn.dsn)
		return err
	})

	return outDB, err
}
