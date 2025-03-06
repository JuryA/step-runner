package pkg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/builtin/steps/oci/publish/pkg/testutil/bldr"
)

func TestParseInputs(t *testing.T) {
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
				inputs, err := ParseInputs(bldr.CLIInputs().WithRegistry(test.registry).Build())
				if test.expectErr == "" {
					require.NoError(t, err)
					require.Equal(t, test.expect, inputs.Registry)
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
				inputs, err := ParseInputs(bldr.CLIInputs().WithRepository(test.repository).Build())
				if test.expectErr == "" {
					require.NoError(t, err)
					require.Equal(t, test.expect, inputs.Repository)
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
				name:      "tag must be semver including patch",
				tag:       "latest",
				expectErr: `tag input: "latest" does not conform to semantic versioning MAJOR.MINOR.PATCH[-release]`,
			},
			{
				name:      "tag cannot be major and minor only",
				tag:       "2.0",
				expectErr: `tag input: "2.0" does not conform to semantic versioning MAJOR.MINOR.PATCH[-release]`,
			},
			{
				name:   "tag can include release candidate",
				tag:    "2.0.0-rc1",
				expect: "2.0.0-rc1",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				inputs, err := ParseInputs(bldr.CLIInputs().WithTag(test.tag).Build())
				if test.expectErr == "" {
					require.NoError(t, err)
					require.Equal(t, test.expect, inputs.Tag)
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				}
			})
		}
	})

	t.Run("common artifacts", func(t *testing.T) {
		t.Run("parses files", func(t *testing.T) {
			commonJSON := `{"files": {"step.yml": "step.yml", " files/templates ": " /templates "}}`
			inputs, err := ParseInputs(bldr.CLIInputs().WithCommon(commonJSON).Build())
			require.NoError(t, err)
			require.Len(t, inputs.Common, 2)
			require.Equal(t, "files/templates", inputs.Common[0].Src)
			require.Equal(t, "/templates", inputs.Common[0].Dst)
			require.Equal(t, "step.yml", inputs.Common[1].Src)
			require.Equal(t, "step.yml", inputs.Common[1].Dst)
		})

		t.Run("trims space", func(t *testing.T) {
			inputs, err := ParseInputs(bldr.CLIInputs().WithCommon(`{"files": {"  step.yml  ": "  step.yml  "}}`).Build())
			require.NoError(t, err)
			require.Len(t, inputs.Common, 1)
			require.Equal(t, "step.yml", inputs.Common[0].Src)
			require.Equal(t, "step.yml", inputs.Common[0].Dst)
		})

		t.Run("can be empty", func(t *testing.T) {
			inputs, err := ParseInputs(bldr.CLIInputs().WithCommon(`{}`).Build())
			require.NoError(t, err)
			require.Len(t, inputs.Common, 0)
		})

		t.Run("can have no files", func(t *testing.T) {
			inputs, err := ParseInputs(bldr.CLIInputs().WithCommon(`{"files": {}}`).Build())
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
				}}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					_, err := ParseInputs(bldr.CLIInputs().WithCommon(test.commonJSON).Build())
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				})
			}
		})
	})

	t.Run("platform artifacts", func(t *testing.T) {
		t.Run("parses platform", func(t *testing.T) {
			platformsJSON := `{"linux_amd64": {"files": {"my_program": "run"}}}`
			inputs, err := ParseInputs(bldr.CLIInputs().WithPlatforms(platformsJSON).Build())
			require.NoError(t, err)
			require.Len(t, inputs.PlatformSpecific, 1)
			require.Equal(t, "linux", inputs.PlatformSpecific[0].Platform.OS)
			require.Equal(t, "amd64", inputs.PlatformSpecific[0].Platform.Architecture)
			require.Equal(t, "my_program", inputs.PlatformSpecific[0].Src)
			require.Equal(t, "run", inputs.PlatformSpecific[0].Dst)
		})

		t.Run("trims space", func(t *testing.T) {
			platformsJSON := `{" linux _ amd64 ": {"files": {"my_program": "run"}}}`
			inputs, err := ParseInputs(bldr.CLIInputs().WithPlatforms(platformsJSON).Build())
			require.NoError(t, err)
			require.Len(t, inputs.PlatformSpecific, 1)
			require.Equal(t, "linux", inputs.PlatformSpecific[0].Platform.OS)
			require.Equal(t, "amd64", inputs.PlatformSpecific[0].Platform.Architecture)
		})

		t.Run("parses many platforms", func(t *testing.T) {
			platformsJSON := `{"linux_arm64": {"files": {"amd_run": "run"}}, "linux_amd64": {"files": {"arm_run": "run"}}}`
			inputs, err := ParseInputs(bldr.CLIInputs().WithPlatforms(platformsJSON).Build())
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
					name:          "too many underscores",
					platformsJSON: `{"linux_amd64_v7": {"files": {"step.yml": "step.yml"}}}`,
					expectErr:     `platforms input: invalid platform os/arch: linux_amd64_v7`,
				},
				{
					name:          "keys other than files",
					platformsJSON: `{"linux_amd64": {"filess": {"step.yml": "step.yml"}}}`,
					expectErr:     `platforms input: json: unknown field "filess"`,
				},
				{
					name:          "same platform defined more than once",
					platformsJSON: `{"linux_amd64": {"files": {"step.yml": "step.yml"}}, "  linux_amd64  ": {"files": {"step.yml": "step.yml"}}}`,
					expectErr:     `platform "linux/amd64" defined more than once`,
				},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					_, err := ParseInputs(bldr.CLIInputs().WithPlatforms(test.platformsJSON).Build())
					require.Error(t, err)
					require.Contains(t, err.Error(), test.expectErr)
				})
			}
		})
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
			inputs := &Inputs{
				Registry:   test.registry,
				Repository: test.repository,
				Tag:        test.tag,
			}

			imgRef, err := inputs.ImgRef()
			require.NoError(t, err)
			require.Equal(t, test.expect, imgRef.String())
		})
	}
}
