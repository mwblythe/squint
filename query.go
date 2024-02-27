package squint

import (
	sqldriver "database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// sqlBuf is a buffer with some smarts for building up SQL
type sqlBuf struct {
	val         string
	lastWasBind bool
}

// Add appends a fragment to the SQL buffer (with separators where appropriate)
func (s *sqlBuf) Add(add string) {
	if add == "" {
		return
	}

	if L := len(s.val); L > 0 {
		last := s.val[L-1:]
		first := add[0:1]

		if first != "," && last != " " && first != " " {
			s.val += " " + add
		} else {
			s.val += add
		}
	} else {
		s.val = add
	}

	s.lastWasBind = false
}

// sqlState is the state of the SQL as it is built,
// to enable special handling of certain phrases.
type sqlState uint8

// SQL states
const (
	stateBase sqlState = iota
	stateInsert
	stateSet
	stateIn
)

var insertRX = regexp.MustCompile(`(?i)\b(INSERT|REPLACE)\s+(?:\w+\s+)*INTO\s+\S+\s*$`)
var setRX = regexp.MustCompile(`(?i)\bSET\s*$`)
var inRX = regexp.MustCompile(`(?i)\bIN\s*$`)

// query represents a single SQL query that is being built
type query struct {
	opt   Options
	sql   sqlBuf
	binds []interface{}

	keepAll bool // internal override of empty mode
}

// state returns the query's current state
func (q *query) state() sqlState {
	switch {
	case insertRX.MatchString(q.sql.val):
		return stateInsert
	case setRX.MatchString(q.sql.val):
		return stateSet
	case inRX.MatchString(q.sql.val):
		return stateIn
	default:
		return stateBase
	}
}

// Add a piece to the query
func (q *query) Add(bit interface{}) {
	if _, ok := bit.(sqldriver.Valuer); ok {
		q.addBind(bit)
		return
	}

	switch b := bit.(type) {
	case Condition:
		q.addCondition(b)
	case Option:
		q.opt.SetOption(b)
	default:
		v := reflect.ValueOf(bit)

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
			q.addBind(bit)
		}
	}
}

func (q *query) addBind(values ...interface{}) {
	for _, v := range values {
		if q.sql.lastWasBind {
			q.sql.Add(", ")
		}

		q.sql.Add(q.opt.bindFn(len(q.binds) + 1))
		q.binds = append(q.binds, v)
		q.sql.lastWasBind = true
	}
}

// addCondition adds bits to the query if condition is true
func (q *query) addCondition(c Condition) {
	if c.isTrue {
		for n := range c.bits {
			q.Add(c.bits[n])
		}
	}
}

// addString adds a string to the query.
// Normal strings will be treated as SQL.
// A string pointer (or Bind type) is treated as a bind value.
func (q *query) addString(v reflect.Value) {
	if v.Type() == reflect.TypeOf(Bind("")) {
		q.addBind(v.String())
	} else {
		q.sql.Add(v.String())
	}
}

// addPointer adds a pointer type to the query
func (q *query) addPointer(v reflect.Value) {
	switch {
	case v.IsNil():
		q.addBind(nil)
	case v.Elem().Kind() == reflect.String:
		// treat string reference as a bind
		q.addBind(v.Elem().Interface())
	default:
		q.Add(v.Elem().Interface())
	}
}

// addSlice adds the contents of a slice to the query.
// Normally it will be flattened and handled as inline arguments.
// Special handling for IN and multi-valued INSERT.
func (q *query) addSlice(v reflect.Value) {
	state := q.state()
	ty := v.Type().Elem().Kind()

	switch {
	case state == stateIn:
		q.sql.Add("(")

		for i := 0; i < v.Len(); i++ {
			q.addBind(v.Index(i).Interface())
		}

		if v.Len() == 0 {
			q.sql.Add("NULL")
		}

		q.sql.Add(")")
	case state == stateInsert && ty == reflect.Struct:
		// multi-row inserts MUST have the same number of binds per row
		// so we keep all values
		q.keepAll = true

		for i := 0; i < v.Len(); i++ {
			el := v.Index(i)
			cols, binds := q.siftStruct(&el)

			if i == 0 {
				q.sql.Add("( " + strings.Join(cols, ", ") + " ) VALUES")
			} else {
				q.sql.Add(",")
			}

			q.sql.Add("(")
			q.addBind(binds...)
			q.sql.Add(")")
		}

		q.keepAll = false
	default:
		for i := 0; i < v.Len(); i++ {
			q.Add(v.Index(i).Interface())
		}
	}
}

