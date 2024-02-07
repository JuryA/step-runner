package runner

import (
	"bytes"
	ctx "context"
	"errors"
	"maps"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/step"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestRun(t *testing.T) {
	cases := []struct {
		name        string
		yaml        string
		globalEnv   map[string]string
		wantLog     string
		wantResults func(*testing.T, []*proto.StepResult)
		wantErr     error
	}{{
		name: "greeting with defaults",
		yaml: `
spec: {}
---
type: steps
steps:
  - name: greet_steppy
    step: ./test_steps/greeting
    inputs: {}
`,
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 1)
			require.Equal(t, "steppy", results[0].Outputs["name"])
			require.Equal(t, "steppy", results[0].Exports["NAME"])
		},
	}, {
		name: "greeting outputs and exports name parameter",
		yaml: `
spec: {}
---
type: steps
steps:
  - name: greet_foo
    step: ./test_steps/greeting
    inputs:
      name: foo
`,
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 1)
			require.Equal(t, "foo", results[0].Outputs["name"])
			require.Equal(t, "foo", results[0].Exports["NAME"])
		},
	}, {
		name: "can access outputs of a previous step",
		yaml: `
spec: {}
---
type: steps
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
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 2)
			require.Equal(t, "foo", results[0].Outputs["name"])
			require.Equal(t, "foo", results[1].Outputs["name"])
		},
	}, {
		name: "can access outputs of a composite step",
		yaml: `
spec: {}
---
type: steps
steps:
  - name: greet_the_crew
    step: ./test_steps/crew
    inputs: {}
  - name: greet_previous
    step: ./test_steps/greeting
    inputs:
      name: ${{steps.greet_the_crew.outputs.crew_name_1}}
`,
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 2)
			require.Equal(t, "sponge bob", results[0].Outputs["crew_name_1"])
			require.Equal(t, "sponge bob", results[1].Outputs["name"])
		},
	}, {
		name: "cannot access outputs of composite children",
		yaml: `
spec: {}
---
type: steps
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
type: steps
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
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 3)
		},
	}, {
		name: "retain global environment",
		yaml: `
spec: {}
---
type: steps
steps:
  - name: greet_steppy
    step: ./test_steps/greeting
    inputs:
      name: ${{ env.name }}
`,
		globalEnv: map[string]string{
			"name": "global",
		},
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 1)
			require.Equal(t, "global", results[0].Outputs["name"])
			require.Equal(t, "global", results[0].Exports["NAME"])
		},
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stepDef, err := step.ReadSteps(c.yaml, "")
			require.NoError(t, err)
			protoStepDef, err := step.CompileSteps(stepDef)
			require.NoError(t, err)

			defs, err := cache.New()
			require.NoError(t, err)
			runner, err := New(defs)
			require.NoError(t, err)

			var log bytes.Buffer

			globalCtx := context.NewGlobal()
			globalCtx.Env["HOME"] = os.Getenv("HOME") // for `go run` steps
			maps.Copy(globalCtx.Env, c.globalEnv)
			globalCtx.Stdout = &log
			globalCtx.Stderr = &log

			params := &Params{}

			result, err := runner.Run(ctx.Background(), protoStepDef, params, globalCtx)
			if c.wantErr != nil {
				require.Equal(t, c.wantErr, err)
			} else {
				require.NoError(t, err)
				if c.wantLog != "" {
					require.Equal(t, c.wantLog, log.String())
				}
				c.wantResults(t, result.ChildrenStepResults)
			}
		})
	}
}
