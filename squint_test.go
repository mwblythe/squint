package squint

import (
	"bytes"
	"database/sql"
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
	s.Run("basic", func() {
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
	})

	type person struct {
		First string
		Last  string
	}

	s.Run("embed", func() {
		s.check(
			"WHERE ID = ? AND First = ? AND Last = ?",
			binds{10, "Frank", "Gallagher"},
			"WHERE",
			struct {
				ID int
				person
			}{10, person{"Frank", "Gallagher"}},
		)
	})

	s.Run("embed+skip", func() {
		s.check(
			"WHERE ID = ?",
			binds{10},
			"WHERE",
			struct {
				ID     int
				person `db:"-"`
			}{10, person{"Frank", "Gallagher"}},
		)
	})

	s.Run("no-tag", func() {
		b := NewBuilder(Tag(""))
		_, binds := b.Build("SELECT", person{})
		s.Len(binds, 2)
	})
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

	type Row struct {
		ID     int
		Size   string
		Rating int `db:"omitempty"`
	}

	s.check(
		"INSERT INTO junk ( ID, Size ) VALUES ( ?, ? )",
		binds{5, "small"},
		"INSERT INTO junk",
		Row{5, "small", 0},
	)

	s.check(
		"INSERT INTO junk ( ID, Size, Rating ) VALUES ( ?, ?, ? ), ( ?, ?, ? )",
		binds{1, "small", 0, 2, "medium", 1},
		"INSERT INTO junk", []Row{
			{1, "small", 0},
			{2, "medium", 1},
		},
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
}

func (s *SquintSuite) TestHasValues() {
	var bar *bool
	type person struct {
		Name string
		Age  int
	}

	empty := []interface{}{
		H{"name": ""},
		H{"foo": nil},
		H{"foo": bar},
		person{},
	}

	s.Run("KeepEmpty", func() {
		b := NewBuilder(KeepEmpty())
		for _, v := range empty {
			s.True(b.HasValues(v))
		}
		s.False(b.HasValues(H{}))
	})

	s.Run("OmitEmpty", func() {
		b := NewBuilder(OmitEmpty())
		for _, v := range empty {
			s.False(b.HasValues(v))
		}
		s.False(b.HasValues(H{}))
	})

	s.Run("NullEmpty", func() {
		b := NewBuilder(NullEmpty())
		for _, v := range empty {
			s.True(b.HasValues(v))
		}
		s.False(b.HasValues(H{}))
	})

	s.Run("NotEmpty", func() {
		b := NewBuilder(OmitEmpty())
		s.True(b.HasValues("hello"))
		s.True(b.HasValues(H{"age": 10}))
		s.True(b.HasValues(person{"Frank", 0}))
	})
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

func (s *SquintSuite) TestEmpty() {
	orig := s.q
	defer func() { s.q = orig }()

	var rec struct {
		Name string
		Num  int
		Flag bool
	}

	s.Run("KeepEmpty", func() {
		s.q = NewBuilder(KeepEmpty())
		s.check(
			"SET Name = ?, Num = ?, Flag = ?",
			binds{"", 0, false},
			"SET", rec,
		)
	})

	s.Run("OmitEmpty", func() {
		s.q = NewBuilder(OmitEmpty())
		s.check(
			"SET", s.empty,
			"SET", rec,
		)
	})

	s.Run("NullEmpty", func() {
		s.q = NewBuilder(NullEmpty())
		s.check(
			"SET Name = ?, Num = ?, Flag = ?",
			binds{nil, nil, nil},
			"SET", rec,
		)
	})

	var rec2 struct {
		Name string `db:"omitempty"`
		Num  int    `db:"nullempty"`
		Flag bool   `db:"keepempty"`
	}

	// all fields have empty mode overrides, so results should be
	// the same regardless of builder's empty mode
	s.Run("EmptyTags", func() {
		for _, o := range []Option{KeepEmpty(), OmitEmpty(), NullEmpty()} {
			s.q.SetOption(o)
			s.check(
				"SET Num = ?, Flag = ?",
				binds{nil, false},
				"SET", rec2,
			)
		}
	})
}

func (s *SquintSuite) TestEmptyFn() {
	type empty struct {
		Int  int
		Str  string
		Bool bool
		Ptr  interface{}
	}

	b := NewBuilder(
		WithEmptyFn(func(in interface{}) (out interface{}, keep bool) {
			return "beer", true
		}),
	)

	s.Run("custom", func() {
		s.NotNil(b.emptyFn)

		_, vals := b.Build("SELECT", empty{})
		s.EqualValues(
			binds{"beer", "beer", "beer", "beer"},
			vals,
		)
	})

	s.Run("bulk", func() {
		// do not keep empty strings
		b.SetOption(WithEmptyFn(func(in interface{}) (out interface{}, keep bool) {
			if _, ok := in.(string); ok {
				return in, false
			}

			return in, true
		}))

		// string should be omitted
		_, vals := b.Build("SELECT", empty{})
		s.EqualValues(
			binds{0, false, nil},
			vals,
		)

		// bulk insert, all values kept
		_, vals = b.Build("INSERT INTO foo", make([]empty, 2))
		s.EqualValues(
			binds{0, "", false, nil, 0, "", false, nil},
			vals,
		)
	})

	s.Run("default", func() {
		b.SetOption(WithDefaultEmpty())

		_, vals := b.Build("SELECT", empty{})
		s.EqualValues(
			binds{0, "", false, nil},
			vals,
		)
	})
}

func (s *SquintSuite) check(wantSQL string, wantBinds interface{}, args ...interface{}) {
	sql, binds := s.q.Build(args...)
	s.Equal(wantSQL, sql)
	s.Equal(wantBinds, binds)
}

func (s *SquintSuite) TestValuer() {
	var val sql.NullString

	s.check("update table set col = ?", binds{val}, "update table set col =", val)

	var record struct {
		Name sql.NullString
	}

	s.check("update table set Name = ?", binds{record.Name}, "update table set", record)

	builder := NewBuilder(OmitEmpty())
	s.False(builder.HasValues(record))

	record.Name.String = "Frank"
	record.Name.Valid = true
	s.True(builder.HasValues(record))
}
