package runner_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/di"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestStepResourceParser_Parse(t *testing.T) {
	t.Run("parses resource type", func(t *testing.T) {
		tests := []struct {
			name         string
			stepRef      *proto.Step_Reference
			expectedType any
		}{
			{
				name: "git",
				stepRef: &proto.Step_Reference{
					Protocol: proto.StepReferenceProtocol_git,
					Url:      "gitlab.com/user/repo",
					Path:     []string{},
					Filename: "step.yml",
					Version:  "main",
				},
				expectedType: (*runner.GitStepResource)(nil),
			},
			{
				name: "oci",
				stepRef: &proto.Step_Reference{
					Protocol:   proto.StepReferenceProtocol_oci,
					Registry:   "registry.gitlab.com",
					Repository: "project/my-repository",
					Tag:        "latest",
					Path:       []string{},
					Filename:   "step.yml",
				},
				expectedType: (*runner.OCIStepResource)(nil),
			},
			{
				name: "local",
				stepRef: &proto.Step_Reference{
					Protocol: proto.StepReferenceProtocol_local,
					Path:     []string{"path", "to", "step_dir"},
					Filename: "step.yml",
				},
				expectedType: (*runner.FileSystemStepResource)(nil),
			},
			{
				name: "dist",
				stepRef: &proto.Step_Reference{
					Protocol: proto.StepReferenceProtocol_dist,
					Path:     []string{"path", "to", "dist", "step"},
					Filename: "step.yml",
				},
				expectedType: (*runner.DistStepResource)(nil),
			},
			{
				name: "dynamic",
				stepRef: &proto.Step_Reference{
					Protocol: proto.StepReferenceProtocol_dynamic,
					Url:      "${{job.VARIABLE}}",
				},
				expectedType: (*runner.DynamicStepResource)(nil),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				parser, err := di.NewContainer().StepResourceParser()
				require.NoError(t, err)

				stepResource, err := parser.Parse("/path/to/step", test.stepRef)
				require.NoError(t, err)

				expectedType := reflect.TypeOf(test.expectedType)
				actualType := reflect.TypeOf(stepResource)
				require.Equal(t, expectedType, actualType)
			})
		}
	})
}

func TestStepResourceParser_ParseLocalStep(t *testing.T) {
	t.Run("loads relative path", func(t *testing.T) {
		parser, err := di.NewContainer().StepResourceParser()
		require.NoError(t, err)

		stepRef := &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_local,
			Path:     []string{"path", "to", "step_dir"},
			Filename: "step.yml",
		}

		stepResource, err := parser.Parse("/parent/dir", stepRef)
		require.NoError(t, err)

		description := stepResource.(*runner.FileSystemStepResource).Describe()
		require.Equal(t, "/parent/dir/path/to/step_dir/step.yml", description)
	})

	t.Run("loads absolute path", func(t *testing.T) {
		parser, err := di.NewContainer().StepResourceParser()
		require.NoError(t, err)

		stepRef := &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_local,
			Path:     []string{"/", "path", "to", "step_dir"},
			Filename: "step.yml",
		}

		stepResource, err := parser.Parse("/parent/dir", stepRef)
		require.NoError(t, err)

		description := stepResource.(*runner.FileSystemStepResource).Describe()
		require.Equal(t, "/path/to/step_dir/step.yml", description)
	})
}
