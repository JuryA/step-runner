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
	var raw map[string]interface{}
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

// GitReference is a reference to a step in a Git repository containing the full
// set of configuration options.
type GitReference struct {
	// Dir corresponds to the JSON schema field "dir".
	Dir *string `json:"dir,omitempty" yaml:"dir,omitempty" mapstructure:"dir,omitempty"`

	// Rev corresponds to the JSON schema field "rev".
	Rev string `json:"rev" yaml:"rev" mapstructure:"rev"`

	// Url corresponds to the JSON schema field "url".
	Url string `json:"url" yaml:"url" mapstructure:"url"`

	// File corresponds to the JSON schema field "file".
	File *string `json:"file,omitempty" yaml:"file,omitempty" mapstructure:"file,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *GitReference) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["rev"]; raw != nil && !ok {
		return fmt.Errorf("field rev in git: required")
	}
	if _, ok := raw["url"]; raw != nil && !ok {
		return fmt.Errorf("field url in git: required")
	}
	type Plain GitReference
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = GitReference(plain)
	return nil
}

// Git a reference to a step in a Git repository.
type Reference struct {
	// Git corresponds to the JSON schema field "git".
	Git GitReference `json:"git" yaml:"git" mapstructure:"git"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Reference) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["git"]; raw != nil && !ok {
		return fmt.Errorf("field git: required")
	}
	type Plain Reference
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Reference(plain)
	return nil
}

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

// Env is a map of environment variable names to string values.
type StepEnv map[string]string

// Inputs is a map of step input names to structured values.
type StepInputs map[string]interface{}

// Outputs are the output values for a sequence. They can reference the outputs of
// sub-steps.
type StepOutputs map[string]interface{}
