package squint

// Options for the squint Builder
type Options struct {
	tag       string // field tag to use
	keepNil   bool   // keep nil struct/map field values
	keepEmpty bool   // keep empty string struct/map field values
	logQuery  bool   // log queries?
	logBinds  bool   // log binds?
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

// NilValues : keep nil struct/map field values?
func NilValues(b bool) Option {
	return func(o *Options) {
		o.keepNil = b
	}
}

// EmptyValues : keep empty string struct/map field values
func EmptyValues(b bool) Option {
	return func(o *Options) {
		o.keepEmpty = b
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
