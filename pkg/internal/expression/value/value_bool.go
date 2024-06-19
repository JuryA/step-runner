package value

import (
	"errors"
)

type ValueBool struct {
	v bool
}

func (v *ValueBool) Dig(key string) Value {
	return NewError(errors.New("not supported"))
}

func (v *ValueBool) Call(method string, args []Value) Value {
	return NewError(errors.New("not supported"))
}

func (v *ValueBool) IsTrue() bool {
	return v.v
}

func (v *ValueBool) IsNull() bool {
	return false
}

func (v *ValueBool) Error() error {
	return nil
}

func (v *ValueBool) ToString() (string, error) {
	if v.v {
		return "true", nil
	} else {
		return "false", nil
	}
}
