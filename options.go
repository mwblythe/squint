package squint

// Options for the squint Builder
type Options struct {
	Tag       string // field tag to use
	KeepNil   bool   // keep nil struct/map field values
	KeepEmpty bool   // keep empty string struct/map field values
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
		o.Tag = tag
	}
}

func NilValues(b bool) Option {
	return func(o *Options) {
		o.KeepNil = b
	}
}

func EmptyValues(b bool) Option {
	return func(o *Options) {
		o.KeepEmpty = b
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
