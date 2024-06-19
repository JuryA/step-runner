package value

import "errors"

type ValueString struct {
	v string
}

func (v *ValueString) Dig(key string) Value {
	return NewError(errors.New("not supported"))
}

func (v *ValueString) Call(method string, args []Value) Value {
	return NewError(errors.New("not supported"))
}

func (v *ValueString) IsTrue() bool {
	return v.v != ""
}

func (v *ValueString) IsNull() bool {
	return false
}

func (v *ValueString) Error() error {
	return nil
}

func (v *ValueString) ToString() (string, error) {
	return v.v, nil
}
