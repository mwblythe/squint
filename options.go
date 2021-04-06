package squint

// Options for the squint Builder
type Options struct {
	tag       string // field tag to use
	keepNil   bool   // keep nil struct/map field values
	keepEmpty bool   // keep empty string struct/map field values
	logQuery  bool   // log queries?
	logBinds  bool   // log binds?
}

type Option func(*Options)

func (o *Options) Option(options ...Option) {
	for _, opt := range options {
		opt(o)
	}
}

func Tag(tag string) Option {
	return func(o *Options) {
		o.tag = tag
	}
}

func NilValues(b bool) Option {
	return func(o *Options) {
		o.keepNil = b
	}
}

func EmptyValues(b bool) Option {
	return func(o *Options) {
		o.keepEmpty = b
	}
}

func Log(b bool) Option {
	return func(o *Options) {
		o.logQuery = b
		o.logBinds = b
	}
}

func LogQuery(b bool) Option {
	return func(o *Options) {
		o.logQuery = b
	}
}

func LogBinds(b bool) Option {
	return func(o *Options) {
		o.logBinds = b
	}
}
