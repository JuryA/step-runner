package schema

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"gopkg.in/yaml.v3"
)

// Outputs is a structure defining how this step should provide
// outputs to the execution context. Outputs can be constructed with
// expressions or they can be taken directly from a delegate step.
type Outputs struct {
	// Outputs is a map of output name to type and optionally a
	// default value.
	Outputs map[string]Output `json:"-" yaml:"-"`
	// Delegate is a boolean value indicating outputs should be
	// taken directly from step outputs returned as a step results
	// structure..
	Delegate bool `json:"-" yaml:"-"`
}

var (
	_ yaml.Unmarshaler = &Outputs{}
	_ yaml.Marshaler   = &Outputs{}
	_ json.Unmarshaler = &Outputs{}
	_ json.Marshaler   = &Outputs{}
)

func (o *Outputs) UnmarshalYAML(value *yaml.Node) error {
	switch value.Tag {
	case yamlStringTag:
		var v string
		err := value.Decode(&v)
		if err != nil {
			return err
		}
		if v != delegate {
			return fmt.Errorf("invalid output method option: %q", v)
		}
		o.Delegate = true
		return nil
	case yamlMapTag:
		return value.Decode(&o.Outputs)
	default:
		return fmt.Errorf("unsupported output method type: %q", value.Tag)
	}
}

func (o Outputs) MarshalYAML() (any, error) {
	switch {
	case o.Delegate:
		return delegate, nil
	default:
		return o.Outputs, nil
	}
}

func (o *Outputs) UnmarshalJSON(data []byte) error {
	var untyped any
	err := json.Unmarshal(data, &untyped)
	if err != nil {
		return err
	}
	switch v := untyped.(type) {
	case string:
		if v != delegate {
			return fmt.Errorf("invalid output method options: %q", v)
		}
		o.Delegate = true
		return nil
	case map[string]any:
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(data, &o.Outputs)
	default:
		return fmt.Errorf("unsupported type: %T", untyped)
	}
}

func (o Outputs) MarshalJSON() ([]byte, error) {
	switch {
	case o.Delegate:
		return json.Marshal(delegate)
	case o.Outputs == nil:
		return json.Marshal(map[string]Output{})
	default:
		return json.Marshal(o.Outputs)
	}
}

func (o Outputs) JSONSchema() *jsonschema.Schema {
	jsr := jsonschema.Reflector{
		DoNotReference: true,
	}
	if err := jsr.AddGoComments("gitlab.com/gitlab-org/step-runner", "./"); err != nil {
		panic(err)
	}
	outputs := jsr.Reflect(map[string]Output{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			outputs,
		},
	}
}
