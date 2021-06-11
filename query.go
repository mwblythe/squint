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

// supported contexts
const (
	BASE sqlContext = iota
	INSERT
	SET
	IN
)

var insertRX = regexp.MustCompile(`(?i)\b(INSERT|REPLACE)\s+(?:\w+\s+)*INTO\s+\S+\s*$`)
var setRX = regexp.MustCompile(`(?i)\bSET\s*$`)
var inRX = regexp.MustCompile(`(?i)\bIN\s*$`)

// query represents a single SQL query that is being built.
// The query type is the heavy lifter of the package
type query struct {
	opt   Options
	sql   sqlBuf
	binds sqlBinds

	keepAll bool // internal override of empty mode
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
	switch b := bit.(type) {
	case Condition:
		if b.isTrue {
			for n := range b.bits {
				q.Add(b.bits[n])
			}
		}
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
			q.sql.Add("?")
			q.binds.Add(bit)
		}
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
		// treat string reference as a bind
		q.sql.Add("?")
		q.binds.Add(v.Elem().Interface())
	} else {
		q.Add(v.Elem().Interface())
	}
}

// addSlice adds the contents of a slice to the query.
// Normally it will be flattened and handled as inline arguments.
// Special handling for IN and multi-valued INSERT.
//
// TODO: skip complex types? handle refs?
func (q *query) addSlice(v reflect.Value) {
	context := q.Context()
	ty := v.Type().Elem().Kind()

	if context == IN {
		q.sql.Add("(")
		for i := 0; i < v.Len(); i++ {
			q.sql.Add("?")
			q.binds.Add(v.Index(i).Interface())
		}
		if v.Len() == 0 {
			q.sql.Add("NULL")
		}
		q.sql.Add(")")
	} else if context == INSERT && ty == reflect.Struct {
		// multi-row inserts MUST have the same number of binds per row
		// so we keep all values
		q.keepAll = true

		for i := 0; i < v.Len(); i++ {
			el := v.Index(i)
			cols, binds := q.siftStruct(&el)
			if i == 0 {
				q.sql.Add("( " + strings.Join(cols, ", ") + " ) VALUES")
			} else {
				q.sql.Add(", ")
			}
			q.sql.Add("( ?" + strings.Repeat(", ?", len(cols)-1) + " )")
			q.binds.Add(binds...)
		}

		q.keepAll = false
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
			c, b := q.siftStruct(&fieldVal)
			cols = append(cols, c...)
			binds = append(binds, b...)
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
// a struct or map into columns and binds. This is controlled by the
// KeepNil and KeepEmpty options.
func (q *query) checkValue(in interface{}, mode emptyMode) (interface{}, bool) {
	v := reflect.ValueOf(in)
	if v.IsValid() && !v.IsZero() {
		return in, true
	}

	if mode == eDefault {
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

// mapField maps a struct field to a db column name
func (q *query) mapField(field reflect.StructField) (name string, mode emptyMode) {
	// check for unexported fields
	if field.PkgPath != "" {
		return
	}

	// now the field tag
	if q.opt.tag != "" {
		tag := field.Tag.Get(q.opt.tag)
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
