package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
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

func (o *Output) compile() (*proto.Spec_Content_Output, error) {
	protoOutput, err := o.compileToProto()
	if err != nil {
		return nil, err
	}
	err = o.verifyDefaultValueMatchesType(protoOutput)
	if err != nil {
		return nil, err
	}
	return protoOutput, nil
}

func (o *Output) compileToProto() (*proto.Spec_Content_Output, error) {
	protoOutput := &proto.Spec_Content_Output{}

	if o.Type == nil {
		return nil, fmt.Errorf("missing output type")
	}

	switch *o.Type {
	case OutputTypeBoolean:
		protoOutput.Type = proto.ValueType_boolean
	case OutputTypeArray:
		protoOutput.Type = proto.ValueType_array
	case OutputTypeNumber:
		protoOutput.Type = proto.ValueType_number
	case OutputTypeString:
		protoOutput.Type = proto.ValueType_string
	case OutputTypeStruct:
		protoOutput.Type = proto.ValueType_struct
	default:
		return nil, fmt.Errorf("unsupported output type: %v", o.Type)
	}
	if o.Default != nil {
		protoV, err := (&valueCompiler{o.Default}).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling default %v: %w", o.Default, err)
		}
		protoOutput.Default = protoV
	}
	if o.Sensitive != nil && *o.Sensitive {
		protoOutput.Sensitive = true
	}
	return protoOutput, nil
}

func (o *Output) verifyDefaultValueMatchesType(protoOutput *proto.Spec_Content_Output) error {
	if o.Default == nil || protoOutput.Default == nil {
		return nil
	}
	if o.Type == nil {
		return nil
	}
	var defaultType OutputType
	switch *o.Type {
	case OutputTypeBoolean:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_BoolValue); ok {
			defaultType = OutputTypeBoolean
		}
	case OutputTypeArray:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_ListValue); ok {
			defaultType = OutputTypeArray
		}
	case OutputTypeNumber:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_NumberValue); ok {
			defaultType = OutputTypeNumber
		}
	case OutputTypeString:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_StringValue); ok {
			defaultType = OutputTypeString
		}
	case OutputTypeStruct:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_StructValue); ok {
			defaultType = OutputTypeStruct
		}
	default:
		return fmt.Errorf("unsupported type: %v", o.Type)
	}
	if defaultType != *o.Type {
		return fmt.Errorf("output type %v and default value type %v must match", o.Type, defaultType)
	}
	return nil
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
