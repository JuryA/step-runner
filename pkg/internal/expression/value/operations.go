package value

import (
	"cmp"
	"fmt"
	"math/big"
	"slices"
)

// Negate negates the value.
//
// For boolean values: !value
// For number values:  -value
func (v Value) Negate() (Value, error) {
	switch v.kind {
	case BoolKind:
		return Value{kind: BoolKind, v: !v.v.(bool), marks: v.marks}, nil

	case NumberKind:
		return Value{kind: NumberKind, v: getBigFloat().Neg(v.v.(*big.Float)), marks: v.marks}, nil

	default:
		return null, ErrInvalidKind
	}
}

// Not returns true if the value is falsy.
func (v Value) Not() Value {
	return Value{kind: BoolKind, v: !v.MustIsTrue(), marks: v.Marks(true)}
}

// LessThan returns true if v is less than other.
func (v Value) LessThan(other Value) Value {
	return Value{kind: BoolKind, v: v.compare(other) < 0, marks: v.Marks(true) | other.Marks(true)}
}

// LessThanEqual returns true if v is less than or equal to other.
func (v Value) LessThanEqual(other Value) Value {
	return Value{kind: BoolKind, v: v.compare(other) <= 0, marks: v.Marks(true) | other.Marks(true)}
}

// Equal returns true if the values are identical.
func (v Value) Equal(other Value) Value {
	return Value{kind: BoolKind, v: v.compare(other) == 0, marks: v.Marks(true) | other.Marks(true)}
}

// GreaterThan returns true if v is greater than other.
func (v Value) GreaterThan(other Value) Value {
	return Value{kind: BoolKind, v: v.compare(other) > 0, marks: v.Marks(true) | other.Marks(true)}
}

// GreaterThanEqual returns true if v is greater than or equal to other.
func (v Value) GreaterThanEqual(other Value) Value {
	return Value{kind: BoolKind, v: v.compare(other) >= 0, marks: v.Marks(true) | other.Marks(true)}
}

func (v Value) compare(other Value) int {
	if v.kind != other.kind {
		return cmp.Compare(valueOrder[v.kind], valueOrder[other.kind])
	}

	switch v.kind {
	case StringKind:
		return cmp.Compare(v.v.(string), other.v.(string))

	case NumberKind:
		a := v.v.(*big.Float)
		b := other.v.(*big.Float)

		af, aa := a.Float64()
		bf, ba := b.Float64()
		if aa == big.Exact || ba == big.Exact {
			return cmp.Compare(af, bf)
		}

		return a.Cmp(b)

	case BoolKind:
		return cmp.Compare(boolOrder[v.v.(bool)], boolOrder[other.v.(bool)])

	case ObjectKind:
		m1 := v.v.(Mapper)
		m2 := other.v.(Mapper)

		// compare by length
		if m1.Len() != m2.Len() {
			return cmp.Compare(m1.Len(), m2.Len())
		}

		keyCmp := slices.CompareFunc(slices.Collect(m1.Keys()), slices.Collect(m2.Keys()), Value.compare)
		if keyCmp != 0 {
			return keyCmp
		}

		return slices.CompareFunc(slices.Collect(m1.Values()), slices.Collect(m2.Values()), Value.compare)

	case ArrayKind:
		return slices.CompareFunc(slices.Collect(v.v.(Indexer).Values()), slices.Collect(other.v.(Indexer).Values()), Value.compare)

	case NullKind:
		return 0

	default:
		panic("unknown value type (compare)")
	}
}

// Add adds x and returns a new value with the result.
func (v Value) Add(x Value) (Value, error) {
	switch {
	case v.kind == NumberKind && x.kind == NumberKind:
		return Value{
			kind:  NumberKind,
			v:     getBigFloat().Add(v.v.(*big.Float), x.v.(*big.Float)),
			marks: v.marks | x.marks,
		}, nil

	case v.kind == StringKind && x.kind == StringKind:
		return Value{kind: StringKind, v: v.v.(string) + x.v.(string), marks: v.marks | x.marks}, nil

	default:
		return null, fmt.Errorf("%w: %v + %v unsupported", ErrInvalidKind, v.kind, x.kind)
	}
}

// Subtract subtracts x and returns a new value with the result.
func (v Value) Subtract(x Value) (Value, error) {
	if v.kind != NumberKind || x.kind != NumberKind {
		return null, fmt.Errorf("%w: %v - %v unsupported", ErrInvalidKind, v.kind, x.kind)
	}

	return Value{
		kind:  NumberKind,
		v:     getBigFloat().Sub(v.v.(*big.Float), x.v.(*big.Float)),
		marks: v.marks | x.marks,
	}, nil
}

// Multiply multiplies by x and returns a new value with the result.
func (v Value) Multiply(x Value) (Value, error) {
	if v.kind != NumberKind || x.kind != NumberKind {
		return null, fmt.Errorf("%w: %v * %v unsupported", ErrInvalidKind, v.kind, x.kind)
	}

	return Value{
		kind:  NumberKind,
		v:     getBigFloat().Mul(v.v.(*big.Float), x.v.(*big.Float)),
		marks: v.marks | x.marks,
	}, nil
}

// Divide divides by x and returns a new value with the result.
func (v Value) Divide(x Value) (Value, error) {
	if v.kind != NumberKind || x.kind != NumberKind {
		return null, fmt.Errorf("%w: %v / %v unsupported", ErrInvalidKind, v.kind, x.kind)
	}

	if x.v.(*big.Float).Sign() == 0 {
		return null, ErrDivisionByZero
	}

	return Value{
		kind:  NumberKind,
		v:     getBigFloat().Quo(v.v.(*big.Float), x.v.(*big.Float)),
		marks: v.marks | x.marks,
	}, nil
}

// Modulo calculates v % x and returns a new value with the result.
func (v Value) Modulo(x Value) (Value, error) {
	if v.kind != NumberKind || x.kind != NumberKind {
		return null, fmt.Errorf("%w: %v %% %v unsupported", ErrInvalidKind, v.kind, x.kind)
	}

	divisor := x.v.(*big.Float)
	if divisor.Sign() == 0 {
		return null, ErrDivisionByZero
	}

	dividend := v.v.(*big.Float)
	quotient := getBigFloat().Quo(dividend, divisor)

	truncate := new(big.Int)
	quotient.Int(truncate) // truncate towards zero
	quotient.SetInt(truncate)
	quotient.Mul(divisor, quotient)

	return Value{
		kind:  NumberKind,
		v:     getBigFloat().Sub(dividend, quotient),
		marks: v.marks | x.marks,
	}, nil
}

// Call calls the function value with the specified arguments.
func (v Value) Call(args ...Value) (Value, error) {
	if v.kind != FuncKind {
		return null, fmt.Errorf("%w: cannot call on type %v", ErrInvalidKind, v.kind)
	}

	return v.v.(Function)(args...)
}
