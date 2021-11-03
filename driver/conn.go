package driver

import (
	"context"
	"database/sql/driver"
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
	builder *builder
}

func (c *connWrapper) CheckNamedValue(*driver.NamedValue) error {
	// accept all bind types
	return nil
}

func (c *connWrapper) Exec(query string, args []driver.Value) (driver.Result, error) {
	return c.conn.Exec(c.builder.BuildValues(query, args))
}

func (c *connWrapper) Query(query string, args []driver.Value) (driver.Rows, error) {
	return c.conn.Query(c.builder.BuildValues(query, args))
}

func (c *connWrapper) Prepare(query string) (driver.Stmt, error) {
	if hasPlaceholders(query) {
		return c.conn.Prepare(query)
	}

	return &stmt{
		conn:    c.conn,
		query:   query,
		builder: c.builder,
	}, nil
}

type connContext interface {
	connCore
	driver.ExecerContext
	driver.QueryerContext
	driver.ConnPrepareContext
	driver.ConnBeginTx
}

type connResetter interface {
	driver.SessionResetter
	driver.Validator
}

type connContextWrapper struct {
	connContext
	builder *builder
}

type connResetterWrapper struct {
	*connContextWrapper
	connResetter
}

// wrap a context-aware Conn, retaining optional interfaces if supported
func wrapConnContext(orig driver.Conn, builder *builder) driver.Conn {
	cc := connContextWrapper{
		builder:     builder,
		connContext: orig.(connContext),
	}

	if cr, ok := orig.(connResetter); ok {
		return &connResetterWrapper{
			connContextWrapper: &cc,
			connResetter:       cr,
		}
	}

	return &cc
}

func (c *connContextWrapper) CheckNamedValue(*driver.NamedValue) error {
	// accept all bind types
	return nil
}

func (c *connContextWrapper) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if hasPlaceholders(query) {
		return c.connContext.PrepareContext(ctx, query)
	}

	return &stmt{
		conn:    c.connContext,
		query:   query,
		builder: c.builder,
	}, nil
}

func (c *connContextWrapper) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.connContext.ExecContext(c.builder.BuildContext(ctx, query, args))
}

func (c *connContextWrapper) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.connContext.QueryContext(c.builder.BuildContext(ctx, query, args))
}
