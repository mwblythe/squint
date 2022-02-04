package squint

import (
	"reflect"
)

// These provide backward compatibility for deprectated options.
// Please do not use them for new code.

// EmptyValues (DEPRECATED): keep empty string struct/map field values
func EmptyValues(b bool) Option {
	return func(o *Options) {
		o.emptyValues = b
		o.emptyFn = compatEmpty(o.emptyValues, o.nilValues)
	}
}

// NilValues (DEPRECATED): keep nil struct/map field values?
func NilValues(b bool) Option {
	return func(o *Options) {
		o.nilValues = b
		o.emptyFn = compatEmpty(o.emptyValues, o.nilValues)
	}
}

// custom empty field handler
func compatEmpty(emptyValues, nilValues bool) EmptyFn {
	return func(in interface{}) (interface{}, bool) {
		v := reflect.ValueOf(in)

		switch v.Kind() {
		case reflect.String:
			return in, emptyValues
		case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Invalid:
			return nil, nilValues
		}

		return in, true
	}
}
