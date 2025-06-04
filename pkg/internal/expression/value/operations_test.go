package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNegate(t *testing.T) {
	negateFalse, err := Bool(false).Negate()
	require.NoError(t, err)
	assert.True(t, negateFalse.MustIsTrue())

	negateTrue, err := Bool(true).Negate()
	require.NoError(t, err)
	assert.False(t, negateTrue.MustIsTrue())

	negatePositiveNumber, err := Number(1).Negate()
	require.NoError(t, err)
	assert.False(t, negatePositiveNumber.MustIsTrue())

	negateNegativeNumber, err := Number(-1).Negate()
	require.NoError(t, err)
	assert.True(t, negateNegativeNumber.MustIsTrue())

	_, err = String("1").Negate()
	require.ErrorIs(t, err, ErrInvalidKind)
}

func TestNot(t *testing.T) {
	assert.True(t, Number(1).Equal(Number(1)).MustIsTrue())
	assert.False(t, Number(0).Equal(Number(1)).MustIsTrue())
	assert.False(t, Number(1).Equal(Number(0)).MustIsTrue())
}

func TestLessThan(t *testing.T) {
	assert.False(t, Number(1).LessThan(Number(1)).MustIsTrue())
	assert.True(t, Number(0).LessThan(Number(1)).MustIsTrue())
	assert.False(t, Number(1).LessThan(Number(0)).MustIsTrue())
}

func TestLessThanEqual(t *testing.T) {
	assert.True(t, Number(1).LessThanEqual(Number(1)).MustIsTrue())
	assert.True(t, Number(0).LessThanEqual(Number(1)).MustIsTrue())
	assert.False(t, Number(1).LessThanEqual(Number(0)).MustIsTrue())
}

func TestEqual(t *testing.T) {
	assert.True(t, Number(1).Equal(Number(1)).MustIsTrue())
	assert.False(t, Number(0).Equal(Number(1)).MustIsTrue())
	assert.False(t, Number(1).Equal(Number(0)).MustIsTrue())
}

func TestGreaterThan(t *testing.T) {
	assert.False(t, Number(1).GreaterThan(Number(1)).MustIsTrue())
	assert.False(t, Number(0).GreaterThan(Number(1)).MustIsTrue())
	assert.True(t, Number(1).GreaterThan(Number(0)).MustIsTrue())
}

func TestGreaterThanEqual(t *testing.T) {
	assert.True(t, Number(1).GreaterThanEqual(Number(1)).MustIsTrue())
	assert.False(t, Number(0).GreaterThanEqual(Number(1)).MustIsTrue())
	assert.True(t, Number(1).GreaterThanEqual(Number(0)).MustIsTrue())
}

func TestCompare(t *testing.T) {
	tests := []struct {
		v1, v2 Value
		result any
	}{
		// type order
		{Null(), Bool(false), -1},
		{Null(), Bool(true), -1},
		{Bool(false), Number(0), -1},
		{Bool(true), Number(0), -1},
		{Number(0), String(""), -1},
		{Number(0), String("abc"), -1},
		{String(""), NewList(Null()), -1},
		{String(""), NewList(Number(1)), -1},
		{NewList(Null()), MustMap(String(""), Null()), -1},
		{NewList(Number(1)), MustMap(String("abc"), Number(1)), -1},

		// string
		{String("a"), String("b"), -1},
		{String("a"), String("a"), 0},
		{String("b"), String("a"), 1},

		// integer
		{Number(0), Number(1), -1},
		{Number(1), Number(1), 0},
		{Number(1), Number(0), 1},

		// float
		{Number(0.0), Number(1.0), -1},
		{Number(1.0), Number(1.0), 0},
		{Number(1.0), Number(0.0), 1},

		// mixed int/float
		{Number(0), Number(1.0), -1},
		{Number(1.0), Number(1), 0},
		{Number(1.0), Number(0), 1},

		// bool
		{Bool(false), Bool(true), -1},
		{Bool(true), Bool(true), 0},
		{Bool(true), Bool(false), 1},

		// object length
		{MustMap(String("a"), Null()), MustMap(String(""), Null(), String("b"), Null()), -1},
		{MustMap(String(""), Null(), String("b"), Null()), MustMap(String("a"), Null()), 1},

		// object keys
		{MustMap(String("a"), Null()), MustMap(String("b"), Null()), -1},
		{MustMap(String("a"), Null()), MustMap(String("a"), Null()), 0},
		{MustMap(String("b"), Null()), MustMap(String("a"), Null()), 1},

		// object values
		{MustMap(String("a"), Number(0)), MustMap(String("a"), Number(1)), -1},
		{MustMap(String("a"), Number(1)), MustMap(String("a"), Number(1)), 0},
		{MustMap(String("a"), Number(1)), MustMap(String("a"), Number(0)), 1},

		// array length
		{NewList(Null()), NewList(Null(), Null()), -1},
		{NewList(Null(), Null()), NewList(Null()), 1},

		// array values
		{NewList(Number(0)), NewList(Number(1)), -1},
		{NewList(Number(1)), NewList(Number(1)), 0},
		{NewList(Number(1)), NewList(Number(0)), 1},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.result, tc.v1.compare(tc.v2), "compare(%v, %v)", tc.v1, tc.v2)
	}

	require.Panics(t, func() {
		Value{}.compare(Value{})
	})
}

