package internal_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-builtins/oci/publish/internal"
	"gitlab.com/gitlab-org/step-builtins/oci/publish/internal/testutil/bldr"
)

func TestParseInputs(t *testing.T) {
	t.Run("common artifacts", func(t *testing.T) {
		t.Run("parses files", func(t *testing.T) {
			commonJSON := `{"files": {"step.yml": "step.yml", " files/templates ": " /templates "}}`
			inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithCommon(commonJSON).Build())
			require.NoError(t, err)
			require.Len(t, inputs.Common, 2)
			require.Equal(t, "files/templates", inputs.Common[0].Src)
			require.Equal(t, "/templates", inputs.Common[0].Dst)
			require.Equal(t, "step.yml", inputs.Common[1].Src)
			require.Equal(t, "step.yml", inputs.Common[1].Dst)
		})

		t.Run("trims space", func(t *testing.T) {
			inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithCommon(`{"files": {"  step.yml  ": "  step.yml  "}}`).Build())
			require.NoError(t, err)
			require.Len(t, inputs.Common, 1)
			require.Equal(t, "step.yml", inputs.Common[0].Src)
			require.Equal(t, "step.yml", inputs.Common[0].Dst)
		})

		t.Run("can be empty", func(t *testing.T) {
			inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithCommon(`{}`).Build())
			require.NoError(t, err)
			require.Len(t, inputs.Common, 0)
		})

		t.Run("can have no files", func(t *testing.T) {
			inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithCommon(`{"files": {}}`).Build())
			require.NoError(t, err)
			require.Len(t, inputs.Common, 0)
		})

		t.Run("errors", func(t *testing.T) {
			tests := []struct {
				name       string
				commonJSON string
				expectErr  string
			}{
				{
					name:       "invalid JSON",
					commonJSON: `{`,
					expectErr:  "common input: unexpected end of JSON input",
				},
				{
					name:       "keys other than files",
					commonJSON: `{"filess": {"step.yml": "step.yml"}}`,
					expectErr:  `common input: json: unknown field "filess"`,
				},
				{
					name:       "empty src path",
					commonJSON: `{"files": {"": "step.yml"}}`,
					expectErr:  `common input: empty source path: "": "step.yml"`,
				},
				{
					name:       "empty dst path",
					commonJSON: `{"files": {"step.yml": ""}}`,
					expectErr:  `common input: empty destination path: "step.yml": ""`,
				},
				{
					name:       "includes variant",
					commonJSON: `{"variant": "v7", "files": {"step.yml": "step.yml"}}`,
					expectErr:  `common input: json: unknown field "variant"`,
				},
				{
					name:       "includes os.features",
					commonJSON: `{"os.features": "win32k", "files": {"step.yml": "step.yml"}}`,
					expectErr:  `common input: json: unknown field "os.features"`,
				},
				{
					name:       "includes os.version",
					commonJSON: `{"os.version": "10.0.26100.3476", "files": {"step.yml": "step.yml"}}`,
					expectErr:  `common input: json: unknown field "os.version"`,
				},
				{
					name:       "includes features",
					commonJSON: `{"features": "gpu", "files": {"step.yml": "step.yml"}}`,
					expectErr:  `common input: json: unknown field "features"`,
				},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					_, err := internal.ParseInputs(bldr.CLIInputs(t).WithCommon(test.commonJSON).Build())
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				})
			}
		})
	})

	t.Run("platform artifacts", func(t *testing.T) {
		t.Run("parses platform", func(t *testing.T) {
			platformsJSON := `{"linux/amd64": {"files": {"my_program": "run"}}}`
			inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithPlatforms(platformsJSON).Build())
			require.NoError(t, err)
			require.Len(t, inputs.PlatformSpecific, 1)
			require.Equal(t, "linux", inputs.PlatformSpecific[0].Platform.OS)
			require.Equal(t, "amd64", inputs.PlatformSpecific[0].Platform.Architecture)
			require.Empty(t, inputs.PlatformSpecific[0].Platform.Variant)
			require.Empty(t, inputs.PlatformSpecific[0].Platform.OSVersion)
			require.Empty(t, inputs.PlatformSpecific[0].Platform.OSFeatures)
			require.Empty(t, inputs.PlatformSpecific[0].Platform.Features)
			require.Equal(t, "my_program", inputs.PlatformSpecific[0].Src)
			require.Equal(t, "run", inputs.PlatformSpecific[0].Dst)
		})

		t.Run("parses optional platform details", func(t *testing.T) {
			platformsJSON := `{
"windows/arm64": {
  "variant": "v7", 
  "os.version": "10.0.26100.3476", 
  "os.features": ["win32k"], 
  "features": ["gpu"], 
  "files": {"step.yml": "step.yml"}
}}`
			inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithPlatforms(platformsJSON).Build())
			require.NoError(t, err)
			require.Len(t, inputs.PlatformSpecific, 1)
			require.Equal(t, "windows", inputs.PlatformSpecific[0].Platform.OS)
			require.Equal(t, "arm64", inputs.PlatformSpecific[0].Platform.Architecture)
			require.Equal(t, "v7", inputs.PlatformSpecific[0].Platform.Variant)
			require.Equal(t, "10.0.26100.3476", inputs.PlatformSpecific[0].Platform.OSVersion)
			require.Equal(t, []string{"win32k"}, inputs.PlatformSpecific[0].Platform.OSFeatures)
			require.Equal(t, []string{"gpu"}, inputs.PlatformSpecific[0].Platform.Features)
		})

		t.Run("trims space", func(t *testing.T) {
			platformsJSON := `{" linux / amd64 ": {"variant":" v8 ", "files": {"my_program": "run"}}}`
			inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithPlatforms(platformsJSON).Build())
			require.NoError(t, err)
			require.Len(t, inputs.PlatformSpecific, 1)
			require.Equal(t, "linux", inputs.PlatformSpecific[0].Platform.OS)
			require.Equal(t, "amd64", inputs.PlatformSpecific[0].Platform.Architecture)
			require.Equal(t, "v8", inputs.PlatformSpecific[0].Platform.Variant)
		})

		t.Run("parses many platforms", func(t *testing.T) {
			platformsJSON := `{"linux/arm64": {"files": {"amd_run": "run"}}, "linux/amd64": {"files": {"arm_run": "run"}}}`
			inputs, err := internal.ParseInputs(bldr.CLIInputs(t).WithPlatforms(platformsJSON).Build())
			require.NoError(t, err)
			require.Len(t, inputs.PlatformSpecific, 2)
		})

		t.Run("errors", func(t *testing.T) {
			tests := []struct {
				name          string
				platformsJSON string
				expectErr     string
			}{
				{
					name:          "invalid JSON",
					platformsJSON: `{`,
					expectErr:     "platforms input: unexpected end of JSON input",
				},
				{
					name:          "no platforms",
					platformsJSON: `{}`,
					expectErr:     `platforms input: must have at least one platform`,
				},
				{
					name:          "missing architecture",
					platformsJSON: `{"linux": {"files": {"step.yml": "step.yml"}}}`,
					expectErr:     `platforms input: invalid platform os/arch: linux`,
				},
				{
					name:          "too many slashes",
					platformsJSON: `{"linux/amd64/v7": {"files": {"step.yml": "step.yml"}}}`,
					expectErr:     `platforms input: invalid platform os/arch: linux/amd64/v7`,
				},
				{
					name:          "keys other than files",
					platformsJSON: `{"linux/amd64": {"filess": {"step.yml": "step.yml"}}}`,
					expectErr:     `platforms input: json: unknown field "filess"`,
				},
				{
					name:          "same platform defined more than once",
					platformsJSON: `{"linux/amd64": {"files": {"step.yml": "step.yml"}}, "  linux/amd64  ": {"files": {"step.yml": "step.yml"}}}`,
					expectErr:     `platform "linux/amd64" defined more than once`,
				},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					_, err := internal.ParseInputs(bldr.CLIInputs(t).WithPlatforms(test.platformsJSON).Build())
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				})
			}
		})
	})

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
}

func TestInputs_ImgRef(t *testing.T) {
	tests := []struct {
		name       string
		registry   string
		repository string
		tag        string
		expect     string
	}{
		{
			name:       "combined registry, repository, and tag",
			registry:   "gitlab.registry.com",
			repository: "my_group/my_project",
			tag:        "3.4.0",
			expect:     "gitlab.registry.com/my_group/my_project:3.4.0",
		},
		{
			name:       "removes unnecessary forward slashes",
			registry:   "gitlab.registry.com/",
			repository: "/my_project/",
			tag:        "1.0.0-rc1",
			expect:     "gitlab.registry.com/my_project:1.0.0-rc1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			build, env := bldr.CLIInputs(t).WithRegistry(test.registry).WithRepository(test.repository).WithTag(test.tag).Build()
			inputs, err := internal.ParseInputs(build, env)
			require.NoError(t, err)
			require.Equal(t, test.expect, inputs.RemoteImageRef.MajorMinorPatch().Name())
		})
	}
}
