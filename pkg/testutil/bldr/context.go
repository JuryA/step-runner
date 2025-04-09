package bldr

import (
	"context"
	"testing"
	"time"
)

type ContextBuilder struct {
	t       *testing.T
	timeout time.Duration
}

func DefaultCtx(t *testing.T) context.Context {
	return Ctx(t).Build()
}

func Ctx(t *testing.T) *ContextBuilder {
	return &ContextBuilder{
		t:       t,
		timeout: 5 * time.Second,
	}
}

func (b *ContextBuilder) Build() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	b.t.Cleanup(cancel)

	return ctx
}
