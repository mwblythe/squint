package bridge_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/mwblythe/squint"
	"github.com/mwblythe/squint/bridge"
	"github.com/stretchr/testify/suite"
)

type BridgeSuite struct {
	suite.Suite
	db      *sqlx.DB
	mock    sqlmock.Sqlmock
	builder *squint.Builder
	bDB     *bridge.DB
}

type QueryTestInfo struct {
	inBits   []interface{}
	outSQL   string
	outBinds []interface{}
}

func (s *BridgeSuite) SetupSuite() {
	var err error
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		s.FailNow("cannot create mock DB", err)
		return
	}

	s.mock = mock
	s.db = sqlx.NewDb(db, "sqlmock")
	s.NotNil(s.db)

	s.builder = squint.NewBuilder()
}

func (s *BridgeSuite) TestDB() {
	s.bDB = bridge.NewDB(s.db, s.builder)
	s.NotNil(s.bDB)
	s.testBridge(&s.bDB.Squint)
}

func (s *BridgeSuite) TestTx() {
	s.mock.ExpectBegin()
	tx, err := s.bDB.Beginx()
	s.Nil(err)
	s.NotNil(tx)

	s.testBridge(&tx.Squint)

	s.mock.ExpectCommit()
	err = tx.Commit()
	s.Nil(err)

	s.mock.ExpectBegin().WillReturnError(errors.New("nope"))
	tx, err = s.bDB.Begin()
	s.NotNil(err)
	s.Nil(tx)

	s.mock.ExpectBegin()
	s.bDB.MustBegin()

	s.Nil(s.mock.ExpectationsWereMet())
}

func (s *BridgeSuite) testBridge(b *bridge.Bridge) {
	s.Run("Exec", func() { s.testExec(b) })
	s.Run("QueryRow", func() { s.testQueryRow(b) })
	s.Run("Query", func() { s.testQuery(b) })
}

func (s *BridgeSuite) expectExec(info *QueryTestInfo) {
	s.mock.ExpectExec(info.outSQL).WithArgs(info.getValues()...).WillReturnResult(sqlmock.NewResult(0, 1))
}

func (s *BridgeSuite) testExec(b *bridge.Bridge) {
	info := s.getTestInfo("DELETE FROM users WHERE id =", 10)

	// a basic call to straight Exec
	s.expectExec(info)
	_, err := s.db.Exec(info.outSQL, info.outBinds...)
	s.Nil(err)

	// bridged Exec
	s.expectExec(info)
	_, err = b.Exec(info.inBits...)
	s.Nil(err)

	// bridged Exec with context
	s.expectExec(info)
	_, err = b.ExecContext(context.TODO(), info.inBits...)
	s.Nil(err)

	// bridged MustExec
	s.expectExec(info)
	s.NotPanics(func() { b.MustExec(info.inBits...) })

	// bridged MustExec with context
	s.expectExec(info)
	s.NotPanics(func() { b.MustExecContext(context.TODO(), info.inBits...) })

	s.Nil(s.mock.ExpectationsWereMet())
}

func (s *BridgeSuite) expectQueryRow(info *QueryTestInfo) {
	row := sqlmock.NewRows([]string{"id"})
	row.AddRow(10)
	s.mock.ExpectQuery(info.outSQL).WithArgs(info.getValues()...).WillReturnRows(row)
}

func (s *BridgeSuite) testQueryRow(b *bridge.Bridge) {
	var id int
	name := "Frank"
	info := s.getTestInfo("SELECT id FROM users WHERE name =", &name)

	// straight QueryRow
	s.expectQueryRow(info)
	err := s.db.QueryRow(info.outSQL, info.outBinds...).Scan(&id)
	s.Nil(err)

	// bridged QueryRow
	s.expectQueryRow(info)
	err = b.QueryRow(info.inBits...).Scan(&id)
	s.Nil(err)

	// bridged QueryRow with context
	s.expectQueryRow(info)
	err = b.QueryRowContext(context.TODO(), info.inBits...).Scan(&id)
	s.Nil(err)

	// bridged QueryRowx
	s.expectQueryRow(info)
	err = b.QueryRowx(info.inBits...).Scan(&id)
	s.Nil(err)

	// bridged QueryRowx with context
	s.expectQueryRow(info)
	err = b.QueryRowxContext(context.TODO(), info.inBits...).Scan(&id)
	s.Nil(err)

	// bridged Get
	s.expectQueryRow(info)
	err = b.Get(&id, info.inBits...)
	s.Nil(err)

	// bridged Get with context
	s.expectQueryRow(info)
	err = b.GetContext(context.TODO(), &id, info.inBits...)
	s.Nil(err)

	s.Nil(s.mock.ExpectationsWereMet())
}

func (s *BridgeSuite) expectQuery(info *QueryTestInfo) {
	rows := sqlmock.NewRows([]string{"id", "status"})
	rows.AddRow(10, "active")
	rows.AddRow(10, "retired")
	s.mock.ExpectQuery(info.outSQL).WithArgs(info.getValues()...).WillReturnRows(rows)
}

func (s *BridgeSuite) testQuery(b *bridge.Bridge) {
	info := s.getTestInfo("SELECT id, status FROM users WHERE username IN",
		[]string{"hsimpson", "mferguson"})

	// straight Query
	s.expectQuery(info)
	rows, err := s.db.Query(info.outSQL, info.outBinds...)
	s.Nil(err)
	s.NotNil(rows)
	s.Nil(rows.Close())

	// bridged Query
	s.expectQuery(info)
	rows, err = b.Query(info.inBits)
	s.Nil(err)
	s.NotNil(rows)
	s.Nil(rows.Close())

	// bridged Query with context
	s.expectQuery(info)
	rows, err = b.QueryContext(context.TODO(), info.inBits)
	s.Nil(err)
	s.NotNil(rows)
	s.Nil(rows.Close())

	// bridged Queryx
	s.expectQuery(info)
	xrows, err := b.Queryx(info.inBits)
	s.Nil(err)
	s.NotNil(xrows)
	s.Nil(xrows.Close())

	// bridged Queryx with context
	s.expectQuery(info)
	xrows, err = b.QueryxContext(context.TODO(), info.inBits)
	s.Nil(err)
	s.NotNil(xrows)
	s.Nil(xrows.Close())

	// bridged Select
	var result []struct {
		ID     int
		Status string
	}

	s.expectQuery(info)
	err = b.Select(&result, info.inBits)
	s.Nil(err)
	s.Equal(2, len(result))

	result = result[0:0]
	s.expectQuery(info)
	err = b.SelectContext(context.TODO(), &result, info.inBits)
	s.Nil(err)
	s.Equal(2, len(result))

	s.Nil(s.mock.ExpectationsWereMet())
}

func (s *BridgeSuite) getTestInfo(bits ...interface{}) *QueryTestInfo {
	info := QueryTestInfo{inBits: bits}
	info.outSQL, info.outBinds = s.builder.Build(info.inBits...)
	return &info
}

// return binds as []driver.Value for exepect methods
func (info *QueryTestInfo) getValues() (out []driver.Value) {
	out = make([]driver.Value, len(info.outBinds))
	for i, v := range info.outBinds {
		out[i] = driver.Value(v)
	}
	return
}

func TestBridge(t *testing.T) {
	suite.Run(t, &BridgeSuite{})
}
