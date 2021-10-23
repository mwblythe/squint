package driver_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/mwblythe/squint/driver"
	"github.com/stretchr/testify/suite"
)

type H map[string]interface{}
type Bits []interface{}

func (b Bits) Split() (string, Bits) {
	return b[0].(string), b[1:]
}

type DriverSuite struct {
	suite.Suite
	driver string
	dsn    string
	db     *sql.DB
	count  int64
}

/*
func DriverTest(t *testing.T, driver string, dsn string) {
	suite.Run(t, &DriverSuite{
		driver: driver,
		dsn:    dsn,
	})
}
*/

func (s *DriverSuite) SetupSuite() {
	driver, err := driver.WrapByName(s.driver)
	if err != nil {
		s.T().Fatal(err)
	}

	name := "squint-" + s.driver
	sql.Register(name, driver)

	s.db, err = sql.Open(name, s.dsn)
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *DriverSuite) TearDownSuite() {
	s.db.Close()
}

func (s *DriverSuite) Test1Ping() {
	s.Nil(s.db.Ping())
	s.Nil(s.db.PingContext(context.TODO()))
}

func (s *DriverSuite) Test2Exec() {
	query, args := s.Insert().Split()
	_, err := s.db.Exec(query, args...)
	s.Nil(err)
}

func (s *DriverSuite) Test2ExecContext() {
	query, args := s.Insert().Split()
	_, err := s.db.ExecContext(context.TODO(), query, args...)
	s.Nil(err)
}

func (s *DriverSuite) Test3QueryRow() {
	var name string
	query, args := s.Get().Split()
	row := s.db.QueryRow(query, args...)
	s.Nil(row.Scan(&name))
	s.True(strings.HasPrefix(name, "user-"))
}

func (s *DriverSuite) Test3QueryRowContext() {
	var name string
	query, args := s.Get().Split()
	row := s.db.QueryRowContext(context.TODO(), query, args...)
	s.Nil(row.Scan(&name))
	s.True(strings.HasPrefix(name, "user-"))
}

func (s *DriverSuite) Insert() Bits {
	s.count++

	return Bits{
		"insert into people",
		H{
			"id":   s.count,
			"name": fmt.Sprintf("user-%d", time.Now().Unix()),
		},
	}
}

func (s *DriverSuite) Get() Bits {
	return Bits{"select name from people where id =", s.count}
}
