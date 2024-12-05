package runner

import (
	"os"
	"strings"
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
		require.NotContains(t, env.Values(), "FOO")
	})

	t.Run("excludes environment variables based on a predicate", func(t *testing.T) {
		require.NoError(t, os.Setenv("INCLUDE_VALUE", "1"))
		require.NoError(t, os.Setenv("EXCLUDE_VALUE", "2"))
		defer func() { _ = os.Unsetenv("INCLUDE_VALUE") }()
		defer func() { _ = os.Unsetenv("EXCLUDE_VALUE") }()

		env, err := NewEnvironmentFromOS(func(envName string) bool { return strings.HasPrefix(envName, "EXCLUDE_") })
		require.NoError(t, err)
		require.Equal(t, "1", env.Values()["INCLUDE_VALUE"])
		require.NotContains(t, env.Values(), "EXCLUDE_VALUE")
	})
}

func TestEnvironment_AddLexicalScope(t *testing.T) {
	t.Run("adds to a new environment", func(t *testing.T) {
		a := NewKVEnvironment("foo", "bar")
		b := a.AddLexicalScope(map[string]string{"baz": "qux"})

		require.Equal(t, map[string]string{"foo": "bar"}, a.Values())
		require.Equal(t, map[string]string{"foo": "bar", "baz": "qux"}, b.Values())
	})

	t.Run("added lexical scope takes precedence over already added environment", func(t *testing.T) {
		a := NewKVEnvironment("foo", "bar")
		b := a.AddLexicalScope(map[string]string{"foo": "qux"})

		require.Equal(t, map[string]string{"foo": "qux"}, b.Values())
	})

	t.Run("does not add scope if there are no vars", func(t *testing.T) {
		a := NewKVEnvironment("foo", "bar")
		b := a.AddLexicalScope(map[string]string{})

		require.Same(t, a, b)
	})
}

func TestEnvironment_Mutations(t *testing.T) {
	t.Run("mutations have higher precedence than initial values", func(t *testing.T) {
		env := NewKVEnvironment("foo", "bar")
		env.Mutate(NewKVEnvironment("foo", "baz", "ping", "pop"))
		env.Mutate(NewKVEnvironment("foo", "bap"))

		require.Equal(t, map[string]string{"foo": "bap", "ping": "pop"}, env.Values())
	})

	t.Run("child environments accesses mutated values", func(t *testing.T) {
		grandparent := NewKVEnvironment("a", "a_value")
		parent := grandparent.AddLexicalScope(map[string]string{"b": "b_value"})
		child := parent.AddLexicalScope(map[string]string{"c": "c_value"})

		require.Equal(t, map[string]string{"a": "a_value"}, grandparent.Values())
		require.Equal(t, map[string]string{"a": "a_value", "b": "b_value"}, parent.Values())
		require.Equal(t, map[string]string{"a": "a_value", "b": "b_value", "c": "c_value"}, child.Values())

		parent.Mutate(NewKVEnvironment("b", "new_b_value"))

		require.Equal(t, map[string]string{"a": "a_value"}, grandparent.Values())
		require.Equal(t, map[string]string{"a": "a_value", "b": "new_b_value"}, parent.Values())
		require.Equal(t, map[string]string{"a": "a_value", "b": "new_b_value", "c": "c_value"}, child.Values())
	})
}
