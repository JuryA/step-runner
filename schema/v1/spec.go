package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Input struct {
	// AdditionalProperties corresponds to the JSON schema field
	// "additionalProperties".
	AdditionalProperties interface{} `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty" mapstructure:"additionalProperties,omitempty"`

	// Default is the default input value. Its type must match `type`.
	Default InputDefault `json:"default,omitempty" yaml:"default,omitempty" mapstructure:"default,omitempty"`

	// Sensitive implies the input is of sensitive nature and effort should be made to
	// prevent accidental disclosure.
	Sensitive *bool `json:"sensitive,omitempty" yaml:"sensitive,omitempty" mapstructure:"sensitive,omitempty"`

	// Type is the value type of the input.
	Type *InputType `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type,omitempty"`
}

// Default is the default input value. Its type must match `type`.
type InputDefault interface{}

type InputDefaultTypes interface{}

type InputType string

const InputTypeArray InputType = "array"
const InputTypeBoolean InputType = "boolean"
const InputTypeNumber InputType = "number"
const InputTypeString InputType = "string"
const InputTypeStruct InputType = "struct"
const InputTypeStep InputType = "step"

var enumValues_InputType = []interface{}{
	"string",
	"number",
	"boolean",
	"struct",
	"array",
	"step",
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *InputType) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValues_InputType {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValues_InputType, v)
	}
	*j = InputType(v)
	return nil
}

// Output describes a single step output.
type Output struct {
	// Default is the default output value.
	Default OutputDefault `json:"default,omitempty" yaml:"default,omitempty" mapstructure:"default,omitempty"`

	// Sensitive implies the output is of sensitive nature and effort should be made
	// to prevent accidental disclosure.
	Sensitive *bool `json:"sensitive,omitempty" yaml:"sensitive,omitempty" mapstructure:"sensitive,omitempty"`

	// Type is the value type of the output.
	Type *OutputType `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type,omitempty"`
}

// Default is the default output value.
type OutputDefault interface{}

type OutputDefaultTypes interface{}

type OutputType string

const OutputTypeArray OutputType = "array"
const OutputTypeBoolean OutputType = "boolean"
const OutputTypeNumber OutputType = "number"
const OutputTypeRawString OutputType = "raw_string"
const OutputTypeString OutputType = "string"
const OutputTypeStruct OutputType = "struct"

var enumValues_OutputType = []interface{}{
	"raw_string",
	"string",
	"number",
	"boolean",
	"struct",
	"array",
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OutputType) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValues_OutputType {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValues_OutputType, v)
	}
	*j = OutputType(v)
	return nil
}

type Outputs map[string]Output

// Signature contains the inputs and outputs of the step.
type Signature struct {
	// Input describes a single step input.
	Inputs SignatureInputs `json:"inputs,omitempty" yaml:"inputs,omitempty" mapstructure:"inputs,omitempty"`

	// Outputs corresponds to the JSON schema field "outputs".
	Outputs interface{} `json:"outputs,omitempty" yaml:"outputs,omitempty" mapstructure:"outputs,omitempty"`
}

// Input describes a single step input.
type SignatureInputs map[string]Input

// Spec is a document describing the interface of a step.
type Spec struct {
	// Spec corresponds to the JSON schema field "spec".
	Spec *Signature `json:"spec,omitempty" yaml:"spec,omitempty" mapstructure:"spec,omitempty"`
}
