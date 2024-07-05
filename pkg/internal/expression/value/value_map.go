package value

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type valueMap struct {
	DefaultFunctions
	v reflect.Value
}

func (v *valueMap) Dig(key string) Value {
	kind := v.v.Type().Key().Kind()

	switch kind {
	case reflect.String:
		mapValue := v.v.MapIndex(reflect.ValueOf(key))
		if !mapValue.IsValid() {
			return NewError(fmt.Errorf("the %q was not found", key))
		}

		return ToValue(mapValue.Interface())

	default:
		return NewError(fmt.Errorf("the map key needs to be %q, but is %q", "string", kind))
	}
}

func (v *valueMap) IsTrue() bool {
	return v.v.Len() > 0
}

func (v *valueMap) IsNull() bool {
	return false
}

func (v *valueMap) Error() error {
	return nil
}

func (v *valueMap) ToString() (string, error) {
	data, err := json.Marshal(v.v)
	return string(data), err
}
