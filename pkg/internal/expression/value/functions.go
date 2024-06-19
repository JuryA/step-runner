package value

import "fmt"

func valueCallOrDefault(v Value, method string, args []Value) Value {
	if len(args) != 1 {
		return NewError(fmt.Errorf("invalid number of arguments (%d) to orDefault()", len(args)))
	}

	if v.IsTrue() {
		return v
	} else {
		return args[0]
	}
}

func valueCallStr(v Value, method string, args []Value) Value {
	if len(args) != 0 {
		return NewError(fmt.Errorf("invalid number of arguments (%d) to str()", len(args)))
	}
	x, err := v.ToString()
	if err != nil {
		return NewError(err)
	}
	return ToValue(x)
}

func valueCall(v Value, method string, args []Value) Value {
	switch method {
	case "orDefault":
		return valueCallOrDefault(v, method, args)

	case "str":
		return valueCallStr(v, method, args)

	default:
		return nil
	}
}
