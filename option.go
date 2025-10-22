package iso8601duration

type config struct {
	// excludeStartDate 初日不算入か (nilの場合は自動)
	excludeStartDate *bool
	// preserveTimeOnZero 期間が0の場合、時刻を維持するか
	preserveTimeOnZero bool
}

type Option func(*config)

func WithExcludeStartDate(v bool) Option {
	return func(o *config) {
		o.excludeStartDate = &v
	}
}

func WithPreserveTimeOnZero() Option {
	return func(o *config) {
		o.preserveTimeOnZero = true
	}
}
