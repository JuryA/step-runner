package schema

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestReferenceCustomMethods(t *testing.T) {
	myStep := "my_step"

	cases := []struct {
		name    string
		json    string
		yaml    string
		wantRef any
	}{{
		name:    "short reference",
		json:    `{"name":"my_step","step":"gitlab.com/components/script@v1"}`,
		yaml:    `{name: my_step, step: gitlab.com/components/script@v1}`,
		wantRef: "gitlab.com/components/script@v1",
	}, {
		name: "long simple git reference",
		json: `
{
  "name": "my_step",
  "step": {
    "git": {
      "url":"gitlab.com/components/script",
      "rev":"v1"
    }
  }
}
`,
		yaml: `
name: my_step
step:
  git:
    url:    gitlab.com/components/script
    rev: v1
`,
		wantRef: &Reference{Git: NewGitReference("gitlab.com/components/script", "v1")},
	}, {
		name: "long git reference with dir",
		json: `
{
  "name": "my_step",
  "step": {
    "git": {
      "url":"gitlab.com/components/script",
      "dir":"bash",
      "rev":"v1"
    }
  }
}
`,
		yaml: `
name: my_step
step:
  git:
    url:    gitlab.com/components/script
    dir:    bash
    rev: v1
`,
		wantRef: &Reference{
			Git: NewGitReference("gitlab.com/components/script", "v1", GitRefDir("bash")),
		},
	}, {
		name: "long one-line git reference with dir",
		json: `{"name":"my_step","step":{"git":{"url":"gitlab.com/components/script","dir":"bash","rev":"v1"}}}`,
		yaml: `{name: my_step, step: {git: {url: gitlab.com/components/script, dir: bash, rev: v1}}}`,
		wantRef: &Reference{
			Git: NewGitReference("gitlab.com/components/script", "v1", GitRefDir("bash")),
		},
	}, {
		name: "oci reference",
		json: `
{
  "name": "my_step",
  "step": {
    "oci": {
      "registry":"registry.gitlab.com",
      "repository":"project/my-repository",
      "tag":"latest"
    }
  }
}
`,
		yaml: `
name: my_step
step:
  oci:
    registry: registry.gitlab.com
    repository: project/my-repository
    tag: latest
`,
		wantRef: &Reference{OCI: NewOCIReference("registry.gitlab.com", "project/my-repository", "latest")},
	}}

	data, err := os.ReadFile("step.json")
	require.NoError(t, err)

	stepsSchema := jsonschema.MustCompileString("step.json", string(data))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			step := Step{
				Name: &myStep,
				Step: tc.wantRef,
			}

			switch v := tc.wantRef.(type) {
			case string:
				step.Step = v
				check(t, json.Marshal, json.Unmarshal, []byte(tc.json), step, stepsSchema)
				check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), step, stepsSchema)
			case *Reference:
				step.Step = v
				check(t, json.Marshal, json.Unmarshal, []byte(tc.json), step, stepsSchema)
				check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), step, stepsSchema)
			}
		})
	}
}

