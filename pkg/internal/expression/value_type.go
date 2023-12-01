package expression

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"
)

func ValueToString(v *structpb.Value) (string, error) {
	switch v.Kind.(type) {
	case *structpb.Value_StringValue:
		return v.GetStringValue(), nil
	default:
		b, err := json.Marshal(v)
		return string(b), err
	}
}
