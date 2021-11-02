package driver

import (
	"database/sql/driver"
	"errors"
)

// TODO: do we need ExecContext + QueryContext?

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
	if s.conn == nil {
		return nil, errors.New("statement already closed")
	}

	// build new query + args
	s.query, args = s.builder.BuildValues(s.query, args)

	// prepare new statement
	stmt, err := s.conn.Prepare(s.query)
	if err != nil {
		return nil, err
	}

	res, err := stmt.Exec(args) //nolint
	stmt.Close()
	return res, err
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.conn == nil {
		return nil, errors.New("statement already closed")
	}

	// build new query + args
	s.query, args = s.builder.BuildValues(s.query, args)

	// prepare new statement
	stmt, err := s.conn.Prepare(s.query)
	if err != nil {
		return nil, err
	}

	return stmt.Query(args) //nolint
}

func (s *stmt) Close() error {
	s.conn = nil
	return nil
}
