package schema

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func (ss StepStep) UnmarshalYAML(value *yaml.Node) error {
	switch value.Tag {
	case yamlStringTag:
		s := ""
		ss = &s
		return value.Decode(ss)
	case yamlMapTag:
		ref := &GitReference{}
		err := value.Decode(ref)
		if err != nil {
			return err
		}
		ss = ref
		return nil
	default:
		return fmt.Errorf("unsupported reference type: %q", value.Tag)
	}
}

func (ss StepStep) UnmarshalJSON(data []byte) error {
	var untyped any
	err := json.Unmarshal(data, &untyped)
	if err != nil {
		return err
	}
	switch t := untyped.(type) {
	case string:
		ss = &t
		return nil
	case map[string]any:
		ref := &Reference{}
		err := json.Unmarshal(data, ref)
		if err != nil {
			return err
		}
		ss = ref
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", untyped)
	}
}
