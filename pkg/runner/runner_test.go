package runner

import (
	"bytes"
	ctx "context"
	"errors"
	"os"
	"os/exec"
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
		wantErr     error
	}{{
		name: "greeting with defaults",
		yaml: `
spec: {}
---
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
spec: {}
---
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
spec: {}
---
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
spec: {}
---
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
spec: {}
---
type: steps
steps:
  - name: greet-the-crew
    step: ./test_steps/crew
    inputs: {}
  - name: greet-previous
    step: ./test_steps/greeting
    inputs:
      name: ${{steps.greet-sponge-bob.outputs.name}}`,
		wantErr: errors.New(`Cannot assign input "name" due to error: steps.greet-sponge-bob.outputs.name: the "greet-sponge-bob" was not found`),
	}, {
		name: "complex steps",
		yaml: `
spec: {}
---
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
			stepDef, err := step.Deserialize(c.yaml, "")
			require.NoError(t, err)

			defs, err := cache.New()
			require.NoError(t, err)
			runner, err := New(defs)
			require.NoError(t, err)

			var log bytes.Buffer

			globalCtx := context.NewGlobal()
			globalCtx.Env["HOME"] = os.Getenv("HOME") // for `go run` steps
			globalCtx.Stdout = &log
			globalCtx.Stderr = &log

			params := &Params{}

			result, err := runner.Run(ctx.Background(), stepDef, params, globalCtx)
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

func TestReplay(t *testing.T) {
	steps := `
- name: replay_exec
  step: "./test_steps/rand"
- name: replay_steps
  step: "./test_steps/multiple_rand"
`

	// Run steps
	cmd := exec.Command("go", "run", "../..", "ci")
	cmd.Env = append(os.Environ(), "STEPS="+steps)
	out, err := cmd.CombinedOutput()
	require.Equal(t, 0, cmd.ProcessState.ExitCode(), string(out))
	require.NoError(t, err, string(out))

	// Replay steps
	cmd = exec.Command("go", "run", "../..", "replay", "step-results.json")
	out, err = cmd.CombinedOutput()
	require.Equal(t, 0, cmd.ProcessState.ExitCode(), string(out))
	require.NoError(t, err, string(out))
}
