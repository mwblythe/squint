package driver_test

import (
	"context"
	"database/sql"
	drv "database/sql/driver"
	"fmt"
	"strings"

	"github.com/mwblythe/squint"
	"github.com/mwblythe/squint/driver"
	"github.com/stretchr/testify/suite"
)

type H map[string]interface{}
type Bits []interface{}

var ctx = context.TODO()

func (b Bits) Split() (string, Bits) {
	return b[0].(string), b[1:]
}

type DriverSuite struct {
	suite.Suite
	driver string  // driver name to wrap
	dsn    string  // dsn to open
	db     *sql.DB // wrapped db handle
	count  int64   // insert count
}

func (s *DriverSuite) SetupSuite() {
	var err error

	name := "squint-" + s.driver
	driver.Register(
		name,
		driver.To(s.driver),
	)

	s.db, err = sql.Open(name, s.dsn)
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *DriverSuite) TearDownSuite() {
	s.db.Close()
}

func (s *DriverSuite) TestDriver() {
	s.Run("Ping", func() {
		s.Nil(s.db.PingContext(ctx))
	})

	s.Run("Exec", func() {
		for n := 0; n < 5; n++ {
			s.InsertPerson()
		}
	})

	s.Run("QueryRow", func() {
		s.GetPerson()
	})

	s.Run("Query", func() {
		s.GetPeople()
	})

	s.Run("Prepared", func() {
		s.Prepared()
	})

	s.Run("Transaction", func() {
		s.Transaction()
	})
}

func (s *DriverSuite) TestFuzz() {
	d := s.db.Driver()
	s.NotNil(d)

	s.Panics(func() {
		driver.Register(s.driver + "-foo1")
	})

	if _, ok := d.(drv.DriverContext); ok {
		open, err := d.Open(s.dsn)
		s.Nil(err)
		open.Close()
	}

	s.NotPanics(func() {
		driver.Register(
			s.driver+"-foo2",
			driver.ToDriver(d),
			driver.Builder(squint.NewBuilder()),
		)
	})
}

func (s *DriverSuite) InsertPerson() {
	s.count++

	query, args := Bits{
		"insert into people",
		H{
			"id":   s.count,
			"name": fmt.Sprintf("user-%d", s.count),
		},
	}.Split()

	_, err := s.db.ExecContext(ctx, query, args...)
	s.Nil(err)
}

func (s *DriverSuite) GetPerson() {
	var name string
	query, args := Bits{"select name from people where id =", s.count}.Split()
	row := s.db.QueryRowContext(ctx, query, args...)

	s.Nil(row.Scan(&name))
	s.True(strings.HasPrefix(name, "user-"))
}

func (s *DriverSuite) GetPeople() {
	query, args := Bits{"select id, name from people", "order by id desc"}.Split()
	res, err := s.db.QueryContext(ctx, query, args...)
	if !s.Nil(err) {
		return
	}
	defer res.Close()

	count := s.count

	for res.Next() {
		var id int64
		var name string
		s.Nil(res.Scan(&id, &name))
		s.Equal(count, id)
		s.NotEmpty(name)

		count--
	}

	s.Empty(count)
}

func (s *DriverSuite) Transaction() {
	// start a transaction
	tx, err := s.db.Begin()
	if !s.Nil(err) {
		return
	}

	// delete a row, check that it worked
	res, err := tx.ExecContext(ctx, "delete from people where id =", s.count)
	s.Nil(err)
	affected, err := res.RowsAffected()
	s.Nil(err)
	s.NotEmpty(affected)

	// rollback so it never happened
	s.Nil(tx.Rollback())

	// confirm count is unchanged
	var count int64
	row := s.db.QueryRowContext(ctx, "select count(*) from people")
	s.Nil(row.Scan(&count))
	s.Equal(s.count, count)

	return
}

func (s *DriverSuite) Prepared() {
	queryRow := func(query string) {
		stmt, err := s.db.PrepareContext(ctx, query)
		if !s.Nil(err) {
			return
		}

		var name string
		row := stmt.QueryRowContext(ctx, s.count)
		s.Nil(row.Scan(&name))
		s.NotEmpty(name)
	}

	s.Run("WithPlaceholders", func() {
		queryRow("select name from people where id = ?")
	})

	s.Run("WithoutPlaceholders", func() {
		queryRow("select name from people where id =")
	})
}
