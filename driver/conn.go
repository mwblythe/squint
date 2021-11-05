package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
)

// compile-time interface checks
var (
	_ driver.Conn               = (*sqConn)(nil)
	_ driver.Pinger             = (*sqConn)(nil)
	_ driver.ExecerContext      = (*sqConn)(nil)
	_ driver.QueryerContext     = (*sqConn)(nil)
	_ driver.ConnPrepareContext = (*sqConn)(nil)
	_ driver.ConnBeginTx        = (*sqConn)(nil)
	_ driver.NamedValueChecker  = (*sqConn)(nil)
)

// sqConn is a proxy that will pre-process queries with squint Builder
type sqConn struct {
	conn    *sql.Conn
	builder *builder
}

func newConn(c *sql.Conn, b *builder) *sqConn {
	return &sqConn{conn: c, builder: b}
}

func (c *sqConn) CheckNamedValue(*driver.NamedValue) error {
	return nil // accept all bind types
}

func (c *sqConn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

func (c *sqConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	stmt, err := c.conn.PrepareContext(ctx, query)
	return sqStmt{stmt}, err
}

func (c *sqConn) Begin() (driver.Tx, error) {
	return c.conn.BeginTx(context.Background(), nil)
}

func (c *sqConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.conn.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.IsolationLevel(opts.Isolation),
		ReadOnly:  opts.ReadOnly,
	})
}

func (c *sqConn) Close() error {
	return c.conn.Close()
}

func (c *sqConn) Ping(ctx context.Context) error {
	return c.conn.PingContext(ctx)
}

func (c *sqConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	query, binds := c.builder.BuildNamed(query, args)
	return c.conn.ExecContext(ctx, query, binds...)
}

func (c *sqConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	query, binds := c.builder.BuildNamed(query, args)
	r, err := c.conn.QueryContext(ctx, query, binds...)
	return sqRows{r}, err
}
