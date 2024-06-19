package value

import (
	"fmt"
)

type Value interface {
	Dig(key string) Value
	IsTrue() bool
	IsNull() bool
	Error() error
	ToString() (string, error)
}

func Equals(value, otherValue Value) Value {
	value1String, value1Err := value.ToString()
	value2String, value2Err := otherValue.ToString()
	if value1Err != nil && value2Err != nil {
		return &ValueError{v: fmt.Errorf("Many errors: %q, %q", value1Err, value2Err)}
	} else if value1Err != nil {
		return &ValueError{v: value1Err}
	} else if value2Err != nil {
		return &ValueError{v: value2Err}
	} else {
		return &ValueBool{v: value1String == value2String}
	}
}
