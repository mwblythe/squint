package driver

import (
	"context"
	"database/sql/driver"

	"github.com/mwblythe/squint"
)

type builder struct {
	*squint.Builder
}

func newBuilder(b *squint.Builder) *builder {
	return &builder{Builder: b}
}

func (b *builder) BuildValues(query string, inVals []driver.Value) (string, []driver.Value) {
	bits := make([]interface{}, len(inVals)+1)
	bits[0] = query
	for n := range inVals {
		bits[n+1] = inVals[n]
	}

	query, binds := b.Build(bits...)

	outVals := make([]driver.Value, len(binds))
	for n := range binds {
		outVals[n] = driver.Value(binds[n])
	}

	return query, outVals
}

func (b *builder) BuildContext(ctx context.Context, query string, inVals []driver.NamedValue) (context.Context, string, []driver.NamedValue) {
	bits := make([]interface{}, len(inVals)+1)
	bits[0] = query
	for n := range inVals {
		bits[n+1] = inVals[n].Value
	}

	query, binds := b.Build(bits...)

	outVals := make([]driver.NamedValue, len(binds))
	for n := range binds {
		outVals[n] = driver.NamedValue{
			Ordinal: n + 1,
			Value:   driver.Value(binds[n]),
		}
	}

	return ctx, query, outVals
}
