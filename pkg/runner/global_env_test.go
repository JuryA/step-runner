package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobalEnvironmentLogging(t *testing.T) {
	t.Run("set as job variable", func(t *testing.T) {
		tests := []struct {
			name          string
			setting       map[string]string
			expectedLevel string
		}{
			{
				name:          "log level set to debug",
				setting:       map[string]string{LogLevelEnvName: "debug"},
				expectedLevel: "debug",
			},
			{
				name:          "log level set to info",
				setting:       map[string]string{LogLevelEnvName: "info"},
				expectedLevel: "info",
			},
			{
				name:          "log level set to warn",
				setting:       map[string]string{LogLevelEnvName: "warn"},
				expectedLevel: "warn",
			},
			{
				name:          "log level set to error",
				setting:       map[string]string{LogLevelEnvName: "error"},
				expectedLevel: "error",
			},
			{
				name:          "case of value does not matter",
				setting:       map[string]string{LogLevelEnvName: "InFo"},
				expectedLevel: "info",
			},
			{
				name:          "whitespace is removed",
				setting:       map[string]string{LogLevelEnvName: "  debug  "},
				expectedLevel: "debug",
			},
			{
				name:          "info if not set",
				setting:       map[string]string{},
				expectedLevel: "info",
			},
		}

		for _, test := range tests {
			t.Run("job variable "+test.name, func(t *testing.T) {
				env, err := GlobalEnvironment(NewEmptyEnvironment(), test.setting)
				require.NoError(t, err)
				require.Equal(t, test.expectedLevel, env.ValueOf(LogLevelEnvName))
			})

			t.Run("parent environment "+test.name, func(t *testing.T) {
				env, err := GlobalEnvironment(NewEnvironment(test.setting), map[string]string{})
				require.NoError(t, err)
				require.Equal(t, test.expectedLevel, env.ValueOf(LogLevelEnvName))
			})
		}
	})

	t.Run("job setting overrides parent environment setting", func(t *testing.T) {
		parent := NewEnvironment(map[string]string{LogLevelEnvName: "debug"})
		globalEnv, err := GlobalEnvironment(parent, map[string]string{LogLevelEnvName: "warn"})
		require.NoError(t, err)
		require.Equal(t, "warn", globalEnv.ValueOf(LogLevelEnvName))
	})

	t.Run("set to info when value is not set", func(t *testing.T) {
		globalEnv, err := GlobalEnvironment(NewEmptyEnvironment(), map[string]string{})
		require.NoError(t, err)
		require.Equal(t, "info", globalEnv.ValueOf(LogLevelEnvName))
	})

	t.Run("invalid log level", func(t *testing.T) {
		t.Run("when set by parent environment", func(t *testing.T) {
			parent := NewEnvironment(map[string]string{LogLevelEnvName: "XXX"})
			_, err := GlobalEnvironment(parent, map[string]string{})
			require.Error(t, err)
			require.Contains(t, err.Error(), "init global environment: log level: xxx not supported")
		})

		t.Run("when set by job variable", func(t *testing.T) {
			_, err := GlobalEnvironment(NewEmptyEnvironment(), map[string]string{LogLevelEnvName: "XXX"})
			require.Error(t, err)
			require.Contains(t, err.Error(), "init global environment: log level: xxx not supported")
		})
	})
}
