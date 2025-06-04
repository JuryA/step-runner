package value

import (
	"errors"
	"fmt"
	"iter"
	"math/big"
	"runtime"
	"strconv"
	"sync"
)

type Kind string

var (
	ErrInvalidKind          = errors.New("invalid kind")
	ErrAttributeNotFound    = errors.New("attribute not found")
	ErrArrayIndexOutOfRange = errors.New("array index out of range")
	ErrInvalidKey           = errors.New("invalid key")
	ErrDivisionByZero       = errors.New("division by zero")

	null = Value{kind: NullKind, v: nil}

	bigfloat = sync.Pool{
		New: func() any {
			return new(big.Float)
		},
	}

	boolOrder  = map[bool]int{false: 0, true: 1}
	valueOrder = map[Kind]int{NullKind: 0, BoolKind: 1, NumberKind: 2, StringKind: 3, ArrayKind: 4, ObjectKind: 5, FuncKind: 6}
)

type (
	Function = func(...Value) (Value, error)
)

const (
	// JSON types
	StringKind Kind = "string"
	NumberKind Kind = "number"
	BoolKind   Kind = "bool"
	ObjectKind Kind = "object"
	ArrayKind  Kind = "array"
	NullKind   Kind = "null"

	// Our types
	FuncKind Kind = "func"

	defaultPrecision = 128
)

// Value represents a value of the following types:
// string, number, bool, object, array, null, func
type Value struct {
	kind  Kind
	v     any
	marks uint16
}

// Kind returns the value's Kind
func (v Value) Kind() Kind {
	return v.kind
}

// String returns a new string value.
func String(v string) Value {
	return Value{kind: StringKind, v: v}
}

// Bool returns a new bool value.
func Bool(v bool) Value {
	return Value{kind: BoolKind, v: v}
}

// Number returns a new number value.
func Number[T ~int | ~int64 | ~uint64 | ~float64](v T) Value {
	switch v := any(v).(type) {
	case int:
		return Value{kind: NumberKind, v: getBigFloat().SetInt64(int64(v))}
	case int64:
		return Value{kind: NumberKind, v: getBigFloat().SetInt64(v)}
	case uint64:
		return Value{kind: NumberKind, v: getBigFloat().SetUint64(v)}
	case float64:
		return Value{kind: NumberKind, v: getBigFloat().SetFloat64(v)}
	}

	panic("unsupported number") // impossible
}

// Object returns a new object value.
func Object(v Mapper) Value {
	return Value{kind: ObjectKind, v: v}
}

// Array returns a new array value.
func Array(v Indexer) Value {
	return Value{kind: ArrayKind, v: v}
}

// Null returns null value.
func Null() Value {
	return null
}

// Func returns a new func value.
func Func(fn func(...Value) (Value, error)) Value {
	return Value{kind: FuncKind, v: fn}
}

// Marks returns the values marks.
func (v Value) Marks(recursive bool) uint16 {
	result := v.marks
	if !recursive {
		return result
	}

	var seq iter.Seq[Value]
	switch v.kind {
	case ObjectKind:
		seq = v.v.(Mapper).Values()
	case ArrayKind:
		seq = v.v.(Indexer).Values()
	}

	if seq == nil {
		return result
	}
	for val := range seq {
		for marks := range val.Marks(true) {
			result |= marks
		}
	}

	return result
}

// HasMark returns whether the valve has the specified mark.
func (v Value) HasMarks(marks uint16) bool {
	return v.marks&marks == marks
}

// WithMarks returns a copy of the value with the specified marks.
//
// Marks are not considered part of the value for equality/comparison.
func (v Value) WithMarks(marks uint16) Value {
	newVal := Value{kind: v.kind, v: v.v, marks: v.marks | marks}

	switch newVal.kind {
	case ObjectKind:
		newVal.v = v.v.(Mapper).WithMarks(marks)

	case ArrayKind:
		newVal.v = v.v.(Indexer).WithMarks(marks)
	}

	return newVal
}