// addComplex adds a struct or map to the query
func (q *query) addComplex(v reflect.Value) {
	cols, binds := q.sift(&v)

	switch q.state() {
	case stateInsert:
		if len(cols) > 0 {
			q.sql.Add("( " + strings.Join(cols, ", ") + " ) VALUES (")
			q.addBind(binds...)
			q.sql.Add(")")
		}
	case stateSet:
		for i, col := range cols {
			if i > 0 {
				q.sql.Add(",")
			}

			q.sql.Add(col + " = ")
			q.addBind(binds[i])
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
				q.sql.Add(col + " = ")
				q.addBind(binds[i])
			}
		}
	}
}

// sift a map or struct into cols + binds
func (q *query) sift(v *reflect.Value) (cols []string, binds []interface{}) {
	switch v.Kind() {
	case reflect.Struct:
		cols, binds = q.siftStruct(v)
	case reflect.Map:
		cols, binds = q.siftMap(v)
	}

	return
}

// siftMap will sift a map into cols + binds
func (q *query) siftMap(src *reflect.Value) ([]string, []interface{}) {
	cols := make([]string, 0, src.Len())
	valmap := make(map[string]interface{})
	iter := src.MapRange()

	// build list of cols and value map
	for iter.Next() {
		if v, ok := q.checkValue(iter.Value().Interface(), eDefault); ok {
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
	cols := make([]string, 0, src.NumField())
	binds := make([]interface{}, 0, src.NumField())

	for i := 0; i < src.NumField(); i++ {
		field := src.Type().Field(i)
		fieldVal := src.Field(i)

		if fieldVal.Kind() == reflect.Struct && field.Anonymous {
			if q.tagValue(field) != "-" {
				c, b := q.siftStruct(&fieldVal)
				cols = append(cols, c...)
				binds = append(binds, b...)
			}
		} else {
			name, mode := q.mapField(field)
			if name == "" {
				continue
			}
			if v, ok := q.checkValue(fieldVal.Interface(), mode); ok {
				cols = append(cols, name)
				binds = append(binds, v)
			}
		}
	}

	return cols, binds
}

// keepValue determines whether a given value should be kept when sifting
// a struct or map into columns and binds. This is controlled by the empty mode.
func (q *query) checkValue(in interface{}, mode emptyMode) (interface{}, bool) {
	var v reflect.Value

	if valuer, ok := in.(sqldriver.Valuer); ok {
		if val, err := valuer.Value(); err == nil {
			v = reflect.ValueOf(val)
		}
	} else {
		v = reflect.ValueOf(in)
	}

	if v.IsValid() && !v.IsZero() {
		return in, true
	}

	if mode == eDefault {
		if fn := q.opt.emptyFn; fn != nil {
			out, keep := fn(in)
			if q.keepAll {
				keep = true
			}

			return out, keep
		}

		mode = q.opt.empty
	}

	switch mode {
	case eOmit:
		if !q.keepAll {
			return nil, false
		}
	case eNull:
		return nil, true
	}

	return in, true // eKeep
}

// tagValue returns a field's tag value (if any)
func (q *query) tagValue(field reflect.StructField) string {
	if q.opt.tag != "" {
		return field.Tag.Get(q.opt.tag)
	}

	return ""
}

// mapField maps a struct field to a db column name
func (q *query) mapField(field reflect.StructField) (name string, mode emptyMode) {
	// check for unexported fields
	if field.PkgPath != "" {
		return
	}

	// now the field tag
	tag := q.tagValue(field)
	if tag == "-" {
		return
	}

	for _, t := range strings.Split(tag, ",") {
		switch t {
		case "keepempty":
			mode = eKeep
		case "omitempty":
			mode = eOmit
		case "nullempty":
			mode = eNull
		default:
			if name == "" {
				name = t
			}
		}
	}

	// default to field name itself
	if name == "" {
		name = field.Name
	}

	return
}

func bindQuestion(int) string {
	return "?"
}

func bindAt(seq int) string {
	return fmt.Sprintf("@p%d", seq)
}

func bindDollar(seq int) string {
	return fmt.Sprintf("$%d", seq)
}

func bindColon(seq int) string {
	return fmt.Sprintf(":b%d", seq)
}
