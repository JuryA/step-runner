package runner_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestGlobalContext_Log(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		output := &strings.Builder{}
		globalCtx := runner.NewGlobalContext(bldr.Env().Build())
		globalCtx.Stdout = output

		err := globalCtx.Logf("Hello %s!", "World")
		require.NoError(t, err)
		require.Equal(t, "Hello World!", output.String())
	})

	t.Run("when errors", func(t *testing.T) {
		globalCtx := runner.NewGlobalContext(bldr.Env().Build())
		globalCtx.Stdout = &ErrWriter{err: errors.New("simulated.error")}

		err := globalCtx.Logf("log message")
		require.Error(t, err)
		require.Contains(t, err.Error(), "writing to stdout: simulated.error")
	})
}

type ErrWriter struct {
	err error
}

func (e *ErrWriter) Write(_ []byte) (n int, err error) {
	return 0, e.err
}