func TestAdd(t *testing.T) {
	addString, err := String("foo").Add(String("bar"))
	require.NoError(t, err)
	assert.Equal(t, "foobar", addString.String())

	addNumber, err := Number(1).Add(Number(5))
	require.NoError(t, err)
	assert.True(t, addNumber.Equal(Number(6)).MustIsTrue())

	_, err = Bool(true).Add(Bool(true))
	require.ErrorIs(t, err, ErrInvalidKind)
}

func TestSubtract(t *testing.T) {
	subtractNumber, err := Number(5).Subtract(Number(1))
	require.NoError(t, err)
	assert.True(t, subtractNumber.Equal(Number(4)).MustIsTrue())

	_, err = Bool(true).Subtract(Bool(true))
	require.ErrorIs(t, err, ErrInvalidKind)
}

func TestMultiply(t *testing.T) {
	subtractNumber, err := Number(1).Multiply(Number(100))
	require.NoError(t, err)
	assert.True(t, subtractNumber.Equal(Number(100)).MustIsTrue())

	_, err = Bool(true).Multiply(Bool(true))
	require.ErrorIs(t, err, ErrInvalidKind)
}

func TestDivide(t *testing.T) {
	divideNumber, err := Number(100).Divide(Number(1))
	require.NoError(t, err)
	assert.True(t, divideNumber.Equal(Number(100)).MustIsTrue())

	divideNumberZero, err := Number(100).Divide(Number(0))
	require.ErrorIs(t, err, ErrDivisionByZero)
	assert.True(t, divideNumberZero.Equal(Null()).MustIsTrue())

	_, err = Bool(true).Divide(Bool(true))
	require.ErrorIs(t, err, ErrInvalidKind)
}

func TestModulo(t *testing.T) {
	moduloNumber, err := Number(5).Modulo(Number(3))
	require.NoError(t, err)
	assert.True(t, moduloNumber.Equal(Number(2)).MustIsTrue())

	moduloNumberZero, err := Number(5).Modulo(Number(0))
	require.ErrorIs(t, err, ErrDivisionByZero)
	assert.True(t, moduloNumberZero.Equal(Null()).MustIsTrue())

	_, err = Bool(true).Modulo(Bool(true))
	require.ErrorIs(t, err, ErrInvalidKind)
}

func TestCall(t *testing.T) {
	fn := Func(func(v ...Value) (Value, error) {
		return Number(len(v)), assert.AnError
	})

	oneArg, err := fn.Call(String("foo"))
	assert.ErrorIs(t, err, assert.AnError)
	assert.True(t, oneArg.Equal(Number(1)).MustIsTrue())

	twoArg, err := fn.Call(String("foo"), String("bar"))
	assert.ErrorIs(t, err, assert.AnError)
	assert.True(t, twoArg.Equal(Number(2)).MustIsTrue())

	_, err = Bool(true).Call(Bool(true))
	require.ErrorIs(t, err, ErrInvalidKind)
}

func BenchmarkArithmetic(b *testing.B) {
	ix := int64(1024)
	iy := int64(2046)

	vx := Number(1024)
	vy := Number(2046)

	b.Run("base_add_int64", func(b *testing.B) {
		for b.Loop() {
			add(ix, iy)
		}
	})

	b.Run("add_int64", func(b *testing.B) {
		for b.Loop() {
			vx.Add(vy)
		}
	})

	b.Run("base_sub_int64", func(b *testing.B) {
		for b.Loop() {
			sub(ix, iy)
		}
	})

	b.Run("sub_int64", func(b *testing.B) {
		for b.Loop() {
			vx.Subtract(vy)
		}
	})

	b.Run("base_mul_int64", func(b *testing.B) {
		for b.Loop() {
			mul(ix, iy)
		}
	})

	b.Run("mul_int64", func(b *testing.B) {
		for b.Loop() {
			vx.Multiply(vy)
		}
	})

	b.Run("base_div_int64", func(b *testing.B) {
		for b.Loop() {
			div(ix, iy)
		}
	})

	b.Run("mul_int64", func(b *testing.B) {
		for b.Loop() {
			vx.Divide(vy)
		}
	})

	b.Run("base_mod_int64", func(b *testing.B) {
		for b.Loop() {
			mod(ix, iy)
		}
	})

	b.Run("mod_int64", func(b *testing.B) {
		for b.Loop() {
			vx.Modulo(vy)
		}
	})
}

func add[T ~int64 | ~uint64 | ~float32 | ~float64](a, b T) T { return a + b }
func sub[T ~int64 | ~uint64 | ~float32 | ~float64](a, b T) T { return a - b }
func mul[T ~int64 | ~uint64 | ~float32 | ~float64](a, b T) T { return a * b }
func div[T ~int64 | ~uint64 | ~float32 | ~float64](a, b T) T { return a / b }
func mod[T ~int64 | ~uint64](a, b T) T                       { return a % b }
