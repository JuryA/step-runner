package value

import (
	"errors"
)

type ValueError struct {
	v error
}

func (v *ValueError) Dig(key string) Value {
	return NewError(errors.New("not supported"))
}

func (v *ValueError) Call(method string, args []Value) Value {
	if res := valueCall(v, method, args); res != nil {
		return res
	}
	return NewError(errors.New("not supported"))
}

func (v *ValueError) IsTrue() bool {
	return false
}

func (v *ValueError) IsNull() bool {
	return false
}

func (v *ValueError) Error() error {
	return v.v
}

func (v *ValueError) ToString() (string, error) {
	return v.v.Error(), nil
}

func NewError(err error) Value {
	return &ValueError{v: err}
}
