package schema

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

var (
	_ yaml.Unmarshaler = &Signature{}
	_ json.Unmarshaler = &Signature{}
)

func (s *Signature) UnmarshalYAML(value *yaml.Node) error {
	err := value.Decode(s)
	if err != nil {
		return err
	}
	return s.unmarshalOutputs()
}

func (s *Signature) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, s)
	if err != nil {
		return err
	}
	return s.unmarshalOutputs()
}

func (s *Signature) unmarshalOutputs() error {
	if s.Outputs == nil {
		return nil
	}
	switch v := s.Outputs.(type) {
	case *string:
		if *v != "delegate" {
			return fmt.Errorf("unsupported value: %v", *v)
		}
		return nil
	case map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("reifying outputs: %w", err)
		}
		err = json.Unmarshal(data, &s.Outputs)
		return err
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}
