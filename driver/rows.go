package driver

import (
	"database/sql"
	"database/sql/driver"
	"io"
)

// compile-time interface checks
var (
	_ driver.Rows = (*sqRows)(nil)
)

// sqRows is a sql.Rows wrapper to implement the driver.Rows interface
type sqRows struct {
	*sql.Rows
}

func (r sqRows) Columns() []string {
	c, _ := r.Rows.Columns()
	return c
}

func (r sqRows) Next(dest []driver.Value) error {
	if !r.Rows.Next() {
		if err := r.Rows.Err(); err != nil {
			return err
		}
		return io.EOF
	}

	into := make([]interface{}, len(dest))
	for n := range into {
		into[n] = new(interface{})
	}

	if err := r.Rows.Scan(into...); err != nil {
		return err
	}

	for n := range into {
		v := into[n].(*interface{})
		dest[n] = driver.Value(*v)
	}

	return nil
}
