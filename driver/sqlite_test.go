//+build sqlite

package driver_test

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
	_ "modernc.org/sqlite"
)

type SQLiteSuite struct {
	DriverSuite
}

func (s *SQLiteSuite) SetupSuite() {
	s.DriverSuite.dsn = "file::memory:"

	s.DriverSuite.SetupSuite()

	_, err := s.db.Exec(`
		CREATE TABLE people (
			id   integer primary key,
			name string  not null
		)
	`)
	if err != nil {
		s.T().Fatal(err)
	}
}

func TestSQLite(t *testing.T) {
	var s SQLiteSuite
	s.driver = "sqlite"
	suite.Run(t, &s)
}

func TestSQLite3(t *testing.T) {
	var s SQLiteSuite
	s.driver = "sqlite3"
	suite.Run(t, &s)
}
