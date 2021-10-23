package driver_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	_ "modernc.org/sqlite"
)

type SQLiteSuite struct {
	DriverSuite
}

func (s *SQLiteSuite) SetupSuite() {
	s.DriverSuite.driver = "sqlite"
	s.DriverSuite.dsn = "file::memory:"

	s.DriverSuite.SetupSuite()

	_, err := s.db.Exec(`
		CREATE TABLE people (
			id integer primary key,
			name string not null
		)
	`)
	if err != nil {
		s.T().Fatal(err)
	}
}

func TestSQLite(t *testing.T) {
	suite.Run(t, &SQLiteSuite{})
}