func TestReferenceValidation(t *testing.T) {
	cases := []struct {
		name    string
		json    string
		wantErr string
	}{{
		name:    "supplying neither git or oci",
		json:    `{}`,
		wantErr: "field git: or oci: required",
	}, {
		name: "supplying both git and oci",
		json: `
{
  "git": {
    "url":"gitlab.com/components/script",
    "rev":"v1"
  },
  "oci": {
    "url":"registry.gitlab.com/components/my-program",
    "tag":"1.0.0"
  }
}
`,
		wantErr: "cannot use both git: and oci: fields, please specify only one step location",
	}}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var ref Reference
			err := json.Unmarshal([]byte(test.json), &ref)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func TestCompileOCI(t *testing.T) {
	t.Run("compiles OCI reference", func(t *testing.T) {
		ref := &Reference{OCI: NewOCIReference("registry.gitlab.com", "project/my-repository", "latest")}

		stepRef, err := ref.compile("my_step", map[string]*structpb.Value{}, map[string]string{})
		require.NoError(t, err)
		require.Equal(t, proto.StepReferenceProtocol_spec_def, stepRef.Protocol)

		steps := stepRef.SpecDef.Definition.Steps
		require.Len(t, steps, 2)

		// nolint:staticcheck // SA1019
		require.Equal(t, []string{"oci", "fetch"}, steps[0].Step.Path)
		require.Equal(t, "registry.gitlab.com", steps[0].Inputs["registry"].GetStringValue())
		require.Equal(t, "project/my-repository", steps[0].Inputs["repository"].GetStringValue())
		require.Equal(t, "latest", steps[0].Inputs["tag"].GetStringValue())
		require.Equal(t, "my_step", steps[1].Name)
		require.Equal(t, "${{steps.fetch_step_my_step.outputs.fetched_step_path}}", steps[1].Step.StepPath.(*proto.Step_Reference_PathExp).PathExp)
	})
}

func TestCompileGit_InternalFolder(t *testing.T) {
	tests := []struct {
		name    string
		ref     *Reference
		isValid bool
	}{
		{
			name:    "valid git reference",
			ref:     &Reference{Git: NewGitReference("gitlab.com/components/script", "v1")},
			isValid: true,
		},
		{
			name:    "valid git reference with subdirectory",
			ref:     &Reference{Git: NewGitReference("gitlab.com/components/script", "v1", GitRefDir("subdir"))},
			isValid: true,
		},
		{
			name:    "invalid git reference to internal folder",
			ref:     &Reference{Git: NewGitReference("gitlab.com/components/script", "v1", GitRefDir("internal"))},
			isValid: false,
		},
		{
			name:    "invalid git reference to nested internal folder",
			ref:     &Reference{Git: NewGitReference("gitlab.com/components/script", "v1", GitRefDir("subdir/internal"))},
			isValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stepRef, err := test.ref.compileGit()
			if test.isValid {
				require.NoError(t, err)
				require.NotNil(t, stepRef)
				require.Equal(t, proto.StepReferenceProtocol_git, stepRef.Protocol)
			} else {
				require.Error(t, err)
				require.Nil(t, stepRef)
				require.Contains(t, err.Error(), "steps inside folders named 'internal' cannot be accessed directly")
			}
		})
	}
}

func TestCompileOCI_InternalFolder(t *testing.T) {
	tests := []struct {
		name    string
		ref     *Reference
		isValid bool
	}{
		{
			name:    "valid OCI reference without directory",
			ref:     &Reference{OCI: NewOCIReference("registry.gitlab.com", "project/my-repository", "latest")},
			isValid: true,
		},
		{
			name:    "valid OCI reference with subdirectory",
			ref:     &Reference{OCI: &OCIReference{Registry: "registry.gitlab.com", Repository: "project/my-repository", Tag: "latest", Dir: ptr("subdir")}},
			isValid: true,
		},
		{
			name:    "invalid OCI reference to internal folder",
			ref:     &Reference{OCI: &OCIReference{Registry: "registry.gitlab.com", Repository: "project/my-repository", Tag: "latest", Dir: ptr("internal")}},
			isValid: false,
		},
		{
			name:    "invalid OCI reference to nested internal folder",
			ref:     &Reference{OCI: &OCIReference{Registry: "registry.gitlab.com", Repository: "project/my-repository", Tag: "latest", Dir: ptr("subdir/internal")}},
			isValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stepRef, err := test.ref.compileOCI("my_step", map[string]*structpb.Value{}, map[string]string{})
			if test.isValid {
				require.NoError(t, err)
				require.NotNil(t, stepRef)
				require.Equal(t, proto.StepReferenceProtocol_spec_def, stepRef.Protocol)
			} else {
				require.Error(t, err)
				require.Nil(t, stepRef)
				require.Contains(t, err.Error(), "steps inside folders named 'internal' cannot be accessed directly")
			}
		})
	}
}

// Helper function to create string pointers
func ptr(s string) *string {
	return &s
}
