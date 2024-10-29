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
		wantErr string
	}{{
		name:    "empty file",
		wantErr: "expected object, but got null",
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
		wantErr: "additionalProperties 'additional' not allowed",
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
		wantErr: `value must be one of "string", "number", "boolean", "struct", "array"`,
		spec: `
spec:
  inputs:
    foo:
      type: invalid
`,
	}, {
		name:    "spec with missing input type",
		wantErr: "expected object, but got null",
		spec: `
spec:
  inputs:
    foo:
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
		wantErr: "expected string, but got object",
		spec: `
spec:
  outputs:
    foo:
      type: invalid
`,
	}, {
		name:    "spec with missing output type",
		wantErr: "expected string, but got object",
		spec: `
spec:
  outputs:
    foo:
`,
	}, {
		name: "spec with delegated output",
		spec: `
spec:
  outputs: delegate
`,
	}, {
		name:    "spec with invalid outputs to delegate",
		wantErr: `value must be "delegate"`,
		spec: `
spec:
  outputs: invalid
`,
	}, {
		name: "input names must be alphanumeric",
		spec: `
spec:
  inputs:
    invalid name: {}
`,
		wantErr: "additionalProperties 'invalid name' not allowed",
	}, {
		name: "output names must be alphanumeric",
		spec: `
spec:
  outputs:
    invalid name: {}
`,
		wantErr: "expected string, but got object",
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var untyped any
			err := yaml.Unmarshal([]byte(c.spec), &untyped)
			require.NoError(t, err)

			err = specSchema.Validate(untyped)
			if c.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
