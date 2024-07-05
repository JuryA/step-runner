package value

import (
	"errors"
	"fmt"
)

type ValueError struct {
	DefaultFunctions
	v error
}

func (v *ValueError) Dig(key string) Value {
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

func NewErrorf(format string, a ...any) Value {
	return NewError(fmt.Errorf(format, a...))
}
