package schema

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

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
	Inputs map[string]Value `json:"inputs,omitempty" yaml:"inputs,omitempty"`

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
	Default *Value    `json:"default" yaml:"default"`
}

type Output struct {
	Default string `json:"default" yaml:"default"`
}

type ValueType string

const (
	ValueTypeString ValueType = "string"
	ValueTypeNumber ValueType = "number"
	ValueTypeBool   ValueType = "bool"
	ValueTypeStruct ValueType = "struct"
	ValueTypeList   ValueType = "list"
)

// Value is a valid JSON value.
type Value struct {
	Type ValueType

	Bool   *bool
	Number *float64
	String *string
	Struct map[string]Value
	List   []Value
}

func BoolValue(b bool) Value {
	return Value{
		Type: ValueTypeBool,
		Bool: &b,
	}
}

func NumberValue(n float64) Value {
	return Value{
		Type:   ValueTypeNumber,
		Number: &n,
	}
}

func StringValue(s string) Value {
	return Value{
		Type:   ValueTypeString,
		String: &s,
	}
}

func StructValue(s map[string]Value) Value {
	if s == nil {
		s = map[string]Value{}
	}
	return Value{
		Type:   ValueTypeStruct,
		Struct: s,
	}
}

func ListValue(l []Value) Value {
	if l == nil {
		l = []Value{}
	}
	return Value{
		Type: ValueTypeList,
		List: l,
	}
}

const (
	boolTag  = "!!bool"
	strTag   = "!!str"
	intTag   = "!!int"
	floatTag = "!!float"
	seqTag   = "!!seq"
	mapTag   = "!!map"
)

func (v *Value) UnmarshalYAML(value *yaml.Node) error {
	var err error
	switch value.ShortTag() {
	case boolTag:
		v.Type = ValueTypeBool
		err = value.Decode(&v.Bool)
	case strTag:
		v.Type = ValueTypeString
		err = value.Decode(&v.String)
	case intTag, floatTag:
		v.Type = ValueTypeNumber
		err = value.Decode(&v.Number)
	case seqTag:
		v.Type = ValueTypeList
		err = value.Decode(&v.List)
	case mapTag:
		v.Type = ValueTypeStruct
		err = value.Decode(&v.Struct)
	default:
		return fmt.Errorf("unsupported type: %v", value.ShortTag())
	}
	return err
}
