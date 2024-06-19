package value

import (
	"fmt"
	"reflect"
)

func ToValue2(v any) (Value, error) {
	switch x := v.(type) {
	case string:
		return &ValueString{v: x}, nil
	case int64:
		return &ValueInt{v: x}, nil
	case int:
		return &ValueInt{v: int64(x)}, nil
	case bool:
		return &ValueBool{v: x}, nil
	case nil:
		return &ValueNil{}, nil
	case error:
		return &ValueError{v: x}, nil
	}

	val := reflect.ValueOf(v)
	val = reflect.Indirect(val) // drop pointer
	switch val.Kind() {
	case reflect.Map:
		return &valueMap{v: val}, nil
	case reflect.Struct:
		return &valueStruct{v: val}, nil
	}

	return nil, fmt.Errorf("unsupported type: %T", v)
}

func ToValue(v any) Value {
	res, err := ToValue2(v)
	if err != nil {
		return NewError(err)
	}
	return res
}
