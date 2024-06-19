package value

import (
	"errors"
	"strconv"
)

type ValueInt struct {
	v int64
}

func (v *ValueInt) Dig(key string) Value {
	return NewError(errors.New("not supported"))
}

func (v *ValueInt) Call(method string, args []Value) Value {
	return NewError(errors.New("not supported"))
}

func (v *ValueInt) IsTrue() bool {
	return v.v != 0
}

func (v *ValueInt) IsNull() bool {
	return false
}

func (v *ValueInt) Error() error {
	return nil
}

func (v *ValueInt) ToString() (string, error) {
	return strconv.FormatInt(v.v, 10), nil
}
