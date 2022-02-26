package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
)

// compile-time interface checks
var (
	_ driver.Stmt             = (*sqStmt)(nil)
	_ driver.StmtExecContext  = (*sqStmt)(nil)
	_ driver.StmtQueryContext = (*sqStmt)(nil)
)

// sqStmt is a sql.Stmt wrapper to implement the driver.Stmt interface
type sqStmt struct {
	*sql.Stmt
}

func (s sqStmt) Exec(args []driver.Value) (driver.Result, error) {
	newArgs := valsToIface(args)
	return s.Stmt.Exec(newArgs...)
}

func (s sqStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	newArgs := namedToIface(args)
	return s.Stmt.ExecContext(ctx, newArgs...)
}

func (s sqStmt) NumInput() int {
	return -1 // no placeholder check
}

func (s sqStmt) Query(args []driver.Value) (driver.Rows, error) {
	newArgs := valsToIface(args)
	r, err := s.Stmt.Query(newArgs...)

	return sqRows{r}, err
}

func (s sqStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	newArgs := namedToIface(args)
	r, err := s.Stmt.QueryContext(ctx, newArgs...)

	return sqRows{r}, err
}

// convert from []Value to []interface{}
func valsToIface(in []driver.Value) (out []interface{}) {
	out = make([]interface{}, len(in))
	for n := range in {
		out[n] = in[n]
	}

	return
}

// convert from []NamedValue to []interface{}
func namedToIface(in []driver.NamedValue) (out []interface{}) {
	out = make([]interface{}, len(in))
	for n := range in {
		out[n] = in[n].Value
	}

	return
}
