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
    step: gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step@master
    inputs:
      echo: hello
`,
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 1)
			requireStringEqualValue(t, "hello", results[0].Outputs["echo"])
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
        rev: master
    inputs:
      echo: hello
`,
		wantResults: func(t *testing.T, results []*proto.StepResult) {
			require.Len(t, results, 1)
			requireStringEqualValue(t, "olleh", results[0].Outputs["echo"])
		},
	}}
	testCases(t, cases)
}
