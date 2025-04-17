package internal_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/dist-steps/oci/fetch/internal"
	"gitlab.com/gitlab-org/dist-steps/oci/fetch/internal/testutil/bldr"
)

func TestParseInputs(t *testing.T) {
	t.Run("parses log level", func(t *testing.T) {
		tests := []struct {
			logLevel       string
			expectLogLevel slog.Level
		}{
			{
				logLevel:       "info",
				expectLogLevel: slog.LevelInfo,
			},
			{
				logLevel:       "debug",
				expectLogLevel: slog.LevelDebug,
			},
			{
				logLevel:       "warn",
				expectLogLevel: slog.LevelWarn,
			},
			{
				logLevel:       "error",
				expectLogLevel: slog.LevelError,
			},
		}

		for _, test := range tests {
			t.Run(test.logLevel, func(t *testing.T) {
				inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithLogLevel(test.logLevel).Build())
				require.NoError(t, err)
				require.Equal(t, test.expectLogLevel, inputs.LogLevel)
			})
		}
	})

	t.Run("registry", func(t *testing.T) {
		tests := []struct {
			name      string
			registry  string
			expect    string
			expectErr string
		}{
			{
				name:     "parses",
				registry: "registry.gitlab.com:5000",
				expect:   "registry.gitlab.com:5000",
			},
			{
				name:     "trims space",
				registry: "  registry.gitlab.com  ",
				expect:   "registry.gitlab.com",
			},
			{
				name:      "cannot be empty",
				registry:  "",
				expectErr: "registry is required",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithRegistry(test.registry).Build())

				if test.expectErr == "" {
					require.NoError(t, err)
					require.Equal(t, test.expect, inputs.RemoteImageRef.Context().RegistryStr())
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				}
			})
		}
	})

	t.Run("repository", func(t *testing.T) {
		tests := []struct {
			name       string
			repository string
			expect     string
			expectErr  string
		}{
			{
				name:       "parses",
				repository: "my_group/my_project/image",
				expect:     "my_group/my_project/image",
			},
			{
				name:       "trims space",
				repository: "  my_group/my_project/image  ",
				expect:     "my_group/my_project/image",
			},
			{
				name:       "cannot be empty",
				repository: "",
				expectErr:  "repository is required",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithRepository(test.repository).Build())

				if test.expectErr == "" {
					require.NoError(t, err)
					require.Equal(t, test.expect, inputs.RemoteImageRef.Context().RepositoryStr())
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				}
			})
		}
	})

	t.Run("tag", func(t *testing.T) {
		tests := []struct {
			name      string
			tag       string
			expect    string
			expectErr string
		}{
			{
				name:   "parses",
				tag:    "1.0.3",
				expect: "1.0.3",
			},
			{
				name:   "trims space",
				tag:    "  12.44.32  ",
				expect: "12.44.32",
			},
			{
				name:      "cannot be empty",
				tag:       "",
				expectErr: "tag is required",
			},
			{
				name:   "supports using a digest",
				tag:    "sha256:47bbdb084c81247335fa9838c5536df7ea1011e0ffe6b7706abbd7658917d296",
				expect: "sha256:47bbdb084c81247335fa9838c5536df7ea1011e0ffe6b7706abbd7658917d296",
			},
			{
				name:   "does not need to conform to semver",
				tag:    "16.334.abc.01-01-2009",
				expect: "16.334.abc.01-01-2009",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithTag(test.tag).Build())

				if test.expectErr == "" {
					require.NoError(t, err)
					require.Equal(t, test.expect, inputs.RemoteImageRef.Identifier())
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				}
			})
		}
	})

	t.Run("step path", func(t *testing.T) {
		tests := []struct {
			name     string
			stepPath string
			expect   string
		}{
			{
				name:     "trims space",
				stepPath: "    path/to/mystep   ",
				expect:   "path/to/mystep",
			},
			{
				name:     "removes additional path separators",
				stepPath: "/path//to/step",
				expect:   "/path/to/step",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithStepPath(test.stepPath).Build())
				require.NoError(t, err)
				require.Equal(t, test.expect, inputs.StepPath)
			})
		}
	})
}

func TestParseInputs_RemoteImageReference(t *testing.T) {
	tests := []struct {
		name       string
		registry   string
		repository string
		tag        string
		expect     string
		expectErr  string
	}{
		{
			name:       "registry, repository, and tag",
			registry:   "registry.gitlab.com",
			repository: "group/project",
			tag:        "1.0.0",
			expect:     "registry.gitlab.com/group/project:1.0.0",
		},
		{
			name:       "removes extra slashes",
			registry:   "registry.gitlab.com//",
			repository: "/group/project/",
			tag:        "latest",
			expect:     "registry.gitlab.com/group/project:latest",
		},
		{
			name:       "registry with port",
			registry:   "registry.gitlab.com:8080",
			repository: "project",
			tag:        "latest",
			expect:     "registry.gitlab.com:8080/project:latest",
		},
		{
			name:       "invalid registry",
			registry:   "registry.gitlab.com/!",
			repository: "project",
			tag:        "latest",
			expectErr:  "could not parse reference: registry.gitlab.com/!/project:latest",
		},
		{
			name:       "invalid tag",
			registry:   "registry.gitlab.com",
			repository: "project",
			tag:        "!err!",
			expectErr:  "could not parse reference: registry.gitlab.com/project:!err!",
		},
		{
			name:       "registry, repository, and digest",
			registry:   "registry.gitlab.com",
			repository: "project",
			tag:        "sha256:f271d3fd90442470614813bd422ad3c1a8286e79904ba4faeca94a3fd0fb5b24",
			expect:     "registry.gitlab.com/project@sha256:f271d3fd90442470614813bd422ad3c1a8286e79904ba4faeca94a3fd0fb5b24",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cliArgs, env := bldr.CLIInputs(t).WithRegistry(test.registry).WithRepository(test.repository).WithTag(test.tag).Build()
			inputs, err := internal.ParseInputs(cliArgs, env)

			if test.expectErr == "" {
				require.NoError(t, err)
				require.Equal(t, test.expect, inputs.RemoteImageRef.String())
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectErr)
			}
		})
	}
}
