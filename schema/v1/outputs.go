package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Outputs map[string]Output

// Output describes a single step output.
type Output struct {
	// Default is the default output value.
	Default any `json:"default,omitempty" yaml:"default,omitempty" mapstructure:"default,omitempty"`

	// Sensitive implies the output is of sensitive nature and effort should be made
	// to prevent accidental disclosure.
	Sensitive *bool `json:"sensitive,omitempty" yaml:"sensitive,omitempty" mapstructure:"sensitive,omitempty"`

	// Type is the value type of the output.
	Type *OutputType `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type,omitempty"`
}

type OutputType string

const OutputTypeArray OutputType = "array"
const OutputTypeBoolean OutputType = "boolean"
const OutputTypeNumber OutputType = "number"
const OutputTypeString OutputType = "string"
const OutputTypeStruct OutputType = "struct"

var enumValues_OutputType = []any{
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
