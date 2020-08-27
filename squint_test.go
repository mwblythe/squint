package squint

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// conveniences
type binds = []interface{}
type H = map[string]interface{}

type SquintSuite struct {
	suite.Suite
	q     *Builder
	empty binds
}

func (s *SquintSuite) SetupSuite() {
	s.q = NewBuilder()
}

func (s *SquintSuite) TestBasic() {
	s.check("", s.empty, "")
	s.check("foo", s.empty, "foo")
	s.check("hello world", s.empty, "hello", "world")
	s.check("hello world", s.empty, "hello ", "world")
	s.check("hello ?", binds{"world"}, "hello", Bind("world"))

	s.check("SELECT ?", binds{true}, "SELECT", true)
	s.check("SELECT IFNULL( ?, ? )", binds{10, false}, "SELECT IFNULL(", 10, false, ")")
}

func (s *SquintSuite) TestPointer() {
	T := true
	s.check("SELECT ?", binds{true}, "SELECT", &T)

	S := "world"
	s.check("hello ?", binds{"world"}, "hello", &S)

	s.check(
		"SELECT * FROM users WHERE Username = ?",
		binds{"hsimpson"},
		"SELECT * FROM users WHERE",
		&struct {
			Username string
		}{"hsimpson"},
	)

	H := H{"id": 10}
	s.check("WHERE id = ?", binds{10}, "WHERE", &H)

	A := [...]int{1, 10}
	s.check("IN ( ?, ? )", binds{1, 10}, "IN", &A)
}

func (s *SquintSuite) TestStruct() {
	s.check(
		"SELECT * FROM users WHERE Id = ? AND name = ? AND Status IN ( ?, ? )",
		binds{10, "Frank", 2, 4},
		"SELECT * FROM users WHERE",
		struct {
			Id     int
			Name   string `db:"name"`
			Secret bool   `db:"-"`
			wealth float32
			Status []int
		}{10, "Frank", true, 1000.00, []int{2, 4}},
	)
}

func (s *SquintSuite) TestMap() {
	s.check(
		"SELECT * FROM flavors WHERE active = ? AND rating IN ( ?, ? )", binds{true, 4, 5},
		"SELECT * FROM flavors WHERE", H{"active": true, "rating": []int{4, 5}},
	)
}

func (s *SquintSuite) TestArray() {
	s.check("IN ( NULL )", s.empty, "IN", []int{})

	s.check(
		"WHERE id IN ( ?, ?, ? )", binds{10, 20, 30},
		"WHERE id IN", []int{10, 20, 30},
	)

	vals := [2]int{1, 2}
	s.check(
		"WHERE id IN ( ?, ? )", binds{1, 2},
		"WHERE id IN", vals,
	)
}

func (s *SquintSuite) TestInsert() {
	s.check(
		"INSERT INTO junk ( id, size ) VALUES ( ?, ? )", binds{10, "large"},
		"INSERT INTO junk", H{"id": 10, "size": "large"},
	)

	s.check(
		"INSERT INTO junk ( Id, Size ) VALUES ( ?, ? )",
		binds{5, "small"},
		"INSERT INTO junk",
		struct {
			Id     int
			Size   string
			Rating int `db:"omitempty"`
		}{5, "small", 0},
	)
}

func (s *SquintSuite) TestSet() {
	s.check(
		"UPDATE table SET is_active = ?, status = ? WHERE id = ?",
		binds{false, "retired", 10},
		"UPDATE table SET",
		H{"is_active": false, "status": "retired"},
		"WHERE id =", 10,
	)

	s.check(
		"UPDATE table SET is_active = ?, status = ? WHERE id = ?",
		binds{false, "retired", 10},
		"UPDATE table SET", struct {
			Active bool   `db:"is_active"`
			Status string `db:"status"`
		}{false, "retired"},
		"WHERE id =", 10,
	)

	// EmptyUpdate
	s.check(
		"UPDATE table SET status = ?",
		binds{"retired"},
		"UPDATE table SET",
		H{"status": "retired", "name": ""},
	)

	// NilUpdate
	var ptr *bool
	s.check(
		"UPDATE table SET status = ?",
		binds{"retired"},
		"UPDATE table SET",
		H{"status": "retired", "name": ptr},
	)
}

func (s *SquintSuite) TestHasValues() {
	var bar *bool
	s.True(s.q.HasValues(H{"age": 10}))
	s.False(s.q.HasValues(H{"name": ""}))
	s.False(s.q.HasValues(H{"foo": nil}))
	s.False(s.q.HasValues(H{"foo": bar}))
	s.False(s.q.HasValues(H{}))

	var junk struct{}
	var trunk struct {
		Name string
	}

	s.False(s.q.HasValues(junk))
	s.False(s.q.HasValues(trunk))
	trunk.Name = "frank"
	s.True(s.q.HasValues(trunk))
}

func (s *SquintSuite) TestIf() {
	s.check("foo", s.empty, "foo", s.q.If(false, "bar"))
	s.check("foo bar", s.empty, "foo", s.q.If(true, "bar"))

	s.check("SELECT ?", binds{20}, "SELECT", s.q.If(false, 10), 20)
	s.check("SELECT ?, ?", binds{10, 20}, "SELECT", s.q.If(true, 10), 20)
}

func (s *SquintSuite) check(wantSQL string, wantBinds interface{}, args ...interface{}) {
	sql, binds := s.q.Build(args...)
	s.Equal(wantSQL, sql)
	s.Equal(wantBinds, binds)
}

func TestSquint(t *testing.T) {
	suite.Run(t, &SquintSuite{})
}
