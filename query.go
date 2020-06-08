package squint

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// sqlBuf is a buffer with some smarts for building up SQL
type sqlBuf struct {
	val string
}

// Add appends a fragment to the SQL buffer (with separators where appropriate)
func (s *sqlBuf) Add(add string) {
	if add == "" {
		return
	}

	if L := len(s.val); L > 0 {
		last := s.val[L-1:]
		first := add[0:1]

		if last == "?" && first == last {
			s.val += ", " + add
		} else if first != "," && last != " " && first != " " {
			s.val += " " + add
		} else {
			s.val += add
		}
	} else {
		s.val = add
	}
}

// sqlBinds is used for collecting SQL bind values
type sqlBinds struct {
	vals []interface{}
}

// Add appends a bind to the collection
func (s *sqlBinds) Add(args ...interface{}) {
	s.vals = append(s.vals, args...)
}

// sqlContext tracks the state of the SQL as it is built, in order
// to support special handling based on context
type sqlContext uint8

const (
	BASE sqlContext = iota
	INSERT
	SET
	IN
)

var insertRX = regexp.MustCompile(`(?i)\b(INSERT|REPLACE)\s+INTO\s+\S+\s*$`)
var setRX = regexp.MustCompile(`(?i)\bSET\s*$`)
var inRX = regexp.MustCompile(`(?i)\bIN\s*$`)

// query represents a single SQL query that is being built.
// The query type is the heavy lifter of the package
type query struct {
	opt   *Options
	sql   sqlBuf
	binds sqlBinds
}

// Context returns the query's current context
func (q *query) Context() sqlContext {
	if insertRX.MatchString(q.sql.val) {
		return INSERT
	}
	if setRX.MatchString(q.sql.val) {
		return SET
	}
	if inRX.MatchString(q.sql.val) {
		return IN
	}

	return BASE
}

// Add a piece to the query
func (q *query) Add(bit interface{}) {
	v := reflect.ValueOf(bit)

	if (v.Type() == reflect.TypeOf(Condition{})) {
		if i := bit.(Condition); i.isTrue {
			for n := range i.bits {
				q.Add(i.bits[n])
			}
		}
		return
	}

	switch v.Kind() {
	case reflect.String:
		q.addString(v)
	case reflect.Ptr:
		q.addPointer(v)
	case reflect.Array, reflect.Slice:
		q.addSlice(v)
	case reflect.Map, reflect.Struct:
		q.addComplex(v)
	default:
		q.sql.Add("?")
		q.binds.Add(bit)
	}
}

// addString adds a string to the query.
// Normal strings will be treated as SQL.
// A string pointer (or Bind type) is treated as a bind value.
func (q *query) addString(v reflect.Value) {
	if /* q.Context() != BASE || */ v.Type() == reflect.TypeOf(Bind("")) {
		q.sql.Add("?")
		q.binds.Add(v.String())
	} else {
		q.sql.Add(v.String())
	}
}

// addPointer adds a pointer type to the query
func (q *query) addPointer(v reflect.Value) {
	if v.IsNil() {
		q.sql.Add("?")
		q.binds.Add(nil)
	} else if v.Elem().Kind() == reflect.String {
		// treat string reference same as Bind
		q.Add(Bind(v.Elem().String()))
	} else {
		q.Add(v.Elem().Interface())
	}
}

// addSlice adds the contents of a slice to the query.
// Normally it will be flattened an handled as inline arguments.
// For an IN statement, it will create a series of binds within parentheses.
//
// TODO: skip complex types? handle refs?
func (q *query) addSlice(v reflect.Value) {
	if q.Context() == IN {
		q.sql.Add("(")
		for i := 0; i < v.Len(); i++ {
			q.sql.Add("?")
			q.binds.Add(v.Index(i).Interface())
		}
		if v.Len() == 0 {
			q.sql.Add("NULL")
		}
		q.sql.Add(")")
	} else {
		for i := 0; i < v.Len(); i++ {
			q.Add(v.Index(i).Interface())
		}
	}
}

// addComlex adds a struct or map to the query. Handling varies by context.
func (q *query) addComplex(v reflect.Value) {
	cols, binds := q.sift(&v)

	switch q.Context() {
	case INSERT:
		if len(cols) > 0 {
			q.sql.Add("( " + strings.Join(cols, ", ") + " ) VALUES ( ?")
			q.sql.Add(strings.Repeat(", ?", len(cols)-1) + " )")
			q.binds.Add(binds...)
		}
	case SET:
		for i, col := range cols {
			if i > 0 {
				q.sql.Add(",")
			}
			q.sql.Add(col + " = ?")
			q.binds.Add(binds[i])
		}
	default:
		for i, col := range cols {
			if i > 0 {
				q.sql.Add("AND")
			}

			switch bv := reflect.ValueOf(binds[i]); bv.Kind() {
			case reflect.Slice, reflect.Array:
				q.sql.Add(col + " IN")
				q.addSlice(bv)
			default:
				q.sql.Add(col + " = ?")
				q.binds.Add(binds[i])
			}
		}
	}
}

// sift a map or struct into cols + binds
func (q *query) sift(v *reflect.Value) ([]string, []interface{}) {
	switch v.Kind() {
	case reflect.Struct:
		return q.siftStruct(v)
	case reflect.Map:
		return q.siftMap(v)
	default:
		return []string{}, []interface{}{}
	}
}

// siftMap will sift a map into cols + binds
func (q *query) siftMap(src *reflect.Value) ([]string, []interface{}) {
	cols := make([]string, src.Len())[:0]
	valmap := make(map[string]interface{})
	iter := src.MapRange()

	// build list of cols and value map
	for iter.Next() {
		if v := iter.Value().Interface(); q.keepValue(v) {
			k := iter.Key().String()
			cols = append(cols, k)
			valmap[k] = v
		}
	}

	sort.Strings(cols)

	// add binds in (sorted) column order
	binds := make([]interface{}, len(cols))
	for i, col := range cols {
		binds[i] = valmap[col]
	}

	return cols, binds
}

// siftStruct will sift a struct into cols + binds
func (q *query) siftStruct(src *reflect.Value) ([]string, []interface{}) {
	cols := make([]string, src.NumField())[:0]
	binds := make([]interface{}, src.NumField())[:0]

	for i := 0; i < src.NumField(); i++ {
		name := q.mapField(src.Type().Field(i))
		if name == "" {
			continue
		}

		if v := src.Field(i).Interface(); q.keepValue(v) {
			cols = append(cols, name)
			binds = append(binds, v)
		}
	}

	return cols, binds
}

// keepValue determines whether a given value should be kept when sifting
// a struct or map into columns and binds. This is controlled by the
// KeepNil and KeepEmpty options.
func (q *query) keepValue(i interface{}) bool {
	v := reflect.ValueOf(i)

	if !q.opt.KeepNil {
		switch v.Kind() {
		case reflect.Ptr, reflect.Map, reflect.Slice:
			return !v.IsNil()
		case reflect.Invalid:
			return false
		}
	}

	if !q.opt.KeepEmpty {
		if v.Kind() == reflect.String {
			return !v.IsZero()
		}
	}

	return true
}

// mapField maps a struct field to a db column name
func (q *query) mapField(field reflect.StructField) string {
	var name string

	// check for unexported fields
	if field.PkgPath != "" {
		return ""
	}

	// now the field tag
	if q.opt.Tag != "" {
		if name = field.Tag.Get(q.opt.Tag); name == "-" {
			return ""
		}
	}

	// default to field name itself
	if name == "" {
		name = field.Name
	}

	return name
}
