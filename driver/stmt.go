package driver

import (
	"context"
	"database/sql/driver"
	"errors"
)

// only used for prepared queries without placeholders.
// usually because a driver returned ErrSkip elsewhere
type stmt struct {
	conn    driver.Conn
	query   string
	builder *builder
}

func (s *stmt) NumInput() int {
	return -1 // disable placeholder check
}

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), valsToNamed(args))
}

func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (res driver.Result, err error) {
	stmt, newArgs, err := s.prepare(ctx, args)
	if err != nil {
		return
	}

	// execute
	if sc, ok := stmt.(driver.StmtExecContext); ok {
		res, err = sc.ExecContext(ctx, newArgs)
	} else {
		res, err = stmt.Exec(namedToVals(newArgs)) // nolint
	}

	return
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), valsToNamed(args))
}

func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (rows driver.Rows, err error) {
	stmt, newArgs, err := s.prepare(ctx, args)
	if err != nil {
		return
	}

	// query
	if sc, ok := stmt.(driver.StmtQueryContext); ok {
		rows, err = sc.QueryContext(ctx, newArgs)
	} else {
		rows, err = stmt.Query(namedToVals(newArgs)) // nolint
	}

	return
}

func (s *stmt) Close() error {
	s.conn = nil
	return nil
}

// build and prepare a new statement ready for new args
func (s *stmt) prepare(ctx context.Context, args []driver.NamedValue) (stmt driver.Stmt, newArgs []driver.NamedValue, err error) {
	if s.conn == nil {
		err = errors.New("statement already closed")
		return
	}

	// build new newQuery + args
	_, newQuery, newArgs := s.builder.BuildContext(ctx, s.query, args)

	// prepare new statement
	if cc, ok := s.conn.(driver.ConnPrepareContext); ok {
		stmt, err = cc.PrepareContext(ctx, newQuery)
	} else {
		stmt, err = s.conn.Prepare(newQuery)
	}

	return
}
