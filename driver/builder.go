package driver

import (
	"database/sql/driver"

	"github.com/mwblythe/squint"
)

type builder struct {
	*squint.Builder
}

func newBuilder(b *squint.Builder) *builder {
	return &builder{Builder: b}
}

func (b *builder) BuildNamed(query string, inVals []driver.NamedValue) (string, []interface{}) {
	bits := make([]interface{}, len(inVals)+1)
	bits[0] = query
	for n := range inVals {
		bits[n+1] = inVals[n].Value
	}

	return b.Build(bits...)
}