// Attr returns an attributes value by name.
func (v Value) Attr(name string) (Value, error) {
	if v.kind != ObjectKind {
		return null, fmt.Errorf("%w: attribute access requires object not %v", ErrInvalidKind, v.kind)
	}

	return v.v.(Mapper).Attr(name)
}

// Index returns an array's value by index.
func (v Value) Index(i int) (Value, error) {
	switch v.kind {
	case ArrayKind:
		return v.v.(Indexer).Index(i)

	case StringKind:
		for idx, r := range v.v.(string) {
			if i == idx {
				return String(string(r)).WithMarks(v.marks), nil
			}
		}

		return null, ErrArrayIndexOutOfRange
	}

	return null, fmt.Errorf("%w: index access requires array not %v", ErrInvalidKind, v.kind)
}

// Get returns either an attribute or index value.
//
// The key can therefore be a number or string.
func (v Value) Get(key Value) (Value, error) {
	switch v.kind {
	case ObjectKind:
		if key.kind != StringKind {
			return null, fmt.Errorf("%w: object requires string key not %v", ErrInvalidKey, key.kind)
		}

		return v.Attr(key.v.(string))

	case ArrayKind, StringKind:
		if key.kind != NumberKind {
			return null, fmt.Errorf("%w: %v requires number key not %v", ErrInvalidKey, v.kind, key.kind)
		}

		idx, _ := key.v.(*big.Float).Int64()

		return v.Index(int(idx))

	default:
		return null, fmt.Errorf("%w: attribute access only works for object, array not %v", ErrInvalidKind, v.kind)
	}
}

func (v Value) MustIsTrue() bool {
	tru, err := v.IsTrue()
	if err != nil {
		panic(err)
	}

	return tru
}

// True returns if the value is "truthy" depending on the type:
//
// bool:   true if true
// string: true if length > 0
// number: true if value > 0
// object: true if length > 0
// array:  true of length > 0
// null:   false
func (v Value) IsTrue() (bool, error) {
	switch v.kind {
	case BoolKind:
		return v.v.(bool), nil

	case StringKind:
		return len(v.v.(string)) > 0, nil

	case NumberKind:
		return v.v.(*big.Float).Sign() == 1, nil

	case ObjectKind:
		return v.v.(Mapper).Len() > 0, nil

	case ArrayKind:
		return v.v.(Indexer).Len() > 0, nil

	case NullKind:
		return false, nil

	case FuncKind:
		return v.v.(Function) != nil, nil

	default:
		panic("unknown value type (truthy)")
	}
}

// String returns a string representation of the value.
//
// For values that are of kind String, this returns the actual string value.
//
// Object and Array kinds are serialized to JSON.
func (v Value) String() string {
	complexFmt := func(v Value) string {
		switch v.Kind() {
		case StringKind:
			return strconv.Quote(v.String())

		default:
			return v.String()
		}
	}

	switch v.kind {
	case StringKind:
		return v.v.(string)

	case NullKind:
		return "<null>"

	case NumberKind:
		return v.v.(*big.Float).Text('g', 6)

	case FuncKind:
		return "<func>"

	case ArrayKind:
		return pretty(v, 0, false, func(v Value, depth int, indent bool) string {
			return complexFmt(v)
		})

	case ObjectKind:
		return pretty(v, 0, false, func(v Value, depth int, indent bool) string {
			return complexFmt(v)
		})

	default:
		return fmt.Sprintf("%v", v.v)
	}
}

// Interface returns the underlying Go value.
func (v Value) Interface() any {
	return v.v
}

// getBigFloat initializes a big.Float from a pool. If a float was about to be
// GC'd at the same time as we need a new one, we instead re-use it.
func getBigFloat() *big.Float {
	f := bigfloat.Get().(*big.Float)
	f.SetPrec(defaultPrecision)
	f.SetUint64(0)

	runtime.SetFinalizer(f, func(f *big.Float) {
		runtime.SetFinalizer(f, nil)
		bigfloat.Put(f)
	})

	return f
}
