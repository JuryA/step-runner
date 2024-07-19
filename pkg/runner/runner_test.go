package runner

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestRun(t *testing.T) {
	cases := []runnerTest{{
		name: "greeting with defaults",
		yaml: `
spec: {}
---
steps:
  - name: greet_steppy
    step: ./test_steps/greeting
    inputs: {}
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "steppy", result.SubStepResults[0].Outputs["name"])
			require.Equal(t, "steppy", result.SubStepResults[0].Exports["NAME"])
		},
	}, {
		name: "greeting outputs and exports name parameter",
		yaml: `
spec: {}
---
steps:
  - name: greet_foo
    step: ./test_steps/greeting
    inputs:
      name: foo
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "foo", result.SubStepResults[0].Outputs["name"])
			require.Equal(t, "foo", result.SubStepResults[0].Exports["NAME"])
		},
	}, {
		name: "can access outputs of a previous step",
		yaml: `
spec: {}
---
steps:
  - name: greet_foo
    step: ./test_steps/greeting
    inputs:
      name: foo
  - name: greet_previous
    step: ./test_steps/greeting
    inputs:
      name: ${{steps.greet_foo.outputs.name}}
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 2)
			requireStringEqualValue(t, "foo", result.SubStepResults[0].Outputs["name"])
			requireStringEqualValue(t, "foo", result.SubStepResults[1].Outputs["name"])
		},
	}, {
		name: "can access outputs of a composite step",
		yaml: `
spec: {}
---
steps:
  - name: greet_the_crew
    step: ./test_steps/crew
    inputs: {}
  - name: greet_previous
    step: ./test_steps/greeting
    inputs:
      name: ${{steps.greet_the_crew.outputs.crew_name_1}}
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 2)
			requireStringEqualValue(t, "sponge bob", result.SubStepResults[0].Outputs["crew_name_1"])
			requireStringEqualValue(t, "sponge bob", result.SubStepResults[1].Outputs["name"])
		},
	}, {
		name: "cannot access outputs of composite children",
		yaml: `
spec: {}
---
steps:
  - name: greet_the_crew
    step: ./test_steps/crew
    inputs: {}
  - name: greet_previous
    step: ./test_steps/greeting
    inputs:
      name: ${{steps.greet_sponge_bob.outputs.name}}`,
		wantErr: errors.New(`Cannot assign input "name" due to error: steps.greet_sponge_bob.outputs.name: the "greet_sponge_bob" was not found`),
	}, {
		name: "complex steps",
		yaml: `
spec: {}
---
steps:
  - name: greet_steppy
    step: ./test_steps/greeting
    inputs:
      name: steppy
      hungry: true
      favorites:
        foods: [hamburger]
  - name: greet_the_crew
    step: ./test_steps/crew
    inputs: {}
  - name: greet_joe
    step: ./test_steps/greeting
    inputs:
      name: joe
      age: 42
      favorites:
        characters: 
          - ${{steps.greet_the_crew.outputs.crew_name_1}}
          - ${{steps.greet_the_crew.outputs.crew_name_2}}
`,
		wantLog: `meet steppy who is 1 likes {"foods":["hamburger"]} and is hungry true
meet sponge bob who is 5 likes {"pants":"square"} and is hungry false
meet patrick star who is 7 likes {"color":"red"} and is hungry true
meet joe who is 42 likes {"characters":["sponge bob","patrick star"]} and is hungry false
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 3)
		},
	}, {
		name: "global environment can be referenced",
		yaml: `
spec: {}
---
steps:
  - name: greet_steppy
    step: ./test_steps/greeting
    inputs:
      name: ${{ env.NAME }}
`,
		globalEnv: map[string]string{
			"NAME": "from-global",
		},
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "from-global", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "steps environment can be referenced",
		yaml: `
spec: {}
---
env:
  NAME: from-steps
steps:
  - name: greet_steppy
    step: ./test_steps/greeting
    inputs:
      name: ${{ env.NAME }}
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "from-steps", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "individual step invocation environment cannot be referenced during invokation",
		yaml: `
spec: {}
---
steps:
  - name: greet_steppy
    step: ./test_steps/greeting
    env:
      NAME: from-step-invocation
    inputs:
      name: ${{ env.NAME }}
`,
		wantErr: errors.New("Cannot assign input \"name\" due to error: env.NAME: the \"NAME\" was not found"),
	}, {
		name: "individual step invocation environment can be referenced by step",
		yaml: `
spec: {}
---
steps:
  - name: greet_steppy
    step: ./test_steps/greeting_name_from_env
    env:
      NAME: from-step-invocation
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "from-step-invocation", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "steps environment takes precedence over global environment",
		yaml: `
spec: {}
---
env:
  NAME: from-steps
steps:
  - name: greet_steppy
    step: ./test_steps/greeting
    inputs:
      name: ${{ env.NAME }}
`,
		globalEnv: map[string]string{
			"NAME": "from-global",
		},
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "from-steps", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "individual step invocation environment takes precedence over global environment",
		yaml: `
spec: {}
---
steps:
  - name: greet_steppy
    step: ./test_steps/greeting_name_from_env
    env:
      NAME: from-step-invocation
`,
		globalEnv: map[string]string{
			"NAME": "from-global",
		},
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "from-step-invocation", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "individual step invocation environment takes precedence over steps environment",
		yaml: `
spec: {}
---
env:
  NAME: from-steps
steps:
  - name: greet_steppy
    step: ./test_steps/greeting_name_from_env
    env:
      NAME: from-step-invocation
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "from-step-invocation", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "steps environment variables are expanded",
		yaml: `
spec: {}
---
env:
  NAME: from-${{ env.WHERE_EXACTLY }}
steps:
  - name: greet_steppy
    step: ./test_steps/greeting_name_from_env
`,
		globalEnv: map[string]string{
			"WHERE_EXACTLY": "global",
		},
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "from-global", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "step invocation environment variables are expanded",
		yaml: `
spec: {}
---
env:
steps:
  - name: greet_steppy
    step: ./test_steps/greeting_name_from_env
    env:
      NAME: from-${{ env.WHERE_EXACTLY }}
`,
		globalEnv: map[string]string{
			"WHERE_EXACTLY": "global",
		},
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "from-global", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "steps environment variables are expanded before invocation",
		yaml: `
spec: {}
---
env:
  WHERE_EXACTLY: ${{ env.WHERE_EXACTLY }}-then-steps
steps:
  - name: greet_steppy
    step: ./test_steps/greeting_name_from_env
    env:
      NAME: from-${{ env.WHERE_EXACTLY }}-then-invocation
`,
		globalEnv: map[string]string{
			"WHERE_EXACTLY": "global",
		},
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "from-global-then-steps-then-invocation", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "steps and parameters are recorded both expanded and not expanded",
		globalEnv: map[string]string{
			"REPLACE_ME": "replaced",
		},
		yaml: `
spec: {}
---
env:
  PLEASE: ${{ env.REPLACE_ME }}
  NAME: subby
steps:
  - name: greet_steppy
    step: ./test_steps/greeting
    inputs:
      name: ${{ env.NAME }}
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {

			// Top level step definition should be recorded but not expanded.
			require.Equal(t, "${{ env.REPLACE_ME }}", result.SpecDefinition.Definition.Env["PLEASE"])
			requireStringEqualValue(t, "${{ env.NAME }}", result.SpecDefinition.Definition.Steps[0].Inputs["name"])

			// Sub-step invokation should be expanded and recorded.
			requireStringEqualValue(t, "subby", result.SubStepResults[0].Step.Inputs["name"])

			// Exec definition should be recorded but not expanded.
			require.Equal(t, "${{ work_dir }}", result.SubStepResults[0].SpecDefinition.Definition.Env["HOME"])

			// Exec environment should be expanded and recorded.
			require.NotContains(t, "work_dir", result.SubStepResults[0].Env["HOME"])

			// Exec results should be recorded and expanded.
			require.NotContains(t, "work_dir", result.SubStepResults[0].ExecResult.WorkDir)
			require.Equal(t, "--name=subby", result.SubStepResults[0].ExecResult.Command[3])

			// Sub-steps environment should be expanded and recorded.
			require.Equal(t, "replaced", result.Env["PLEASE"])
		},
	}, {
		name: "delegate to exec step",
		yaml: `
spec:
  outputs: delegate
---
steps:
  - name: exec_step
    step: ./test_steps/greeting
    inputs:
      name: steppy loves delegation
delegate: exec_step
`,
		wantResults: func(t *testing.T, results *proto.StepResult) {
			requireStringEqualValue(t, "steppy loves delegation", results.Outputs["name"])
		},
	}, {
		name: "delegate to composite step",
		yaml: `
spec:
  outputs: delegate
---
steps:
  - name: composite_step
    step: ./test_steps/greeting_delegate
    inputs:
      name: steppy loves delegation
delegate: composite_step
`,
		wantResults: func(t *testing.T, results *proto.StepResult) {
			requireStringEqualValue(t, "steppy loves delegation", results.Outputs["name"])
		},
	}, {
		name: "return results even with an error",
		yaml: `
spec: {}
---
steps:
  - name: bang
    script: exit 1
`,
		wantErr: fmt.Errorf(`failed step "bang": exec: exit status 1, `),
		wantResults: func(t *testing.T, results *proto.StepResult) {
			require.NotNil(t, results)
			require.Equal(t, proto.StepResult_failure, results.Status)
			require.Len(t, results.SubStepResults, 1)
			require.Equal(t, proto.StepResult_failure, results.SubStepResults[0].Status)
			require.Equal(t, int32(1), results.SubStepResults[0].ExecResult.ExitCode)
		},
	}, {
		name: "non-sensitive input can derives value from sensitive output",
		yaml: `
spec:
---
steps:
  - name: secret_factory
    step: ./test_steps/secret_factory
  - name: greeting
    step: ./test_steps/greeting
    inputs:
      name: look, a secret! ${{ steps.secret_factory.outputs.secret }}
`,
		wantResults: func(t *testing.T, results *proto.StepResult) {
			require.Equal(t, proto.StepResult_success, results.Status)
		},
	}}

	for _, c := range cases {
		t.Run(c.name, runTest(c))
	}
}
