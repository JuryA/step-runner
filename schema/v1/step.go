package schema

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Step is a unit of execution.
type Step struct {
	// Action is a GitHub action to run.
	Action *string `json:"action,omitempty" yaml:"action,omitempty" mapstructure:"action,omitempty"`

	// Delegate selects a step by name which will produce the outputs a run.
	Delegate *string `json:"delegate,omitempty" yaml:"delegate,omitempty" mapstructure:"delegate,omitempty"`

	// Env is a map of environment variable names to string values.
	Env StepEnv `json:"env,omitempty" yaml:"env,omitempty" mapstructure:"env,omitempty"`

	// Exec is a command to run.
	Exec *Exec `json:"exec,omitempty" yaml:"exec,omitempty" mapstructure:"exec,omitempty"`

	// Inputs is a map of step input names to structured values.
	Inputs StepInputs `json:"inputs,omitempty" yaml:"inputs,omitempty" mapstructure:"inputs,omitempty"`

	// Name is a unique identifier for this step.
	Name *string `json:"name,omitempty" yaml:"name,omitempty" mapstructure:"name,omitempty"`

	// Outputs are the output values for a sequence. They can reference the outputs of
	// sub-steps.
	Outputs StepOutputs `json:"outputs,omitempty" yaml:"outputs,omitempty" mapstructure:"outputs,omitempty"`

	// Script is a shell script to evaluate.
	Script *string `json:"script,omitempty" yaml:"script,omitempty" mapstructure:"script,omitempty"`

	// Step is a reference to another step to invoke.
	Step interface{} `json:"step,omitempty" yaml:"step,omitempty" mapstructure:"step,omitempty"`

	// Run is a list of sub-steps to run.
	Run []Step `json:"run,omitempty" yaml:"run,omitempty" mapstructure:"run,omitempty"`
}

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
		ref := &Reference{}
		err = json.Unmarshal(data, ref)
		if err != nil {
			return fmt.Errorf("reifying step: %w", err)
		}
		s.Step = ref
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// Env is a map of environment variable names to string values.
type StepEnv map[string]string

// Inputs is a map of step input names to structured values.
type StepInputs map[string]interface{}

// Outputs are the output values for a sequence. They can reference the outputs of
// sub-steps.
type StepOutputs map[string]interface{}
