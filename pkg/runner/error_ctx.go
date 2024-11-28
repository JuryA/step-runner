package runner

import "fmt"

type ErrorCtx struct {
	description      string // description explains what the additional context is
	additionalCxt    string // additional context to be logged when in debug mode
	logAdditionalCtx bool
}

func NewErrorCtx(description string, additionalCxt []byte, options ...func(*ErrorCtx)) *ErrorCtx {
	errCtx := &ErrorCtx{
		description:      description,
		additionalCxt:    string(additionalCxt),
		logAdditionalCtx: RunningInDebugMode,
	}

	for _, opt := range options {
		opt(errCtx)
	}

	return errCtx
}

func (e *ErrorCtx) Errorf(message string, v ...any) error {
	if e.logAdditionalCtx {
		v = append(v, e.description, e.additionalCxt)
		return fmt.Errorf(message+", %s: %s", v...)
	}

	return fmt.Errorf(message, v...)
}

func WithErrCtxLogAdditionalCtx(logAdditionalCtx bool) func(*ErrorCtx) {
	return func(errCtx *ErrorCtx) {
		errCtx.logAdditionalCtx = logAdditionalCtx
	}
}
