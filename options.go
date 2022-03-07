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

// BindFn is a bind placholder handler
type BindFn func(pos int) string

// Options for the squint Builder
type Options struct {
	tag      string    // field tag to use
	empty    emptyMode // how to treat empty field values
	logQuery bool      // log queries?
	logBinds bool      // log binds?
	emptyFn  EmptyFn   // custom empty field handler
	bindFn   BindFn    // bind placeholder handler

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

// BindQuestion uses ? as placeholder (MySQL, sqlite)
func BindQuestion() Option {
	return func(o *Options) {
		o.bindFn = bindQuestion
	}
}

// BindAt uses @p1, @p2 style placeholders (sqlserver)
func BindAt() Option {
	return func(o *Options) {
		o.bindFn = bindAt
	}
}

// BindDollar uses $1, $2 style placeholders (postgres, sqlite)
func BindDollar() Option {
	return func(o *Options) {
		o.bindFn = bindDollar
	}
}

// BindColon uses :b1, :b2 style placeholders (oracle)
func BindColon() Option {
	return func(o *Options) {
		o.bindFn = bindColon
	}
}

// WithBindFn uses a custom bind placeholder function of the form:
//
// func(seq int) string
//
// where seq is a 1-based bind sequence
func WithBindFn(fn BindFn) Option {
	return func(o *Options) {
		o.bindFn = fn
	}
}
