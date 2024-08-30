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
	type Default Signature
	d := (*Default)(s)
	err := value.Decode(d)
	if err != nil {
		return err
	}
	return s.unmarshalOutputs()
}

func (s *Signature) UnmarshalJSON(data []byte) error {
	type Default Signature
	d := (*Default)(s)
	err := json.Unmarshal(data, d)
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
	case string:
		if v != "delegate" {
			return fmt.Errorf("unsupported value: %v", v)
		}
		return nil
	case map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("reifying outputs: %w", err)
		}
		outputs := Outputs{}
		err = json.Unmarshal(data, outputs)
		if err != nil {
			return fmt.Errorf("reifying outputs: %w", err)
		}
		s.Outputs = outputs
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}
