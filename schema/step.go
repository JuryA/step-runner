package schema

// Definition is the implementation of a step.
type Definition struct {
	Type    DefinitionType    `json:"type" jsonschema:"enum=exec,enum=steps"`
	Steps   []*Step           `json:"steps"`
	Exec    Exec              `json:"exec"`
	Outputs map[string]string `json:"outputs"`

	Script    string `json:"script"`
	Container string `json:"container"`
}

type DefinitionType string

const (
	DefinitionTypeExec  DefinitionType = "exec"
	DefinitionTypeSteps DefinitionType = "steps"
)

type Exec struct {
	Command []string `json:"command"`
	WorkDir string   `json:"work_dir"`
}

// Step is a single step invocation.
type Step struct {
	// Name is a unique identifier for this step.
	Name string `json:"name"`
	// Step is a reference to the step to invoke.
	Step string `json:"step"`
	// Env is a map of environment variable names to values.
	Env map[string]string `json:"env,omitempty"`
	// Inputs is a map of step input names to structured values.
	Inputs map[string]Value `json:"inputs,omitempty"`

	// Script is a shell script to evaluate.
	Script string `json:"script"`
	// Container is a Docker container in which to run the step.
	Container string `json:"container"`
}

// Value is a valid JSON value.
type Value interface {
	isValue()
}

// NullValue is a JSON null.
type NullValue struct{}

func (v NullValue) isValue() {}

// BoolValue is a JSON bool.
type BoolValue bool

func (v BoolValue) isValue() {}

// NumberValue is a JSON number (float64).
type NumberValue float64

func (v NumberValue) isValue() {}

// StringValue is a JSON string.
type StringValue string

func (v StringValue) isValue() {}

// StructValue is a JSON struct.
type StructValue map[string]Value

func (v StructValue) isValue() {}

// ListValue is a JSON list.
type ListValue []Value

func (v ListValue) isValue() {}

type Spec struct {
	Spec Content `json:"spec"`
}

type Content struct {
	Inputs  map[string]Input  `json:"inputs"`
	Outputs map[string]Output `json:"outputs"`
}

type Input struct {
	Type    InputType `json:"type"`
	Default Value     `json:"default"`
}

type Output struct {
	Default string `json:"default"`
}

type InputType string

const (
	InputTypeString InputType = "string"
	InputTypeNumber InputType = "number"
	InputTypeBool   InputType = "bool"
	InputTypeStruct InputType = "struct"
	InputTypeList   InputType = "list"
)
