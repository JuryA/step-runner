package ci

import (
	"os"
	"testing"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestCICmd(t *testing.T) {
	stepResultsFile := "step-results.json"
	t.Run("runs steps", func(t *testing.T) {
		steps := `
- name: secret_factory_a
  step: ../../e2e_tests/test_steps/secret_factory
- name: secret_factory_b
  step: ../../e2e_tests/test_steps/secret_factory
  inputs:
    secret_override: ${{ steps.secret_factory_a.outputs.secret }}
`
		require.NoError(t, os.Setenv("STEPS", steps))
		defer func() { _ = os.Unsetenv("STEPS") }()

		cmd := NewCmd()
		cmd.SetArgs([]string{"--write-steps-results"})
		err := cmd.Execute()
		require.NoError(t, err)

		file, err := os.ReadFile(stepResultsFile)
		require.NoError(t, err)

		var result proto.StepResult
		err = protojson.Unmarshal(file, &result)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, result.Status)

		require.Len(t, result.SubStepResults, 2)
		require.Equal(t, result.SubStepResults[0].Outputs["secret"], result.SubStepResults[1].Outputs["secret"])
	})

	t.Run("generates step-results file", func(t *testing.T) {
		tests := []struct {
			name             string
			runInDebugMode   bool
			args             []string
			expectFileExists bool
		}{
			{
				name:             "generates step file when CLI arg used",
				runInDebugMode:   false,
				args:             []string{"--write-steps-results"},
				expectFileExists: true,
			},
			{
				name:             "generates step file when env variable set",
				runInDebugMode:   true,
				args:             []string{},
				expectFileExists: true,
			},
			{
				name:             "does not generate step file when env variable not set and CLI arg not used",
				runInDebugMode:   false,
				args:             []string{},
				expectFileExists: false,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				_ = os.Remove(stepResultsFile)

				require.NoError(t, os.Setenv("STEPS", `- step: ../../e2e_tests/test_steps/secret_factory`))
				defer func() { _ = os.Unsetenv("STEPS") }()

				beforeValue := runner.RunningInDebugMode
				defer func() { runner.RunningInDebugMode = beforeValue }()
				runner.RunningInDebugMode = test.runInDebugMode

				cmd := NewCmd()
				cmd.SetArgs(test.args)
				err := cmd.Execute()
				require.NoError(t, err)

				if test.expectFileExists {
					require.FileExists(t, stepResultsFile)
				} else {
					require.NoFileExists(t, stepResultsFile)
				}
			})
		}
	})

	t.Run("can access environment variables", func(t *testing.T) {
		steps := `
- name: echo
  step: ../../e2e_tests/test_steps/echo
  inputs:
    echo: env value is ${{env.LOGNAME}}
`
		require.NoError(t, os.Setenv("STEPS", steps))
		require.NoError(t, os.Setenv("LOGNAME", "test.user"))
		defer func() { _ = os.Unsetenv("LOGNAME") }()
		defer func() { _ = os.Unsetenv("STEPS") }()

		cmd := NewCmd()
		cmd.SetArgs([]string{"--write-steps-results"})
		err := cmd.Execute()
		require.NoError(t, err)

		file, err := os.ReadFile(stepResultsFile)
		require.NoError(t, err)

		var result proto.StepResult
		err = protojson.Unmarshal(file, &result)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, result.Status)

		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, "env value is test.user", result.SubStepResults[0].Outputs["echo"].GetStringValue())
	})

	t.Run("failed step returns error", func(t *testing.T) {
		steps := `
- name: exit
  step: ../../e2e_tests/test_steps/exit
  inputs:
    exit_code: 99
`
		require.NoError(t, os.Setenv("STEPS", steps))
		defer func() { _ = os.Unsetenv("STEPS") }()

		cmd := NewCmd()
		cmd.SetArgs([]string{"--write-steps-results"})
		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "exec: exit status 99")

		file, err := os.ReadFile(stepResultsFile)
		require.NoError(t, err)

		var result proto.StepResult
		err = protojson.Unmarshal(file, &result)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_failure, result.Status)
	})
}
