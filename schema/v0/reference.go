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
	err := value.Decode(s)
	if err != nil {
		return err
	}
	return s.unmarshalStep()
}

func (s *Step) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, s)
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
	case *string:
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
