package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSpecSchemaValidate(t *testing.T) {
	cases := []struct {
		name    string
		spec    string
		wantErr bool
	}{{
		name:    "empty file",
		wantErr: true,
		spec:    "",
	}, {
		name: "empty document",
		spec: `
{}
`,
	}, {
		name: "noop spec",
		spec: `
spec: {}
`,
	}, {
		name:    "spec with additional properties",
		wantErr: true,
		spec: `
spec:
  additional: property
`,
	}, {
		name: "spec with different input types",
		spec: `
spec:
  inputs:
    foo:
      type: string
    bar:
      type: number
    baz:
      type: array
    bam:
      type: struct
    bow:
      type: boolean
`,
	}, {
		name:    "spec with invalid input type",
		wantErr: true,
		spec: `
spec:
  inputs:
    foo:
      type: invalid
`,
	}, {
		name: "spec with different output types",
		spec: `
spec:
  outputs:
    foo:
      type: string
    bar:
      type: number
    baz:
      type: array
    bam:
      type: struct
    bow:
      type: boolean
`,
	}, {
		name:    "spec with invalid output type",
		wantErr: true,
		spec: `
spec:
  outputs:
    foo:
      type: invalid
`,
	}, {
		name: "spec with delegated output",
		spec: `
spec:
  outputs: delegate
`,
	}, {
		name:    "spec with invalid string",
		wantErr: true,
		spec: `
spec:
  outputs: invalid
`,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var untyped any
			err := yaml.Unmarshal([]byte(c.spec), &untyped)
			require.NoError(t, err)
			if c.wantErr {
				require.Error(t, specSchema.Validate(untyped))
			} else {
				require.NoError(t, specSchema.Validate(untyped))
			}
		})
	}
}
