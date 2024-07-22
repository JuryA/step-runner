//go:build integration

package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestRunRemote(t *testing.T) {
	cases := []runnerTest{{
		name: "echo",
		yaml: `
spec: {}
---
steps:
  - name: echo_hello
    step: gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step@main
    inputs:
      echo: hello
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "hello", result.SubStepResults[0].Outputs["echo"])
		},
	}, {
		name: "echo reverse",
		yaml: `
spec: {}
---
steps:
  - name: echo_hello_reverse
    step: 
      git:
        url: gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step
        dir: reverse
        rev: main
    inputs:
      echo: hello
`,
		wantResults: func(t *testing.T, result *proto.StepResult) {
			require.Len(t, result.SubStepResults, 1)
			requireStringEqualValue(t, "olleh", result.SubStepResults[0].Outputs["echo"])
		},
	}}

	for _, c := range cases {
		t.Run(c.name, runTest(c))
	}
}
