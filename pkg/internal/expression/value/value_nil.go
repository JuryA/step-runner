package value

import (
	"errors"
)

type ValueNil struct {
}

func (v *ValueNil) Dig(key string) Value {
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
