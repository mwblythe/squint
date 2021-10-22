package driver

import (
	"context"
	"database/sql/driver"
	"log"

	"github.com/mwblythe/squint"
)

type connCore interface {
	driver.Conn
	driver.Pinger
}

type conn interface {
	connCore
	driver.Execer  // nolint
	driver.Queryer // nolint
}

type connWrapper struct {
	conn
}

func (c *connWrapper) CheckNamedValue(*driver.NamedValue) error {
	// accept all bind types
	return nil
}

func (c *connWrapper) Exec(query string, args []driver.Value) (driver.Result, error) {
	log.Println("Exec")
	return c.conn.Exec(c.build(query, args))
}

func (c *connWrapper) Query(query string, args []driver.Value) (driver.Rows, error) {
	log.Println("Query")
	return c.conn.Query(c.build(query, args))
}

func (c *connWrapper) build(query string, inVals []driver.Value) (string, []driver.Value) {
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

type connContext interface {
	connCore
	driver.ExecerContext
	driver.QueryerContext
	driver.ConnPrepareContext
	driver.ConnBeginTx
	driver.SessionResetter // XXX
	driver.Validator       // XXX
}

type connContextWrapper struct {
	connContext
}

func (c *connContextWrapper) CheckNamedValue(*driver.NamedValue) error {
	// accept all bind types
	return nil
}

func (c *connContextWrapper) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	log.Println("ExecContext")
	return c.connContext.ExecContext(c.build(ctx, query, args))
}

func (c *connContextWrapper) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	log.Println("QueryContext")
	return c.connContext.QueryContext(c.build(ctx, query, args))
}

func (c *connContextWrapper) build(ctx context.Context, query string, inVals []driver.NamedValue) (context.Context, string, []driver.NamedValue) {
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
