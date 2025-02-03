package releaser

import "log/slog"

type Option func(*options) error

type options struct {
	logger *slog.Logger
	dir    string
}

func WithLogger(logger *slog.Logger) Option {
	return func(o *options) error {
		o.logger = logger
		return nil
	}
}

func WithDirectory(dir string) Option {
	return func(o *options) error {
		o.dir = dir
		return nil
	}
}
