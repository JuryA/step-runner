package value

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNumber(t *testing.T) {
	tests := []struct {
		in  string
		out Value
		err error
	}{
		{`1e2`, Number(1e2), nil},
		{`1e-2`, Number(1e-2), nil},
		{`1E-2`, Number(1e-2), nil},
		{`1.5e5`, Number(1.5e5), nil},
		{`5e-324`, Number(5e-324), nil},
		{`18446744073709551615`, Number(uint64(math.MaxUint64)), nil},
		{`-9223372036854775808`, Number(int64(math.MinInt64)), nil},
		{`9223372036854775807`, Number(int64(math.MaxInt64)), nil},
		{`18446744073709551615`, Number(uint64(math.MaxUint64)), nil},
		{`1E-2.5`, null, errInvalidNumber},
		{`1e1e`, null, errInvalidNumber},
	}

	for _, tc := range tests {
		t.Run(tc.in+"->"+tc.out.String(), func(t *testing.T) {
			out, err := ParseNumber(tc.in)
			assert.True(t, tc.out.Equal(out).MustIsTrue())
			assert.ErrorIs(t, err, tc.err)
		})
	}
}

func TestParseString(t *testing.T) {
	tests := []struct {
		in  string
		out Value
		err error
	}{
		{`""`, String(""), nil},
		{`"a"`, String("a"), nil},
		{`"abc"`, String("abc"), nil},
		{`"☺"`, String("☺"), nil},
		{`"hello world"`, String("hello world"), nil},
		{`"\a\b\f\n\r\t\v\\\""`, String("\a\b\f\n\r\t\v\\\""), nil},
		{`"'"`, String("'"), nil},
		{`'a'`, String("a"), nil},
		{`'☹'`, String("☹"), nil},
		{`' '`, String(" "), nil},
		{`'\''`, String("'"), nil},
		{`'"'`, String("\""), nil},
		{"''", String(""), nil},
		{"'abc'", String("abc"), nil},
		{"'hello world'", String("hello world"), nil},
		{"'\\\\'", String("\\"), nil},
		{"'\n'", String("\n"), nil},
		{"'	'", String("	"), nil},
		{"' '", String(" "), nil},
		{"'a\rb'", String("a\rb"), nil},
		{`'foo\$bar'`, String("foo$bar"), nil},
		{`"foo\$bar"`, String("foo$bar"), nil},

		{``, null, errStringNotTerminating},
		{`"`, null, errStringNotTerminating},
		{`"a`, null, errStringNotTerminating},
		{`b"`, null, errStringNotTerminating},
		{`aba`, null, errStringNotTerminating},

		{`"foo\qbar"`, null, errInvalidEscape},
		{`'foo\qbar'`, null, errInvalidEscape},
	}

	for _, tc := range tests {
		t.Run(tc.in+"->"+tc.out.String(), func(t *testing.T) {
			out, err := ParseString(tc.in)
			assert.Equal(t, tc.out, out)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}
