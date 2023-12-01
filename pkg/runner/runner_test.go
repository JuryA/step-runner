package runner

import (
	ctx "context"
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
		wantLog     string
		wantResults func(*testing.T, []*proto.StepResult)
		wantErr     bool
	}{{
		name: "greeting with defaults",
		yaml: `
type: steps
steps:
  - name: greet-steppy
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
type: steps
steps:
  - name: greet-foo
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
type: steps
steps:
  - name: greet-foo
    step: ./test_steps/greeting
    inputs:
      name: foo
  - name: greet-previous
    step: ./test_steps/greeting
    inputs:
      name: ${{steps.greet-foo.outputs.name}}
`,
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 2)
			require.Equal(t, "foo", results[0].Outputs["name"])
			require.Equal(t, "foo", results[1].Outputs["name"])
		},
	}, {
		name: "can access outputs of a composite step",
		yaml: `
type: steps
steps:
  - name: greet-the-crew
    step: ./test_steps/crew
    inputs: {}
  - name: greet-previous
    step: ./test_steps/greeting
    inputs:
      name: ${{steps.greet-the-crew.outputs.crew-name-1}}
`,
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 2)
			require.Equal(t, "sponge bob", results[0].Outputs["crew-name-1"])
			require.Equal(t, "sponge bob", results[1].Outputs["name"])
		},
	}, {
		name: "cannot access outputs of composite children",
		yaml: `
type: steps
steps:
  - name: greet-the-crew
    step: ./test_steps/crew
    inputs: {}
  - name: greet-previous
    step: ./test_steps/greeting
    inputs:
      name: ${{steps.greet-sponge-bob.outputs.name}}`,
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 2)
			require.Equal(t, "sponge bob", results[0].Outputs["crew-name-1"])
			require.Equal(t, "sponge bob", results[0].ChildrenStepResults[0].Outputs["name"])
			require.Equal(t, "${{steps.greet-sponge-bob.outputs.name}}", results[1].Outputs["name"]) // not expanded
		},
	}, {
		name: "complex steps",
		yaml: `
type: steps
steps:
  - name: greet-steppy
    step: ./test_steps/greeting
    inputs:
      name: steppy
      hungry: true
      favorites:
        foods: [hamburger]
  - name: greet-the-crew
    step: ./test_steps/crew
    inputs: {}
  - name: greet-joe
    step: ./test_steps/greeting
    inputs:
      name: joe
      age: 42
      favorites:
        characters: 
          - ${{steps.greet-the-crew.outputs.crew-name-1}}
          - ${{steps.greet-the-crew.outputs.crew-name-2}}
`,
		wantLog: `meet steppy who is 1 likes {"foods":["hamburger"]} and is hungry true
meet sponge bob who is 5 likes {"pants":"square"} and is hungry false
meet patrick star who is 7 likes {"color":"red"} and is hungry true
meet joe who is 42 likes {"characters":["sponge bob","patrick star"]} and is hungry false
`,
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 3)
		},
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			def, err := step.ReadSteps(c.yaml)
			require.NoError(t, err)
			defs, err := cache.New()
			require.NoError(t, err)
			defer defs.Cleanup()
			globalCtx := context.NewGlobal()
			globalCtx.Env["HOME"] = os.Getenv("HOME") // for `go run` steps
			runner, err := New(ctx.Background(), defs, globalCtx, def.Steps)
			require.NoError(t, err)
			var (
				results []*proto.StepResult
				log     string
			)
			fn := func(r *proto.StepResult, l string) {
				results = append(results, r)
				log += l
			}
			err = runner.Run(fn)
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if c.wantLog != "" {
					require.Equal(t, c.wantLog, log)
				}
				c.wantResults(t, results)
			}
		})
	}
}
