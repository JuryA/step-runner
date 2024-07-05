package value

import (
	"reflect"
)

func AutoCaller(self Value, method string, args []Value) (ret Value) {
	reflectSelf := reflect.ValueOf(self)
	reflectMethod := reflectSelf.MethodByName("Call_" + method)
	if !reflectMethod.IsValid() {
		return NewErrorf("Method %q not found in %q", method, reflectSelf.Type().Name())
	}

	reflectArgs := []reflect.Value{
		reflect.ValueOf(self),
	}
	for _, arg := range args {
		reflectArgs = append(reflectArgs, reflect.ValueOf(arg))
	}

	defer func() {
		if err := recover(); err != nil {
			ret = NewErrorf("%v", err)
		}
	}()

	returnValues := reflectMethod.Call(reflectArgs)
	if len(returnValues) != 1 {
		return NewErrorf("Method %q.%q returned %d arguments instead of 1",
			method, reflectSelf.Type().Name(), len(returnValues))
	}

	switch x := returnValues[0].Interface().(type) {
	case Value:
		return x

	default:
		return NewErrorf("Method %q.%q needs to return Value instead of %v",
			method, reflectSelf.Type().Name(), returnValues[0].Type().Name())
	}
}
