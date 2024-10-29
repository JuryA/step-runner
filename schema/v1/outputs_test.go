package schema

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

func TestOutputsCustomMethods(t *testing.T) {
	cases := []struct {
		name          string
		json          string
		yaml          string
		wantSpec      Spec
		wantSchemaErr bool
	}{{
		name: "empty outputs",
		json: `{"spec":{}}`,
		yaml: `spec: {}`,
		wantSpec: Spec{
			Spec: &Signature{},
		},
	}, {
		name: "outputs map",
		json: `{"spec":{"outputs":{"name":{"type":"string"}}}}`,
		yaml: `
spec:
  outputs:
    name:
      type: string
`,
		wantSpec: Spec{
			Spec: &Signature{
				Outputs: &Outputs{
					"name": {
						Type: func() *OutputType { o := OutputType("string"); return &o }(),
					},
				},
			},
		},
	}, {
		name: "delegate outputs",
		json: `{"spec":{"outputs":"delegate"}}`,
		yaml: `
spec:
  outputs: delegate
`,
		wantSpec: Spec{
			Spec: &Signature{
				Outputs: "delegate",
			},
		},
	}}

	data, err := os.ReadFile("spec.json")
	if err != nil {
		panic(err)
	}
	specSchema := jsonschema.MustCompileString("spec.json", string(data))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check(t, json.Marshal, json.Unmarshal, []byte(tc.json), tc.wantSpec, specSchema)
			check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), tc.wantSpec, specSchema)
		})
	}
}
