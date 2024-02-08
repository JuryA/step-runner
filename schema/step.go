package schema

type StepDefinition struct {
	Spec       *Spec
	Definition *Definition
	Dir        string
}

// Definition is the implementation of a step.
type Definition struct {
	Type    DefinitionType    `json:"type" yaml:"type" jsonschema:"enum=exec,enum=steps"`
	Steps   []*Step           `json:"steps" yaml:"steps"`
	Exec    Exec              `json:"exec" yaml:"exec"`
	Outputs map[string]string `json:"outputs" yaml:"outputs"`

	// Script is a shell script to evaluate.
	Script string `json:"script" yaml:"script"`
}

type DefinitionType string

const (
	DefinitionTypeExec  DefinitionType = "exec"
	DefinitionTypeSteps DefinitionType = "steps"
)

type Exec struct {
	Command []string `json:"command" yaml:"command"`
	WorkDir string   `json:"work_dir" yaml:"work_dir"`
}

// Step is a single step invocation.
type Step struct {
	// Name is a unique identifier for this step.
	Name string `json:"name" yaml:"name"`
	// Step is a reference to the step to invoke.
	Step string `json:"step" yaml:"step"`
	// Env is a map of environment variable names to values.
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	// Inputs is a map of step input names to structured values.
	Inputs map[string]any `json:"inputs,omitempty" yaml:"inputs,omitempty"`

	// Script is a shell script to evaluate.
	Script string `json:"script" yaml:"script"`
}

type Spec struct {
	Spec Content `json:"spec" yaml:"spec"`
}

type Content struct {
	Inputs  map[string]Input  `json:"inputs" yaml:"inputs"`
	Outputs map[string]Output `json:"outputs" yaml:"outputs"`
}

type Input struct {
	Type    ValueType `json:"type" yaml:"type"`
	Default any       `json:"default" yaml:"default"`
}

type Output struct {
	Default string `json:"default" yaml:"default"`
}

type ValueType string

const (
	ValueTypeNull   ValueType = "null"
	ValueTypeString ValueType = "string"
	ValueTypeNumber ValueType = "number"
	ValueTypeBool   ValueType = "bool"
	ValueTypeStruct ValueType = "struct"
	ValueTypeList   ValueType = "list"
)
