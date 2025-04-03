package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Input struct {
	// AdditionalProperties corresponds to the JSON schema field
	// "additionalProperties".
	AdditionalProperties any `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty" mapstructure:"additionalProperties,omitempty"`

	// Default is the default input value. Its type must match `type`.
	Default any `json:"default,omitempty" yaml:"default,omitempty" mapstructure:"default,omitempty"`

	// Sensitive implies the input is of sensitive nature and effort should be made to
	// prevent accidental disclosure.
	Sensitive *bool `json:"sensitive,omitempty" yaml:"sensitive,omitempty" mapstructure:"sensitive,omitempty"`

	// Type is the value type of the input.
	Type *InputType `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type,omitempty"`
}

func (i *Input) compile() (*proto.Spec_Content_Input, error) {
	protoInput, err := i.compileToProto()
	if err != nil {
		return nil, err
	}
	err = i.verifyDefaultValueMatchesType(protoInput)
	if err != nil {
		return nil, err
	}
	return protoInput, nil
}

func (i *Input) compileToProto() (*proto.Spec_Content_Input, error) {
	protoInput := &proto.Spec_Content_Input{}

	if i.Type == nil {
		return nil, fmt.Errorf("missing input type")
	}

	switch *i.Type {
	case InputTypeBoolean:
		protoInput.Type = proto.ValueType_boolean
	case InputTypeArray:
		protoInput.Type = proto.ValueType_array
	case InputTypeNumber:
		protoInput.Type = proto.ValueType_number
	case InputTypeString:
		protoInput.Type = proto.ValueType_string
	case InputTypeStruct:
		protoInput.Type = proto.ValueType_struct
	default:
		return nil, fmt.Errorf("unsupported input type: %v", i.Type)
	}
	if i.Default != nil {
		protoV, err := (&valueCompiler{i.Default}).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling default %v: %w", i.Default, err)
		}
		protoInput.Default = protoV
	}
	if i.Sensitive != nil && *i.Sensitive {
		protoInput.Sensitive = true
	}
	return protoInput, nil
}

func (i *Input) verifyDefaultValueMatchesType(protoInput *proto.Spec_Content_Input) error {
	if i.Default == nil || protoInput.Default == nil {
		return nil
	}
	if i.Type == nil {
		return nil
	}
	var defaultType InputType
	switch *i.Type {
	case InputTypeBoolean:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_BoolValue); ok {
			defaultType = InputTypeBoolean
		}
	case InputTypeArray:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_ListValue); ok {
			defaultType = InputTypeArray
		}
	case InputTypeNumber:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_NumberValue); ok {
			defaultType = InputTypeNumber
		}
	case InputTypeString:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_StringValue); ok {
			defaultType = InputTypeString
		}
	case InputTypeStruct:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_StructValue); ok {
			defaultType = InputTypeStruct
		}
	default:
		return fmt.Errorf("unsupported type: %v", i.Type)
	}
	if defaultType != *i.Type {
		return fmt.Errorf("input type %v and default value type %v must match", i.Type, defaultType)
	}
	return nil
}

type InputType string

const InputTypeArray InputType = "array"
const InputTypeBoolean InputType = "boolean"
const InputTypeNumber InputType = "number"
const InputTypeString InputType = "string"
const InputTypeStruct InputType = "struct"

var enumValues_InputType = []any{
	"string",
	"number",
	"boolean",
	"struct",
	"array",
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
