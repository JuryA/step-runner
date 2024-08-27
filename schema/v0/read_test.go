package schema

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestRead(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr bool
	}{{
		name: "documents out of order",
		yaml: `
type: exec
exec:
  command: [echo, "${{inputs.name}}"]
---
spec:
  inputs:
    name:
`,
		wantErr: true,
	}, {
		name: "missing spec",
		yaml: `
type: exec
exec:
  command: [echo, "${{inputs.name}}"]
`,
		wantErr: true,
	}, {
		name: "missing definition",
		yaml: `
spec:
  inputs:
    name:
`,
		wantErr: true,
	}, {
		name: "minimal step",
		yaml: `
{}
---
steps:
    - name: ""
      script: echo hello world
`,
	}, {
		name: "everything step",
		yaml: `
spec:
    inputs:
        age:
            type: number
            default: 12
        favorites:
            type: struct
            default:
                food: apple
        name:
            type: string
            default: foo
---
exec:
    command:
        - echo
        - hello world
`,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stepDef, err := ReadSteps(c.yaml, "")
			if c.wantErr {
				require.Error(t, err)
				require.Nil(t, stepDef)
			} else {
				require.NoError(t, err)
				// Assert that the whole step is preserved round-trip
				got, err := WriteSteps(stepDef)
				require.NoError(t, err)
				want := strings.TrimSpace(c.yaml)
				got = strings.TrimSpace(got)
				require.Equal(t, want, got)
			}
		})
	}
}

func readProto(content, dir string) (*proto.SpecDefinition, error) {
	var (
		spec       proto.Spec
		definition proto.Definition
	)

	if err := unmarshalProto(content, &spec, &definition); err != nil {
		return nil, fmt.Errorf("unmarshaling proto: %w", err)
	}
	stepDef := &proto.SpecDefinition{
		Spec:       &spec,
		Definition: &definition,
		Dir:        dir,
	}
	if err := validateStepDefinition(stepDef); err != nil {
		return nil, err
	}
	return stepDef, nil
}

func unmarshalProto(input string, subjects ...protoreflect.ProtoMessage) error {
	d := yaml.NewDecoder(strings.NewReader(input))
	d.KnownFields(true)

	for _, subject := range subjects {
		var decoded any
		err := d.Decode(&decoded)
		if err != nil {
			return fmt.Errorf("decoding: %w", err)
		}

		// convert to json
		encoded, err := json.Marshal(decoded)
		if err != nil {
			return fmt.Errorf("converting to json: %w", err)
		}

		// convert to proto
		if err := protojson.Unmarshal(encoded, subject); err != nil {
			return fmt.Errorf("converting to proto: %w", err)
		}
	}

	return nil

}
