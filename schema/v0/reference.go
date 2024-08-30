package schema

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

var (
	_ yaml.Unmarshaler = &Step{}
	_ json.Unmarshaler = &Step{}
)

func (s *Step) UnmarshalYAML(value *yaml.Node) error {
	type Default Step
	d := (*Default)(s)
	err := value.Decode(d)
	if err != nil {
		return err
	}
	return s.unmarshalStep()
}

func (s *Step) UnmarshalJSON(data []byte) error {
	type Default Step
	d := (*Default)(s)
	err := json.Unmarshal(data, d)
	if err != nil {
		return err
	}
	return s.unmarshalStep()
}

func (s *Step) unmarshalStep() error {
	if s.Step == nil {
		return nil
	}
	switch v := s.Step.(type) {
	case string:
		return nil
	case map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("reifying step: %w", err)
		}
		step := &Step{}
		err = json.Unmarshal(data, step)
		if err != nil {
			return fmt.Errorf("reifying step: %w", err)
		}
		s.Step = step
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}
