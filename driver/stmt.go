package driver

// TODO: StmtExecContext and StmtQueryContext

import (
	"database/sql"
	"database/sql/driver"
)

// sqStmt is a sql.Stmt wrapper to implement the driver.Stmt interface
type sqStmt struct {
	*sql.Stmt
}

func (s sqStmt) Exec(args []driver.Value) (driver.Result, error) {
	newArgs := valsToIface(args)
	return s.Stmt.Exec(newArgs...)
}

func (s sqStmt) NumInput() int {
	return -1 // no placeholder check
}

func (s sqStmt) Query(args []driver.Value) (driver.Rows, error) {
	newArgs := valsToIface(args)
	r, err := s.Stmt.Query(newArgs...)
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
