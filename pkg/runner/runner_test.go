package runner_test

import (
	"bytes"
	ctx "context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

func TestRun(t *testing.T) {
	cases := []runnerTest{{
		name: "greeting with defaults",
		yaml: `
spec: {}
---
run:
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
run:
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
run:
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
run:
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
run:
  - name: greet_the_crew
    step: ./test_steps/crew
    inputs: {}
  - name: greet_previous
    step: ./test_steps/greeting
    inputs:
      name: ${{steps.greet_sponge_bob.outputs.name}}`,
		wantErr: errors.New(`failed to run step "greet_previous": failed to load: expand input "name": steps.greet_sponge_bob.outputs.name: the "greet_sponge_bob" was not found`),
	}, {
		name: "complex steps",
		yaml: `
spec: {}
---
run:
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
run:
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
		name: "run environment can be referenced",
		yaml: `
spec: {}
---
env:
  NAME: from-run
run:
  - name: greet_steppy
    step: ./test_steps/greeting
    inputs:
      name: ${{ env.NAME }}
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "from-run", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "individual step invocation environment cannot be referenced during invokation",
		yaml: `
spec: {}
---
run:
  - name: greet_steppy
    step: ./test_steps/greeting
    env:
      NAME: from-step-invocation
    inputs:
      name: ${{ env.NAME }}
`,
		wantErr: errors.New(`failed to run step "greet_steppy": failed to load: expand input "name": env.NAME: the "NAME" was not found`),
	}, {
		name: "individual step invocation environment can be referenced by step",
		yaml: `
spec: {}
---
run:
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
  NAME: from-run
run:
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
			requireStringEqualValue(t, "from-run", result.SubStepResults[0].Outputs["name"])
		},
	}, {
		name: "individual step invocation environment takes precedence over global environment",
		yaml: `
spec: {}
---
run:
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
  NAME: from-run
run:
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
run:
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
run:
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
  WHERE_EXACTLY: ${{ env.WHERE_EXACTLY }}-then-run
run:
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
			requireStringEqualValue(t, "from-global-then-run-then-invocation", result.SubStepResults[0].Outputs["name"])
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
run:
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
run:
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
run:
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
run:
  - name: bang
    script: exit 1
`,
		wantErr: fmt.Errorf(`failed to run sequence of steps: failed to run step "bang": exec: exit status 1`),
		wantResults: func(t *testing.T, results *proto.StepResult) {
			require.NotNil(t, results)
			require.Equal(t, proto.StepResult_failure, results.Status)
			require.Len(t, results.SubStepResults, 1)
			require.Equal(t, proto.StepResult_failure, results.SubStepResults[0].Status)
			require.Equal(t, int32(1), results.SubStepResults[0].ExecResult.ExitCode)
		},
	}, {
		name: "exported env can be used in subsequent step",
		yaml: `
spec:
---
run:
  - name: set_export_var
    step: ./test_steps/export_env
    inputs:
      name: FOO
      value: BAR
  - name: verify_foo_can_be_used
    step: ./test_steps/echo
    inputs:
      echo: "FOO is ${{env.FOO}}"
`,
		wantResults: func(t *testing.T, results *proto.StepResult) {
			require.NotNil(t, results)
		},
	}}

	for _, c := range cases {
		t.Run(c.name, runTest(c))
	}
}

type runnerTest struct {
	name        string
	yaml        string
	globalEnv   map[string]string
	wantLog     string
	wantResults func(*testing.T, *proto.StepResult)
	wantErr     error
}

func requireStringEqualValue(t *testing.T, str string, got *structpb.Value) {
	want := structpb.NewStringValue(str)
	require.True(t, protobuf.Equal(want, got), "want %+v. got %+v", want, got)
}

func runTest(testCase runnerTest) func(*testing.T) {
	return func(t *testing.T) {
		schemaSpec, schemaStep, err := schema.ReadSteps(testCase.yaml)
		require.NoError(t, err)
		protoSpec, err := schemaSpec.Compile()
		require.NoError(t, err)
		protoDef, err := schemaStep.Compile()
		require.NoError(t, err)
		protoStepDef := &proto.SpecDefinition{
			Spec:       protoSpec,
			Definition: protoDef,
		}
		require.NoError(t, err)
		protoStepDef.Dir, _ = os.Getwd()

		defs, err := cache.New()
		require.NoError(t, err)

		var log bytes.Buffer

		osEnv, err := runner.NewEnvironmentFromOS()
		require.NoError(t, err)

		globalCtx, err := runner.NewGlobalContext(osEnv)
		require.NoError(t, err)
		defer globalCtx.Cleanup()
		globalCtx.Env = runner.NewEnvironment(testCase.globalEnv)
		globalCtx.Stdout = &log
		globalCtx.Stderr = &log
		globalCtx.WorkDir, _ = os.UserHomeDir()

		params := &runner.Params{}

		step, err := runner.NewParser(globalCtx, defs).Parse(protoStepDef, params, runner.StepDefinedInGitLabJob)
		require.NoError(t, err)

		inputs := params.NewInputsWithDefault(protoStepDef.Spec.Spec.Inputs)
		stepsCtx := runner.NewStepsContext(globalCtx, protoStepDef.Dir, inputs, globalCtx.Env)
		result, err := step.Run(ctx.Background(), stepsCtx)

		if testCase.wantErr != nil {
			require.Error(t, err)
			require.Equal(t, testCase.wantErr.Error(), err.Error())
			if testCase.wantResults != nil {
				testCase.wantResults(t, result)
			}
		} else {
			require.NoError(t, err)
			if testCase.wantLog != "" {
				require.Equal(t, testCase.wantLog, log.String())
			}
			testCase.wantResults(t, result)
		}
	}
}
