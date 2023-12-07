package gitlab_step

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/step"
)

func TestSerialize(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		wantYaml string
		wantErr  string
	}{{
		name: "simple case",
		yaml: `
spec:
    inputs:
        name: {}
---
exec:
    command:
        - echo
        - ${{inputs.name}}
type: exec
`,
		wantYaml: `
spec:
    inputs:
        name: {}
---
exec:
    command:
        - echo
        - ${{inputs.name}}
type: exec
`}, {
		name: "convert syntactic sugar",
		yaml: `
spec: {}
---
steps:
    - script: bundle install
type: steps
`,
		wantYaml: `
spec: {}
---
steps:
    - inputs:
        script: bundle install
      step: gitlab.com/components/script@v1.0
type: steps
`}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stepDef, err := Parse(c.yaml, "")
			if c.wantErr != "" {
				require.EqualError(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
				yaml, err := step.Serialize(stepDef)
				require.NoError(t, err)
				require.Equal(t, strings.TrimSpace(c.wantYaml), strings.TrimSpace(yaml))
			}
		})
	}
}
