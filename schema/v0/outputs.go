package schema

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

var (
	_ yaml.Unmarshaler = SpecJsonSpecOutputs(nil)
	_ yaml.Marshaler   = SpecJsonSpecOutputs(nil)
	_ json.Unmarshaler = SpecJsonSpecOutputs(nil)
	_ json.Marshaler   = SpecJsonSpecOutputs(nil)
)

func (o SpecJsonSpecOutputs) UnmarshalYAML(value *yaml.Node) error {
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
		o = "delegate"
		return nil
	case yamlMapTag:
		o := &Outputs{}
		return value.Decode(o)
	default:
		return fmt.Errorf("unsupported output method type: %q", value.Tag)
	}
}

func (o SpecJsonSpecOutputs) UnmarshalJSON(data []byte) error {
	var untyped any
	err := json.Unmarshal(data, &untyped)
	if err != nil {
		return err
	}
	switch v := untyped.(type) {
	case string:
		o = v
		return nil
	case map[string]any:
		o = &Outputs{}
		return json.Unmarshal(data, o)
	default:
		return fmt.Errorf("unsupported type: %T", untyped)
	}
}
