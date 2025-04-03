package schema

import "google.golang.org/protobuf/types/known/structpb"

type valueCompiler struct {
	v any
}

func (value *valueCompiler) compile() (*structpb.Value, error) {
	var simplifyTypes func(any) any
	simplifyTypes = func(v any) any {
		// Map a few types from our model to ones that
		// structpb can handle.
		switch v := v.(type) {
		case *string:
			if v != nil {
				return *v
			}
		case StepInputs:
			simpleMap := map[string]any{}
			for k, v := range v {
				simpleMap[k] = simplifyTypes(v)
			}
			return simpleMap
		}
		return v
	}
	// We let structpb do all the heavy lifting
	// and verify the type matches our
	// expectations later.
	return structpb.NewValue(simplifyTypes(value.v))
}
