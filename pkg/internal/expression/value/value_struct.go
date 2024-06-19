package value

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type valueStruct struct {
	v reflect.Value
}

func digStruct(value reflect.Value, key string) (any, error) {
	if value.Kind() != reflect.Struct {
		return nil, errors.New("not a struct")
	}

	for i := 0; i < value.Type().NumField(); i += 1 {
		structField := value.Type().Field(i)
		if !structField.IsExported() {
			continue
		} else if structField.Anonymous {
			res, err := digStruct(value.FieldByIndex(structField.Index), key)
			if err == nil {
				return res, nil
			}
		} else if tag, ok := structField.Tag.Lookup("json"); ok && tag != "-" {
			tagName, _, _ := strings.Cut(tag, ",")
			if tagName == key {
				return value.FieldByIndex(structField.Index).Interface(), nil
			}
		}
	}

	return nil, fmt.Errorf("the %q was not found", key)
}

func (v *valueStruct) Dig(key string) Value {
	value, err := digStruct(v.v, key)
	if err != nil {
		return NewError(err)
	}

	return ToValue(value)
}

func (v *valueStruct) Call(method string, args []Value) Value {
	if res := valueCall(v, method, args); res != nil {
		return res
	}
	return NewError(errors.New("not supported"))
}

func (v *valueStruct) IsTrue() bool {
	return true
}

func (v *valueStruct) IsNull() bool {
	return false
}

func (v *valueStruct) Error() error {
	return nil
}

func (v *valueStruct) ToString() (string, error) {
	data, err := json.Marshal(v.v)
	return string(data), err
}
