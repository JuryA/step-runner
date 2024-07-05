package value

type DefaultFunctions struct {
}

func (d *DefaultFunctions) Call_orDefault(self Value, defaultVal Value) Value {
	if self.IsTrue() {
		return self
	} else {
		return defaultVal
	}
}

func (d *DefaultFunctions) Call_str(self Value) Value {
	x, err := self.ToString()
	if err != nil {
		return NewError(err)
	}
	return ToValue(x)
}
