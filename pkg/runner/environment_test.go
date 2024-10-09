package runner

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEnvironment(t *testing.T) {
	t.Run("loads all operating system environment variables", func(t *testing.T) {
		require.NoError(t, os.Setenv("LANG", "en"))
		defer func() { _ = os.Unsetenv("LANG") }()

		env, err := NewEnvironmentFromOS()
		require.NoError(t, err)
		require.Equal(t, "en", env.Values()["LANG"])
	})

	t.Run("loads predefined operating system environment variables", func(t *testing.T) {
		require.NoError(t, os.Setenv("LANG", "en"))
		require.NoError(t, os.Setenv("FOO", "BAR"))
		defer func() { _ = os.Unsetenv("LANG") }()
		defer func() { _ = os.Unsetenv("FOO") }()

		env, err := NewEnvironmentFromOSWithKnownVars()
		require.NoError(t, err)
		require.Equal(t, "en", env.Values()["LANG"])
		require.NotContains(t, "FOO", env.Values())
	})
}

func TestEnvironment_AddLexicalScope(t *testing.T) {
	t.Run("adds to a new environment", func(t *testing.T) {
		a := NewEnvironment(map[string]string{"foo": "bar"})
		b := a.AddLexicalScope(map[string]string{"baz": "qux"})

		require.Equal(t, map[string]string{"foo": "bar"}, a.Values())
		require.Equal(t, map[string]string{"foo": "bar", "baz": "qux"}, b.Values())
	})

	t.Run("added lexical scope takes precedence over already added environment", func(t *testing.T) {
		a := NewEnvironment(map[string]string{"foo": "bar"})
		b := a.AddLexicalScope(map[string]string{"foo": "qux"})

		require.Equal(t, map[string]string{"foo": "qux"}, b.Values())
	})
}
