package expression

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

func ValueToString(v *structpb.Value) (string, error) {
	switch v.Kind.(type) {
	case *structpb.Value_StringValue:
		return v.GetStringValue(), nil
	case *structpb.Value_NumberValue:
		n, _ := json.Marshal(v.GetNumberValue())
		return string(n), nil
	case *structpb.Value_BoolValue:
		b, _ := json.Marshal(v.GetBoolValue())
		return string(b), nil
	case *structpb.Value_StructValue:
		s, err := json.Marshal(v.GetStructValue())
		if err != nil {
			return "", fmt.Errorf("json marshaling struct value %v", v)
		}
		return string(s), nil
	default:
		return "", fmt.Errorf("unsupported type %T", v.Kind)
	}
}
