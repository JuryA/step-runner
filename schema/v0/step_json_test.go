package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValidStep(t *testing.T) {
}

func TestInvalidStep(t *testing.T) {
	cases := []struct {
		name string
		step string
	}{{
		name: "step and script mutually exclusive",
		step: `
step: ./my-step
script: my script
`,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var untyped any
			err := yaml.Unmarshal([]byte(c.step), &untyped)
			require.NoError(t, err)
			require.Error(t, stepSchema.Validate(untyped))
		})
	}

}
