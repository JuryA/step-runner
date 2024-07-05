package value

import (
	"fmt"
)

type Value interface {
	Dig(key string) Value
	Call(method string, args []Value) Value
	IsTrue() bool
	IsNull() bool
	Error() error
	ToString() (string, error)
}
