package driver_test

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/suite"
)

type MySQLSuite struct {
	DriverSuite
}

func (s *MySQLSuite) SetupSuite() {
	s.DriverSuite.driver = "mysql"
	s.DriverSuite.dsn = "root@(localhost)/squint"

	s.DriverSuite.SetupSuite()

	_, err := s.db.Exec(`drop table if exists people`)
	if err == nil {
		_, err = s.db.Exec(`
			create table people (
				id   integer primary key,
				name text not null
			)
		`)
	}

	if err != nil {
		s.T().Fatal(err)
	}
}

func TestMySQL(t *testing.T) {
	suite.Run(t, &MySQLSuite{})
}
