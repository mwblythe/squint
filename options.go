package squint

type emptyMode int

const (
	eDefault emptyMode = iota
	eKeep
	eOmit
	eNull
)

// Options for the squint Builder
type Options struct {
	tag      string    // field tag to use
	empty    emptyMode // how to treat empty field values
	logQuery bool      // log queries?
	logBinds bool      // log binds?
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
