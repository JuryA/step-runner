package lexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScan(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected []Token
	}{
		"empty": {
			input: "",
		},
		"identifier": {
			input: "hello",
			expected: []Token{
				{Ident, 0, 5, "hello"},
			},
		},
		"reserved identifier": {
			input: "var",
			expected: []Token{
				{Reserved, 0, 3, "var"},
			},
		},
		"identifier starting with ASCII digit": {
			input: "1foo",
			expected: []Token{
				{Number, 0, 1, "1"},
				{Ident, 1, 3, "foo"},
			},
		},
		"identifier starting with UTF-8 digit": {
			input: "๗var",
			expected: []Token{
				{Error, 0, 1, "๗"},
			},
		},
		"ignore whitespace": {
			input: "\t( foo .  \n bar   )",
			expected: []Token{
				{OpenParenthesis, 1, 1, "("},
				{Ident, 3, 3, "foo"},
				{Dot, 7, 1, "."},
				{Ident, 12, 3, "bar"},
				{CloseParenthesis, 18, 1, ")"},
			},
		},
		"unicode offsets": {
			input: "  日本語 '😙🤌'",
			expected: []Token{
				{Ident, 2, 3, "日本語"},
				{String, 6, 4, "'😙🤌'"},
			},
		},
		"string (double quote)": {
			input: `"hello"`,
			expected: []Token{
				{String, 0, 7, `"hello"`},
			},
		},
		"string (double quote, escaped)": {
			input: `"f\oo\"bar\n\\"`,
			expected: []Token{
				{String, 0, 15, `"f\oo\"bar\n\\"`},
			},
		},
		"string (single quote)": {
			input: `'hello'`,
			expected: []Token{
				{String, 0, 7, "'hello'"},
			},
		},
		"string (single quote, escaped)": {
			input: `'f\oo\'bar\n\\'`,
			expected: []Token{
				{String, 0, 15, `'f\oo\'bar\n\\'`},
			},
		},
		"string (with newline)": {
			input: "\"hello\nworld\"",
			expected: []Token{
				{String, 0, 13, "\"hello\nworld\""},
			},
		},
		"invalid string (eof)": {
			input: `"The interrupting sheep wh-`,
			expected: []Token{
				{Error, 0, 27, `"The interrupting sheep wh-`},
			},
		},
		"integer": {
			input: `1234`,
			expected: []Token{
				{Number, 0, 4, "1234"},
			},
		},
		"float": {
			input: `12.34`,
			expected: []Token{
				{Number, 0, 5, "12.34"},
			},
		},
		"float no zero prefix": {
			input: `.5`,
			expected: []Token{
				{Number, 0, 2, ".5"},
			},
		},
		"float accept single dot": {
			input: `12.3.4`,
			expected: []Token{
				{Number, 0, 4, "12.3"},
				{Number, 4, 2, ".4"},
			},
		},
		"float scientific notation: 1e2": {
			input: `1e2`,
			expected: []Token{
				{Number, 0, 3, "1e2"},
			},
		},
		"float scientific notation: 1e+2": {
			input: `1e+2`,
			expected: []Token{
				{Number, 0, 4, "1e+2"},
			},
		},
		"float scientific notation: 1e-2": {
			input: `1e-2`,
			expected: []Token{
				{Number, 0, 4, "1e-2"},
			},
		},
		"float scientific notation: 1E-2": {
			input: `1E-2`,
			expected: []Token{
				{Number, 0, 4, "1E-2"},
			},
		},
		"float scientific notation: 1.5e5": {
			input: `1.5e5`,
			expected: []Token{
				{Number, 0, 5, "1.5e5"},
			},
		},
		"float scientific notation: 1E-2.5 (invalid)": {
			input: `1E-2.5`,
			expected: []Token{
				{Number, 0, 4, "1E-2"},
				{Number, 4, 2, ".5"},
			},
		},
		"float scientific notation: 1e1e (invalid)": {
			input: `1e1e`,
			expected: []Token{
				{Number, 0, 3, "1e1"},
				{Ident, 3, 1, "e"},
			},
		},
		"logical operators": {
			input: `a && b || c == c != d || a > b || b < d || x <= x || x >= !y`,
			expected: []Token{
				{Ident, 0, 1, "a"},
				{And, 2, 2, "&&"},
				{Ident, 5, 1, "b"},
				{Or, 7, 2, "||"},
				{Ident, 10, 1, "c"},
				{Equal, 12, 2, "=="},
				{Ident, 15, 1, "c"},
				{NotEqual, 17, 2, "!="},
				{Ident, 20, 1, "d"},
				{Or, 22, 2, "||"},
				{Ident, 25, 1, "a"},
				{GreaterThan, 27, 1, ">"},
				{Ident, 29, 1, "b"},
				{Or, 31, 2, "||"},
				{Ident, 34, 1, "b"},
				{LessThan, 36, 1, "<"},
				{Ident, 38, 1, "d"},
				{Or, 40, 2, "||"},
				{Ident, 43, 1, "x"},
				{LessThanEqual, 45, 2, "<="},
				{Ident, 48, 1, "x"},
				{Or, 50, 2, "||"},
				{Ident, 53, 1, "x"},
				{GreaterThanEqual, 55, 2, ">="},
				{Not, 58, 1, "!"},
				{Ident, 59, 1, "y"},
			},
		},
		"invalid logical operators": {
			input: `a &! b`,
			expected: []Token{
				{Ident, 0, 1, "a"},
				{Error, 2, 2, "&!"},
			},
		},
		"arithmetic operators": {
			input: "a - b * 12.30 / d % 1 + 100",
			expected: []Token{
				{Ident, 0, 1, "a"},
				{Subtract, 2, 1, "-"},
				{Ident, 4, 1, "b"},
				{Multiply, 6, 1, "*"},
				{Number, 8, 5, "12.30"},
				{Divide, 14, 1, "/"},
				{Ident, 16, 1, "d"},
				{Modulo, 18, 1, "%"},
				{Number, 20, 1, "1"},
				{Add, 22, 1, "+"},
				{Number, 24, 3, "100"},
			},
		},
		"punctuation": {
			input: "( z ) * (( x.y )) + a[1] + {a: b, b: 'yes'}",
			expected: []Token{
				{OpenParenthesis, 0, 1, "("},
				{Ident, 2, 1, "z"},
				{CloseParenthesis, 4, 1, ")"},
				{Multiply, 6, 1, "*"},
				{OpenParenthesis, 8, 1, "("},
				{OpenParenthesis, 9, 1, "("},
				{Ident, 11, 1, "x"},
				{Dot, 12, 1, "."},
				{Ident, 13, 1, "y"},
				{CloseParenthesis, 15, 1, ")"},
				{CloseParenthesis, 16, 1, ")"},
				{Add, 18, 1, "+"},
				{Ident, 20, 1, "a"},
				{OpenBracket, 21, 1, "["},
				{Number, 22, 1, "1"},
				{CloseBracket, 23, 1, "]"},
				{Add, 25, 1, "+"},
				{OpenBrace, 27, 1, "{"},
				{Ident, 28, 1, "a"},
				{Colon, 29, 1, ":"},
				{Ident, 31, 1, "b"},
				{Comma, 32, 1, ","},
				{Ident, 34, 1, "b"},
				{Colon, 35, 1, ":"},
				{String, 37, 5, "'yes'"},
				{CloseBrace, 42, 1, "}"},
			},
		},
		"invalid punctuation": {
			input: "( z ) * \\ x",
			expected: []Token{
				{OpenParenthesis, 0, 1, "("},
				{Ident, 2, 1, "z"},
				{CloseParenthesis, 4, 1, ")"},
				{Multiply, 6, 1, "*"},
				{Error, 8, 1, `\`},
			},
		},
		"dot": {
			input: "foo. bar.buzz",
			expected: []Token{
				{Ident, 0, 3, "foo"},
				{Dot, 3, 1, "."},
				{Ident, 5, 3, "bar"},
				{Dot, 8, 1, "."},
				{Ident, 9, 4, "buzz"},
			},
		},
		"known idents": {
			input: "true false null",
			expected: []Token{
				{True, 0, 4, "true"},
				{False, 5, 5, "false"},
				{Null, 11, 4, "null"},
			},
		},
		"template": {
			input: `true "hello ${{ 'world' }}" false`,
			expected: []Token{
				{True, 0, 4, "true"},
				{Template, 5, 7, "\"hello "},
				{OpenExpr, 12, 3, "${{"},
				{String, 16, 7, "'world'"},
				{CloseExpr, 24, 2, "}}"},
				{Template, 26, 1, "\""},
				{False, 28, 5, "false"},
			},
		},
		"template escape": {
			input: `"\${{ 'world' }}"`,
			expected: []Token{
				{String, 0, 17, "\"\\${{ 'world' }}\""},
			},
		},
		"template nested": {
			input: `true "hello ${{ 'wo ${{ {'foo': {'foo': 'bar'}} }}rld' }}" false`,
			expected: []Token{
				{True, 0, 4, "true"},
				{Template, 5, 7, "\"hello "},
				{OpenExpr, 12, 3, "${{"},
				{Template, 16, 4, "'wo "},
				{OpenExpr, 20, 3, "${{"},
				{OpenBrace, 24, 1, "{"},
				{String, 25, 5, "'foo'"},
				{Colon, 30, 1, ":"},
				{OpenBrace, 32, 1, "{"},
				{String, 33, 5, "'foo'"},
				{Colon, 38, 1, ":"},
				{String, 40, 5, "'bar'"},
				{CloseBrace, 45, 1, "}"},
				{CloseBrace, 46, 1, "}"},
				{CloseExpr, 48, 2, "}}"},
				{Template, 50, 4, "rld'"},
				{CloseExpr, 55, 2, "}}"},
				{Template, 57, 1, "\""},
				{False, 59, 5, "false"},
			},
		},
		"template exprs together": {
			input: `"${{ 1 }}${{ 1 }}"`,
			expected: []Token{
				{Template, 0, 1, "\""},
				{OpenExpr, 1, 3, "${{"},
				{Number, 5, 1, "1"},
				{CloseExpr, 7, 2, "}}"},
				{OpenExpr, 9, 3, "${{"},
				{Number, 13, 1, "1"},
				{CloseExpr, 15, 2, "}}"},
				{Template, 17, 1, "\""},
			},
		},
		"template space": {
			input: `"${{ 1 }} ${{ 1 }}"`,
			expected: []Token{
				{Template, 0, 1, "\""},
				{OpenExpr, 1, 3, "${{"},
				{Number, 5, 1, "1"},
				{CloseExpr, 7, 2, "}}"},
				{Template, 9, 1, " "},
				{OpenExpr, 10, 3, "${{"},
				{Number, 14, 1, "1"},
				{CloseExpr, 16, 2, "}}"},
				{Template, 18, 1, "\""},
			},
		},
		"template after template": {
			input: `"${{ 1 }}""${{ 1 }}"`,
			expected: []Token{
				{Template, 0, 1, "\""},
				{OpenExpr, 1, 3, "${{"},
				{Number, 5, 1, "1"},
				{CloseExpr, 7, 2, "}}"},
				{Template, 9, 1, "\""},
				{Template, 10, 1, "\""},
				{OpenExpr, 11, 3, "${{"},
				{Number, 15, 1, "1"},
				{CloseExpr, 17, 2, "}}"},
				{Template, 19, 1, "\""},
			},
		},
		"interpolation open interrupted": {
			input: `true "${ string" false`,
			expected: []Token{
				{True, 0, 4, "true"},
				{String, 5, 11, `"${ string"`},
				{False, 17, 5, "false"},
			},
		},
		"interpolation close interrupted": {
			input: `true "${{ ident }" false`,
			expected: []Token{
				{True, 0, 4, "true"},
				{Template, 5, 1, "\""},
				{OpenExpr, 6, 3, "${{"},
				{Ident, 10, 5, `ident`},
				{Error, 16, 1, "}"},
			},
		},
	}

	scanner := NewScanner("")
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			scanner.Reset(tc.input)
			assert.Equal(t, tc.expected, readAllTokens(t, scanner))
		})
	}
}

func TestReset(t *testing.T) {
	scanner := NewScanner("a")
	require.Equal(t, []Token{{Ident, 0, 1, "a"}}, readAllTokens(t, scanner))

	scanner.Reset("b + b")
	require.Equal(t, []Token{{Ident, 0, 1, "b"}, {Add, 2, 1, "+"}, {Ident, 4, 1, "b"}}, readAllTokens(t, scanner))
}

func TestReadPeek(t *testing.T) {
	scanner := NewScanner("a b")
	assert.Equal(t, Token{Ident, 0, 1, "a"}, scanner.Scan())
	assert.Equal(t, Token{Ident, 2, 1, "b"}, scanner.Scan())
	assert.Equal(t, Token{EOF, 3, 0, ""}, scanner.Scan())
	assert.Equal(t, Token{EOF, 3, 0, ""}, scanner.Scan())

	scanner = NewScanner("a b")
	assert.Equal(t, Token{Ident, 0, 1, "a"}, scanner.Peek())
	assert.Equal(t, Token{Ident, 0, 1, "a"}, scanner.Peek())
	assert.Equal(t, Token{Ident, 0, 1, "a"}, scanner.Scan())

	assert.Equal(t, Token{Ident, 2, 1, "b"}, scanner.Peek())
	assert.Equal(t, Token{Ident, 2, 1, "b"}, scanner.Peek())
	assert.Equal(t, Token{Ident, 2, 1, "b"}, scanner.Scan())

	assert.Equal(t, Token{EOF, 3, 0, ""}, scanner.Peek())
	assert.Equal(t, Token{EOF, 3, 0, ""}, scanner.Peek())
	assert.Equal(t, Token{EOF, 3, 0, ""}, scanner.Scan())
	assert.Equal(t, EOF, scanner.scan())
	assert.Equal(t, EOF, scanner.scan())
	assert.Equal(t, Token{EOF, 3, 0, ""}, scanner.Scan())

	scanner = NewScanner("")
	assert.Equal(t, Token{EOF, 0, 0, ""}, scanner.Scan())
}

func TestTokenString(t *testing.T) {
	scanner := NewScanner("1.2 + abc / 'hello'")

	var tokens []string
	for {
		next := scanner.scan()
		if next == EOF || next == Error {
			break
		}

		tokens = append(tokens, next.String())
	}

	assert.Equal(t, []string{"+", "ident", "/", "string"}, tokens)
}

func readAllTokens(t *testing.T, s *Scanner) []Token {
	var tokens []Token
	for {
		expectedTyp := s.Peek()

		tok := s.Scan()
		if tok.Type != EOF {
			tokens = append(tokens, tok)
		}

		assert.Equal(t, s.Token(), tok)
		assert.Equal(t, expectedTyp, tok)

		if tok.Type == EOF || tok.Type == Error {
			break
		}
	}

	return tokens
}

func BenchmarkLexer(b *testing.B) {
	input := `( a ) * (( b )) / ( "foo" ) * foo.bar - "hello \"world\"" 'foo\\\'bar' + "${{ 0.5 }}" % d`

	b.Run("baseline", func(b *testing.B) {
		src := strings.NewReader("")
		var dst strings.Builder

		b.SetBytes(int64(len(input)))
		for b.Loop() {
			src.Reset(input)
			dst.Reset()
			for {
				next, _, err := src.ReadRune()
				if err != nil {
					break
				}
				dst.WriteRune(next)
			}
			_ = dst.String()
		}
	})

	b.Run("scanner", func(b *testing.B) {
		scanner := NewScanner("")

		b.SetBytes(int64(len(input)))
		for b.Loop() {
			scanner.Reset(input)
			for {
				token := scanner.Scan()
				if token.Type == Error {
					panic(token)
				}
				if token.Type == EOF {
					break
				}
			}
		}
	})
}
