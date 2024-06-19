package value

import (
	"errors"
)

type ValueNil struct {
}

func (v *ValueNil) Dig(key string) Value {
	// going deeper is always nil
	return v
}

func (v *ValueNil) Call(method string, args []Value) Value {
	return NewError(errors.New("not supported"))
}

func (v *ValueNil) IsTrue() bool {
	return false
}

func (v *ValueNil) IsNull() bool {
	return true
}

func (v *ValueNil) Error() error {
	return nil
}

func (v *ValueNil) ToString() (string, error) {
	return "nil", nil
}
