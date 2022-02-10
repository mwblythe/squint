package driver_test

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mwblythe/squint"
	"github.com/mwblythe/squint/driver"
	"github.com/stretchr/testify/suite"
)

type H map[string]interface{}
type Bits []interface{}

var ctx = context.TODO()

func (b Bits) Split() (context.Context, string, Bits) {
	return ctx, b[0].(string), b[1:]
}

type DriverSuite struct {
	suite.Suite
	mock    sqlmock.Sqlmock
	db      *sql.DB
	builder *squint.Builder
}

func TestDriver(t *testing.T) {
	suite.Run(t, &DriverSuite{})
}

func (s *DriverSuite) SetupSuite() {
	var err error
	dsn := "driver-tests"

	_, mock, err := sqlmock.NewWithDSN(
		dsn,
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual),
		sqlmock.MonitorPingsOption(true),
	)

	if err != nil {
		s.FailNow("cannot create mock DB", err)
		return
	}

	s.mock = mock
	s.builder = squint.NewBuilder()

	driver.Register("sqlmock", driver.Builder(s.builder))

	s.db, err = sql.Open("squint-sqlmock", dsn)
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *DriverSuite) TestPing() {
	s.mock.ExpectPing()
	s.Nil(s.db.PingContext(ctx))
	s.Nil(s.mock.ExpectationsWereMet())
}

func (s *DriverSuite) TestExec() {
	bits := Bits{"delete from junk where id =", 10}
	sql, binds := s.builder.Build(bits...)
	s.mock.ExpectExec(sql).WithArgs(s.getValues(binds)...).WillReturnResult(sqlmock.NewResult(0, 1))
	_, err := s.db.ExecContext(bits.Split())
	s.Nil(err)
	s.Nil(s.mock.ExpectationsWereMet())
}

func (s *DriverSuite) TestQuery() {
	bits := Bits{"select id from junk where id =", 10}
	sql, binds := s.builder.Build(bits...)

	row := sqlmock.NewRows([]string{"id"}).AddRow(10).AddRow(11)
	s.mock.ExpectQuery(sql).WithArgs(s.getValues(binds)...).WillReturnRows(row)
	rows, err := s.db.QueryContext(bits.Split())
	s.Nil(err)

	if s.NotNil(rows) {
		defer rows.Close()
		count := 0

		for rows.Next() {
			count++
			var id int
			s.Nil(rows.Scan(&id))
			s.NotEmpty(id)
		}

		s.NotEmpty(count)
	}

	s.Nil(s.mock.ExpectationsWereMet())
}

func (s *DriverSuite) TestTransaction() {
	s.mock.ExpectBegin()
	s.mock.ExpectCommit()

	tx, err := s.db.Begin()
	s.Nil(err)
	s.Nil(tx.Commit())

	s.Nil(s.mock.ExpectationsWereMet())
}

func (s *DriverSuite) TestPrepare() {
	s.Run("query", func() {
		sql := "select ?"

		s.mock.ExpectPrepare(sql)

		st, err := s.db.Prepare(sql)
		s.Nil(err)
		s.NotNil(st)

		s.mock.ExpectQuery(sql).WillReturnRows(
			sqlmock.NewRows([]string{"id"}).AddRow(10),
		)

		row := st.QueryRowContext(ctx, 10)
		if s.NotNil(row) {
			var id int
			s.Nil(row.Scan(&id))
			s.Equal(10, id)
		}

		s.Nil(s.mock.ExpectationsWereMet())
	})

	s.Run("exec", func() {
		sql := "update junk set foo = 3"

		s.mock.ExpectPrepare(sql)
		st, err := s.db.Prepare(sql)
		s.Nil(err)
		s.NotNil(st)

		s.mock.ExpectExec(sql).WillReturnResult(sqlmock.NewResult(0, 1))
		_, err = st.ExecContext(ctx)
		s.Nil(err)

		s.Nil(s.mock.ExpectationsWereMet())
	})
}

func (s *DriverSuite) getValues(in Bits) (out []sqldriver.Value) {
	out = make([]sqldriver.Value, len(in))
	for i, v := range in {
		out[i] = sqldriver.Value(v)
	}
	return
}

func (s *DriverSuite) TearDownSuite() {
	s.Nil(s.db.Close())
}

func (s *DriverSuite) TestZDeprecated() {
	// Note that these deprecated driver functions are impossible to trigger
	// via the sql package if the newer Context versions exist. However, they
	// are still required to satisfy the driver interfaces. So, we find a way
	// to sort of test them.

	drv := s.db.Driver()
	s.NotNil(drv)

	con, err := drv.Open("driver-tests")
	s.Nil(err)
	s.NotNil(con)

	s.Run("begin", func() {
		s.mock.ExpectBegin()
		tx, err := con.Begin() // nolint
		s.Nil(err)
		s.NotNil(tx)

		s.mock.ExpectRollback()
		s.Nil(tx.Rollback())

		s.Nil(s.mock.ExpectationsWereMet())
	})

	s.Run("prepare/query", func() {
		sql := "select ?"
		s.mock.ExpectPrepare(sql)
		st, err := con.Prepare(sql)
		s.Nil(err)
		s.NotNil(st)

		s.mock.ExpectQuery(sql).WillReturnRows(
			sqlmock.NewRows([]string{"id"}).AddRow(10),
		)

		rows, err := st.Query([]sqldriver.Value{10}) // nolint
		s.Nil(err)
		s.NotNil(rows)

		s.Nil(s.mock.ExpectationsWereMet())
	})

	s.Run("prepare/exec", func() {
		sql := "delete from foo"
		s.mock.ExpectPrepare(sql)
		st, err := con.Prepare(sql)
		s.Nil(err)
		s.NotNil(st)

		s.mock.ExpectExec(sql).WillReturnResult(sqlmock.NewResult(0, 1))
		_, err = st.Exec(nil) // nolint
		s.Nil(err)
		s.Nil(s.mock.ExpectationsWereMet())
	})
}
