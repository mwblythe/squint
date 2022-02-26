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
		err := r.Rows.Err()
		if err == nil {
			err = io.EOF
		}

		return err
	}

	into := make([]interface{}, len(dest))
	for n := range into {
		into[n] = new(interface{})
	}

	err := r.Rows.Scan(into...)
	if err == nil {
		for n := range into {
			v := into[n].(*interface{})
			dest[n] = driver.Value(*v)
		}
	}

	return err
}
