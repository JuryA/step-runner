package schema

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// var (
// 	_ yaml.Unmarshaler = SignatureOutputs(nil)
// 	_ json.Unmarshaler = SignatureOutputs(nil)
// )

func (so SignatureOutputs) UnmarshalYAML(value *yaml.Node) error {
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
		so = "delegate"
		return nil
	case yamlMapTag:
		o := &Outputs{}
		return value.Decode(o)
	default:
		return fmt.Errorf("unsupported output method type: %q", value.Tag)
	}
}

func (so SignatureOutputs) UnmarshalJSON(data []byte) error {
	var untyped any
	err := json.Unmarshal(data, &untyped)
	if err != nil {
		return err
	}
	switch v := untyped.(type) {
	case string:
		so = v
		return nil
	case map[string]any:
		so = &Outputs{}
		return json.Unmarshal(data, so)
	default:
		return fmt.Errorf("unsupported type: %T", untyped)
	}
}
