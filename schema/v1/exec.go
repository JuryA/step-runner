package schema

import (
	"encoding/json"
	"fmt"
)

type Exec struct {
	// Command are the parameters to the system exec API. It does not invoke a shell.
	Command []string `json:"command" yaml:"command" mapstructure:"command"`

	// WorkDir is the working directly in which `command` will be exec'ed.
	WorkDir *string `json:"work_dir,omitempty" yaml:"work_dir,omitempty" mapstructure:"work_dir,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Exec) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["command"]; raw != nil && !ok {
		return fmt.Errorf("field command in Exec: required")
	}
	type Plain Exec
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.Command != nil && len(plain.Command) < 1 {
		return fmt.Errorf("field %s length: must be >= %d", "command", 1)
	}
	*j = Exec(plain)
	return nil
}
