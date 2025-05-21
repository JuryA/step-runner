package internal_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/promote/internal"
	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/promote/testutil/bldr"
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
				inputs, err := internal.ParseInputs(bldr.CLIInputs().WithLogLevel(test.logLevel).Build())
				require.NoError(t, err)
				require.Equal(t, test.expectLogLevel, inputs.LogLevel)
			})
		}
	})

	t.Run("parses from image reference", func(t *testing.T) {
		tests := []struct {
			name      string
			image     string
			expect    string
			expectErr string
		}{
			{
				name:   "parses reference with tag",
				image:  "registry.gitlab.com/my-group/my-image:latest",
				expect: "registry.gitlab.com/my-group/my-image:latest",
			},
			{
				name:   "parses reference with digest",
				image:  "registry.gitlab.com/my-group/my-image@sha256:6219b35369a22ed1c57ffb6227f85884dd98d707542df474ae10aec228cb9a43",
				expect: "registry.gitlab.com/my-group/my-image@sha256:6219b35369a22ed1c57ffb6227f85884dd98d707542df474ae10aec228cb9a43",
			},
			{
				name:   "trims space",
				image:  "    registry.gitlab.com/my-group/my-image:latest   ",
				expect: "registry.gitlab.com/my-group/my-image:latest",
			},
			{
				name:      "cannot be empty",
				image:     "  ",
				expectErr: "from image is required",
			},
			{
				name:      "must be a valid image reference",
				image:     "registry.gitlab@com/image:1",
				expectErr: "from image: could not parse reference: registry.gitlab@com/image:1",
			},
			{
				name:      "must specify a tag or a digest",
				image:     "registry.gitlab.com/image",
				expectErr: "from image: must specify tag or digest",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				inputs, err := internal.ParseInputs(bldr.CLIInputs().WithFromImage(test.image).Build())

				if test.expectErr == "" {
					require.NoError(t, err)
					require.Equal(t, test.expect, inputs.FromImageRef.String())
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				}
			})
		}
	})

	t.Run("parses to image", func(t *testing.T) {
		cliInputs, env := bldr.CLIInputs().
			WithToRegistry("registry.gitlab.com").
			WithToRepository("my-group/image").
			WithToVersion("2.5.6").
			Build()

		inputs, err := internal.ParseInputs(cliInputs, env)

		require.NoError(t, err)
		require.Equal(t, "registry.gitlab.com/my-group/image:2.5.6", inputs.ToImageRef.String())
	})
}
