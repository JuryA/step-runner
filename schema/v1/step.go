package schema

type StepDefinition struct {
	Spec       *Spec
	Definition *Definition
	Dir        string
}

type Steps []*Step

// Definition is the implementation of a step.
type Definition struct {
	// Steps is a list of sub-steps to run for the `steps` type.
	Steps Steps `json:"steps,omitempty" yaml:"steps,omitempty" jsonschema:"oneof_required=steps"`
	// Exec is a command to run for the `exec` type.
	Exec Exec `json:"exec,omitempty" yaml:"exec,omitempty" jsonschema:"oneof_required=exec"`
	// Outputs are the output values for a `steps` type. They can reference the outputs of sub-steps.
	Outputs map[string]any `json:"outputs,omitempty" yaml:"outputs,omitempty"`
	// Env is a map of environment variable names to values for all steps
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	// Delegate selects a step by name which will produce the outputs for this step.
	Delegate string `json:"delegate,omitempty" yaml:"delegate,omitempty"`
}

type Exec struct {
	// Command are the parameters to the system exec API. It does not invoke a shell.
	Command []string `json:"command" yaml:"command"`
	// WorkDir is the working directly in which `command` will be exec'ed.
	WorkDir string `json:"work_dir,omitempty" yaml:"work_dir,omitempty"`
}

// Step is a single step invocation.
type Step struct {
	// Name is a unique identifier for this step.
	Name string `json:"name" yaml:"name"`
	// Step is a reference to the step to invoke.
	Step Reference `json:"step,omitempty" yaml:"step,omitempty" jsonschema:"oneof_required=step"`
	// Env is a map of environment variable names to string values.
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	// Inputs is a map of step input names to structured values.
	Inputs map[string]any `json:"inputs,omitempty" yaml:"inputs,omitempty"`

	// Script is a shell script to evaluate.
	Script string `json:"script,omitempty" yaml:"script,omitempty" jsonschema:"oneof_required=script"`
	// Action is a GitHub action to run.
	Action string `json:"action,omitempty" yaml:"action,omitempty" jsonschema:"oneof_required=action"`
}

// Spec is a document describing the interface of the step.
type Spec struct {
	Spec Signature `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// Content contains the inputs and outputs of the step.
type Signature struct {
	// Inputs is a map of input names to their parameters.
	Inputs map[string]Input `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	// Outputs is a map of output names to their parameters.
	Outputs Outputs `json:"outputs,omitempty" yaml:"outputs,omitempty"`
}

// Input describes a single step input.
type Input struct {
	// Type is the value type of the input.
	Type ValueType `json:"type,omitempty" yaml:"type,omitempty"`
	// Default is the default input value. Its type must match `type`.
	Default any `json:"default,omitempty" yaml:"default,omitempty"`
	// Sensitive implies the input is of sensitive nature and effort should be made to prevent accidental disclosure.
	Sensitive bool `json:"sensitive,omitempty" yaml:"sensitive,omitempty"`
}

// Output describes a single step output.
type Output struct {
	// Type is the value type of the output.
	Type ValueType `json:"type,omitempty" yaml:"type,omitempty"`
	// Default is the default output value.
	Default any `json:"default,omitempty" yaml:"default,omitempty"`
	// Sensitive implies the output is of sensitive nature and effort should be made to prevent accidental disclosure.
	Sensitive bool `json:"sensitive,omitempty" yaml:"sensitive,omitempty"`
}

type ValueType string

const (
	ValueTypeRawString  ValueType = "raw_string"
	ValueTypeString     ValueType = "string"
	ValueTypeNumber     ValueType = "number"
	ValueTypeBool       ValueType = "boolean"
	ValueTypeStruct     ValueType = "struct"
	ValueTypeList       ValueType = "array"
	ValueTypeStepResult ValueType = "step_result"
)
