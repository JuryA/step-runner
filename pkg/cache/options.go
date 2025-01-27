package cache

type Option func(*options) error

type options struct {
	dir      string
	gitDepth int
}

func WithGitDepth(depth int) Option {
	return func(o *options) error {
		o.gitDepth = depth
		return nil
	}
}

func WithDir(dir string) Option {
	return func(o *options) error {
		o.dir = dir
		return nil
	}
}
