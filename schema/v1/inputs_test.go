package schema

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

func TestInputsCustomMethods(t *testing.T) {
	cases := []struct {
		name          string
		json          string
		yaml          string
		wantInput     Input
		wantSchemaErr bool
	}{{
		name: "empty input",
		json: `{}`,
		yaml: `{}`,
		wantInput: Input{},
	}, {
		name: "input with type",
		json: `{"type":"string"}`,
		yaml: `type: string`,
		wantInput: Input{
			Type: "string",
		},
	}, {
		name: "input with description",
		json: `{"description":"This is a test input"}`,
		yaml: `description: This is a test input`,
		wantInput: Input{
			Description: "This is a test input",
		},
	}, {
		name: "input with type and description",
		json: `{"type":"string","description":"This is a test input"}`,
		yaml: `
type: string
description: This is a test input
`,
		wantInput: Input{
			Type:        "string",
			Description: "This is a test input",
		},
	}}

	data, err := os.ReadFile("spec.json")
	if err != nil {
		panic(err)
	}
	specSchema := jsonschema.MustCompileString("spec.json", string(data))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check(t, json.Marshal, json.Unmarshal, []byte(tc.json), tc.wantInput, specSchema)
			check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), tc.wantInput, specSchema)
		})
	}
}