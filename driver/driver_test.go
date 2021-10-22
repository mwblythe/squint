package driver

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
)

type DriverSuite struct {
	suite.Suite
	db   *sql.DB
	mock sqlmock.Sqlmock
}

func (s *DriverSuite) SetupSuite() {
	name := "squintdriver"

	db, mock, err := sqlmock.NewWithDSN(name, sqlmock.MonitorPingsOption(true))
	if err != nil {
		s.FailNow("cannot create mock DB", err)
		return
	}

	sql.Register(name, Wrap(db.Driver()))

	if db, err = sql.Open(name, name); err != nil {
		s.FailNow("cannot open mock DB", err)
		return
	}

	s.db = db
	s.mock = mock
}

func (s *DriverSuite) TestPing() {
	s.mock.ExpectPing()
	s.Nil(s.db.Ping())
	s.Nil(s.mock.ExpectationsWereMet())
}

func TestDriver(t *testing.T) {
	suite.Run(t, &DriverSuite{})
}
