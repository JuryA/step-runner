package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarks(t *testing.T) {
	const (
		sensitiveMark = 1 << iota
		anotherMark
		unknownMark
	)

	v := String("I'm a sensitive value!").WithMarks(sensitiveMark)
	assert.True(t, v.HasMarks(sensitiveMark))

	modified := v.WithMarks(anotherMark)
	assert.True(t, modified.HasMarks(sensitiveMark))
	assert.True(t, modified.HasMarks(anotherMark))
	assert.False(t, modified.HasMarks(unknownMark))

	duped := modified.WithMarks(sensitiveMark)
	assert.True(t, duped.HasMarks(sensitiveMark|anotherMark))
	assert.False(t, duped.HasMarks(unknownMark))

	obj := MustMap(String("a"), MustMap(String("b"), String("c"))).WithMarks(sensitiveMark)
	item, err := obj.Attr("a")
	require.NoError(t, err)
	assert.True(t, item.HasMarks(sensitiveMark))

	arr := NewList(String("a"), String("b")).WithMarks(sensitiveMark)
	item, err = arr.Index(1)
	require.NoError(t, err)
	assert.True(t, item.HasMarks(sensitiveMark))
}

func TestTrue(t *testing.T) {
	equality := []Value{
		String("x"),
		Number(1.0),
		Number(1),
		MustMap(String(""), Null()),
		NewList(Null()),
		Bool(true),
		Func(func(v ...Value) (Value, error) { return null, nil }),
	}

	for idx, equal := range equality {
		assert.True(t, equal.MustIsTrue(), idx)
		assert.False(t, equal.Not().MustIsTrue(), idx)
	}

	inequality := []Value{
		String(""),
		Number(0),
		Number(0),
		Number(-1.0),
		Number(-1),
		MustMap(),
		NewList(),
		Bool(false),
		Null(),
		Func(nil),
	}

	for idx, equal := range inequality {
		assert.False(t, equal.MustIsTrue(), idx)
		assert.True(t, equal.Not().MustIsTrue(), idx)
	}

	require.Panics(t, func() {
		Value{}.MustIsTrue()
	})
}

func TestString(t *testing.T) {
	assert.Equal(t, "foobar", String("foobar").String())
	assert.Equal(t, "<null>", Null().String())
	assert.Equal(t, `{"foo": "bar","bool": true}`, MustMap(String("foo"), String("bar"), String("bool"), Bool(true)).String())
	assert.Equal(t, `["string", true]`, NewList(String("string"), Bool(true)).String())
	assert.Equal(t, "1.4", Number(1.4).String())
	assert.Equal(t, "9000", Number(9000).String())
	assert.Equal(t, "<func>", Func(nil).String())
}

func TestObjectGetAttrIndex(t *testing.T) {
	obj := MustMap(
		String("foo"), String("bar"),
		String("bool"), Bool(true),
	)
	arr := NewList(Bool(true))
	str := String("foobar")

	tests := map[string]func(t *testing.T){
		"attr valid key": func(t *testing.T) {
			v, err := obj.Attr("foo")
			require.NoError(t, err)
			assert.True(t, String("bar").Equal(v).MustIsTrue())
		},
		"attr unknown key": func(t *testing.T) {
			v, err := obj.Attr("unknown")
			assert.ErrorIs(t, err, ErrAttributeNotFound)
			assert.Equal(t, Null(), v)
		},
		"attr invalid kind": func(t *testing.T) {
			v, err := arr.Attr("foo")
			assert.ErrorIs(t, err, ErrInvalidKind)
			assert.Equal(t, Null(), v)
		},
		"get valid key": func(t *testing.T) {
			v, err := obj.Get(String("foo"))
			require.NoError(t, err)
			assert.True(t, v.Equal(String("bar")).MustIsTrue())
		},
		"get unknown key": func(t *testing.T) {
			v, err := obj.Get(String("nope"))
			assert.ErrorIs(t, err, ErrAttributeNotFound)
			assert.Equal(t, Null(), v)
		},
		"get invalid by int": func(t *testing.T) {
			v, err := obj.Get(Number(0))
			assert.ErrorIs(t, err, ErrInvalidKey)
			assert.Equal(t, Null(), v)
		},
		"index valid (array)": func(t *testing.T) {
			v, err := arr.Index(0)
			require.NoError(t, err)
			assert.Equal(t, Bool(true), v)
		},
		"index valid (string)": func(t *testing.T) {
			v, err := str.Index(0)
			require.NoError(t, err)
			assert.Equal(t, String("f"), v)
		},
		"index invalid (number)": func(t *testing.T) {
			v, err := arr.Index(1)
			assert.ErrorIs(t, err, ErrArrayIndexOutOfRange)
			assert.Equal(t, Null(), v)
		},
		"index invalid (string)": func(t *testing.T) {
			v, err := str.Index(10)
			assert.ErrorIs(t, err, ErrArrayIndexOutOfRange)
			assert.Equal(t, Null(), v)
		},
		"index invalid kind": func(t *testing.T) {
			v, err := obj.Index(0)
			assert.ErrorIs(t, err, ErrInvalidKind)
			assert.Equal(t, Null(), v)
		},
		"get valid by int": func(t *testing.T) {
			v, err := arr.Get(Number(0))
			require.NoError(t, err)
			assert.True(t, v.Equal(Bool(true)).MustIsTrue())
		},
		"get invalid by string": func(t *testing.T) {
			v, err := arr.Get(String("foo"))
			assert.ErrorIs(t, err, ErrInvalidKey)
			assert.Equal(t, Null(), v)
		},
		"get out-of-bounds by int": func(t *testing.T) {
			v, err := arr.Get(Number(1))
			assert.ErrorIs(t, err, ErrArrayIndexOutOfRange)
			assert.Equal(t, Null(), v)
		},
		"get valid by float": func(t *testing.T) {
			v, err := arr.Get(Number(0.0))
			require.NoError(t, err)
			assert.True(t, v.Equal(Bool(true)).MustIsTrue())
		},
		"get valid by float (int cast)": func(t *testing.T) {
			v, err := arr.Get(Number(0.9))
			require.NoError(t, err)
			assert.True(t, v.Equal(Bool(true)).MustIsTrue())
		},
		"get invalid kind": func(t *testing.T) {
			v, err := Number(5.5).Get(String("hello"))
			assert.ErrorIs(t, err, ErrInvalidKind)
			assert.True(t, v.Equal(Null()).MustIsTrue())
		},
	}

	for tn, tc := range tests {
		t.Run(tn, tc)
	}
}
