package squint

import (
	"log"
	"reflect"
)

// Bind treats a string as a bind rather than SQL fragment
type Bind string

// Condition will conditionally process a list of arguments
type Condition struct {
	isTrue bool
	bits   []interface{}
}

// Builder is the core of public squint interactions.
// It's responsible for processing inputs into SQL and binds
type Builder struct {
	Options
}

// NewBuilder returns a new Builder with the supplied options
func NewBuilder(options ...Option) *Builder {
	var b Builder
	b.SetOption(Tag("db"), KeepEmpty())
	b.SetOption(options...)
	return &b
}

// Build accepts a list of SQL fragments and Go variables and
// interpolates them into a query and a set of binds. These are
// appropriate to pass into a variety of execution methods of
// the sql (or sqlx) package.
//
// sql, binds := b.Build("INSERT INTO users", &User)
//
func (b *Builder) Build(bits ...interface{}) (string, []interface{}) {
	q := query{opt: b.Options}

	for _, bit := range bits {
		q.Add(bit)
	}

	if q.opt.logQuery {
		log.Println("SQL:", q.sql.val)
	}

	if q.opt.logBinds {
		log.Println("BINDS:", q.binds.vals)
	}

	return q.sql.val, q.binds.vals
}

// If allows for conditionally including a list of arguments in a query.
// This is a convenience to allow a bit of inline logic when calling Build:
//
// sql, binds := b.Build(
//   "SELECT u.* FROM users u",
//   b.If(
//     EmployeesOnly,
//     "JOIN employees e ON u.id = e.id"
//   ),
//   "WHERE id IN", ids
// )
//
func (b *Builder) If(condition bool, bits ...interface{}) Condition {
	return If(condition, bits...)
}

// If : package level version, in case Builder instance isn't handy
func If(condition bool, bits ...interface{}) Condition {
	return Condition{
		isTrue: condition,
		bits:   bits,
	}
}

// HasValues evaluates whether a struct or map has values that
// would be used according to the Builder's "Keep*"" options.
//
// If you have a situation where one might be considered empty,
// you can use this as a pre-check to avoid generating invalid SQL
func (b *Builder) HasValues(src interface{}) bool {
	q := query{opt: b.Options}
	v := reflect.ValueOf(src)

	switch v.Kind() {
	case reflect.Struct, reflect.Map:
		cols, _ := q.sift(&v)
		return len(cols) > 0
	default:
		return true
	}
}
