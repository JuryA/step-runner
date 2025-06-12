package expression

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

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

func ObjectToProtoValue(object any) (*structpb.Value, error) {
	switch v := object.(type) {
	case structpb.Value:
		return &v, nil
	case *structpb.Value:
		return v, nil
	case map[string]*structpb.Value:
		return structpb.NewStructValue(&structpb.Struct{Fields: v}), nil
	default:
		return structpb.NewValue(object)
	}
}

func digStruct(value reflect.Value, key string) (any, error) {
	for i := 0; i < value.Type().NumField(); i += 1 {
		structField := value.Type().Field(i)
		if !structField.IsExported() {
			continue
		}

		if structField.Anonymous {
			res, err := DigObject(value.FieldByIndex(structField.Index).Interface(), key)
			if err == nil {
				return res, nil
			}
		}

		fieldName := structField.Name

		if tag, ok := structField.Tag.Lookup("json"); ok {
			if tag == "-" {
				continue
			}
			tagName, _, _ := strings.Cut(tag, ",")
			if tagName != "" {
				fieldName = tagName
			}
		}

		if fieldName == key {
			return value.FieldByIndex(structField.Index).Interface(), nil
		}
	}
	return nil, fmt.Errorf("the %q was not found", key)
}

func digMap(value reflect.Value, key string) (any, error) {
	switch value.Type().Key().Kind() {
	case reflect.String:
		mapValue := value.MapIndex(reflect.ValueOf(key))
		if !mapValue.IsValid() {
			return nil, fmt.Errorf("the %q was not found", key)
		}

		return mapValue.Interface(), nil

	default:
		return nil, fmt.Errorf("the map key needs to be %q, but is %q", "string", value.Type().Key().Kind())
	}
}

func digProtoValue(value *structpb.Value, key string) (any, error) {
	switch value.Kind.(type) {
	case *structpb.Value_StructValue:
		structValue := value.GetStructValue()
		if fieldValue, ok := structValue.Fields[key]; ok {
			return fieldValue, nil
		}
		return nil, fmt.Errorf("the %q was not found", key)

	default:
		return nil, fmt.Errorf("the %q is not map to get %q key", value.Kind, key)
	}
}

func DigObject(object any, key string) (any, error) {
	if v, ok := object.(*structpb.Value); ok {
		return digProtoValue(v, key)
	}

	value := reflect.ValueOf(object)
	value = reflect.Indirect(value) // drop pointer

	switch value.Type().Kind() {
	case reflect.Struct:
		return digStruct(value, key)

	case reflect.Map:
		return digMap(value, key)

	default:
		return nil, fmt.Errorf("%q is not supported", value.Type().Kind())
	}
}

func DigProtoValue(object any, key string) (*structpb.Value, error) {
	res, err := DigObject(object, key)
	if err != nil {
		return nil, err
	}
	return ObjectToProtoValue(res)
}
