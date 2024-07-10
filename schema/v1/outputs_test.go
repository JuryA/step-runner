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
		name:     "empty outputs",
		json:     `{"spec":{}}`,
		yaml:     `spec: {}`,
		wantSpec: Spec{},
	}, {
		name: "outputs map",
		json: `{"spec":{"outputs":{"name":{}}}}`,
		yaml: `
spec:
  outputs:
    name: {}
`,
		wantSpec: Spec{
			Spec: Signature{
				Outputs: Outputs{
					Outputs: map[string]Output{
						"name": {},
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
			Spec: Signature{
				Outputs: Outputs{
					Delegate: true,
				},
			},
		},
	}, {
		name: "output with all fields",
		json: `{"spec":{"outputs":{"name":{"type":"string","default":"foobar","sensitive":true}}}}`,
		yaml: `
spec:
  outputs:
    name:
      type: string
      default: foobar
      sensitive: true
`,
		wantSpec: Spec{
			Spec: Signature{
				Outputs: Outputs{
					Outputs: map[string]Output{
						"name": {
							Type:      "string",
							Default:   "foobar",
							Sensitive: true,
						},
					},
				},
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
			check(t, json.Marshal, json.Unmarshal, []byte(tc.json), tc.wantSpec, specSchema, tc.wantSpec)
			check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), tc.wantSpec, specSchema, tc.wantSpec)
		})
	}
}
