package squint

type emptyMode int

const (
	eDefault emptyMode = iota
	eKeep
	eOmit
	eNull
)

// EmptyFn is an empty field handler
type EmptyFn func(in interface{}) (out interface{}, keep bool)

// Options for the squint Builder
type Options struct {
	tag      string    // field tag to use
	empty    emptyMode // how to treat empty field values
	logQuery bool      // log queries?
	logBinds bool      // log binds?
	emptyFn  EmptyFn   // custom empty field handler

	// deprecated
	emptyValues bool
	nilValues   bool
}

// Option is a functional option
type Option func(*Options)

// SetOption applies the given options
func (o *Options) SetOption(options ...Option) {
	for _, opt := range options {
		opt(o)
	}
}

// Tag is the field tag to use
func Tag(tag string) Option {
	return func(o *Options) {
		o.tag = tag
	}
}

// KeepEmpty will keep empty fields
func KeepEmpty() Option {
	return func(o *Options) {
		o.empty = eKeep
	}
}

// OmitEmpty will omit empty fields
func OmitEmpty() Option {
	return func(o *Options) {
		o.empty = eOmit
	}
}

// NullEmpty will treat empty fields as null
func NullEmpty() Option {
	return func(o *Options) {
		o.empty = eNull
	}
}

// Log : log queries and binds
func Log(b bool) Option {
	return func(o *Options) {
		o.logQuery = b
		o.logBinds = b
	}
}

// LogQuery : log queries
func LogQuery(b bool) Option {
	return func(o *Options) {
		o.logQuery = b
	}
}

// LogBinds : log binds
func LogBinds(b bool) Option {
	return func(o *Options) {
		o.logBinds = b
	}
}

// WithEmptyFn : use custom empty field handler:
//
// func(in interface{}) (out interface{}, keep bool)
//
// in - incoming value
// out - outgoing value to use in SQL
// keep - keep the value or skip it?
func WithEmptyFn(fn EmptyFn) Option {
	return func(o *Options) {
		o.emptyFn = fn
	}
}

// WithDefaultEmpty : use default empty field handler
func WithDefaultEmpty() Option {
	return func(o *Options) {
		o.emptyFn = nil
	}
}
