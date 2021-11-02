package driver

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/suite"
)

type InternalSuite struct {
	suite.Suite
}

func TestInternal(t *testing.T) {
	suite.Run(t, &InternalSuite{})
}

func (s *InternalSuite) TestVals() {
	vals := []driver.Value{"hello", "world", 42}
	named := valsToNamed(vals)

	s.Run("valsToNamed", func() {
		if s.Len(named, len(vals)) {
			for n := range named {
				s.Equal(n+1, named[n].Ordinal)
				s.Equal(vals[n], named[n].Value)
			}
		}
	})

	s.Run("namedToVals", func() {
		s.Equal(vals, namedToVals(named))
	})
}
