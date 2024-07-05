package value

type ValueNil struct {
	DefaultFunctions
}

func (v *ValueNil) Dig(key string) Value {
	// going deeper is always nil
	return v
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
