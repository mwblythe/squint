package squint

import (
	"bytes"
	"log"
	"reflect"
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

func TestSquint(t *testing.T) {
	suite.Run(t, &SquintSuite{})
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

	var N *int
	s.check("a = ?", binds{nil}, "a =", N)
}

func (s *SquintSuite) TestStruct() {
	s.check(
		"SELECT * FROM users WHERE ID = ? AND name = ? AND Status IN ( ?, ? )",
		binds{10, "Frank", 2, 4},
		"SELECT * FROM users WHERE",
		struct {
			ID     int
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
		"INSERT IGNORE INTO junk ( id, size ) VALUES ( ?, ? )", binds{10, "large"},
		"INSERT IGNORE INTO junk", H{"id": 10, "size": "large"},
	)

	s.check(
		"INSERT INTO junk SET id = ?, size = ?", binds{10, "large"},
		"INSERT INTO junk SET", H{"id": 10, "size": "large"},
	)

	s.check(
		"INSERT INTO junk ( ID, Size ) VALUES ( ?, ? )",
		binds{5, "small"},
		"INSERT INTO junk",
		struct {
			ID     int
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
	b1 := NewBuilder()
	b2 := NewBuilder(EmptyValues(true), NilValues(true))

	// test maps
	var bar *bool
	s.True(b1.HasValues(H{"age": 10}))
	s.False(b1.HasValues(H{"name": ""}))
	s.False(b1.HasValues(H{"foo": nil}))
	s.False(b1.HasValues(H{"foo": bar}))
	s.False(b1.HasValues(H{}))

	s.True(b2.HasValues(H{"name": ""}))
	s.True(b2.HasValues(H{"foo": nil}))
	s.True(b2.HasValues(H{"foo": bar}))

	// test structs
	var junk struct{}
	type trunk struct {
		Name string
	}

	s.False(b1.HasValues(junk))
	s.False(b1.HasValues(trunk{}))
	s.True(b1.HasValues(trunk{"Frank"}))
	s.True(b2.HasValues(trunk{}))

	// test other
	s.True(b1.HasValues("hello"))
}

func (s *SquintSuite) TestIf() {
	s.check("foo", s.empty, "foo", s.q.If(false, "bar"))
	s.check("foo bar", s.empty, "foo", s.q.If(true, "bar"))

	s.check("SELECT ?", binds{20}, "SELECT", s.q.If(false, 10), 20)
	s.check("SELECT ?, ?", binds{10, 20}, "SELECT", s.q.If(true, 10), 20)
}

func (s *SquintSuite) TestLog() {
	w := log.Writer()
	defer log.SetOutput(w)

	var buf bytes.Buffer
	log.SetOutput(&buf)

	// log everything
	s.q.Build(Log(true), "select", 3)
	s.Contains(buf.String(), "SQL:")
	s.Contains(buf.String(), "BINDS:")

	// log only query
	buf.Reset()
	s.q.Build(LogQuery(true), "select", 3)
	s.Contains(buf.String(), "SQL:")
	s.NotContains(buf.String(), "BINDS:")

	// log only binds
	buf.Reset()
	s.q.Build(LogBinds(true), "select", 3)
	s.NotContains(buf.String(), "SQL:")
	s.Contains(buf.String(), "BINDS:")
}

func (s *SquintSuite) TestFuzz() {
	var q query

	// sift something other than struct or map
	v := reflect.ValueOf("hi")
	cols, binds := q.sift(&v)
	s.Empty(cols)
	s.Empty(binds)
}

func (s *SquintSuite) check(wantSQL string, wantBinds interface{}, args ...interface{}) {
	sql, binds := s.q.Build(args...)
	s.Equal(wantSQL, sql)
	s.Equal(wantBinds, binds)
}
