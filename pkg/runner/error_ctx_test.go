package runner

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorCtx_Errorf(t *testing.T) {
	t.Run("creates error with message", func(t *testing.T) {
		errCtx := NewErrorCtx("description", []byte("additional details"), WithErrCtxLogAdditionalCtx(false))
		err := errCtx.Errorf("failed: %w", errors.New("simulated.err"))
		require.Equal(t, "failed: simulated.err", err.Error())
	})

	t.Run("creates error with message and additional context", func(t *testing.T) {
		errCtx := NewErrorCtx("description", []byte("additional details"), WithErrCtxLogAdditionalCtx(true))
		err := errCtx.Errorf("failed: %w", errors.New("simulated.err"))
		require.Equal(t, "failed: simulated.err, description: additional details", err.Error())
	})
}
