package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type TokenType int

func (tt TokenType) String() string {
	return tokenNames[tt]
}

const (
	Error TokenType = iota
	EOF
	Reserved

	// types & identifiers
	Ident
	Number
	String
	Null
	True
	False

	// nested expressions
	OpenExpr
	CloseExpr
	Template

	// logical operators
	Equal
	NotEqual
	And
	Or
	LessThanEqual
	LessThan
	GreaterThanEqual
	GreaterThan
	Not

	// arithmetic operators
	Add
	Subtract
	Multiply
	Divide
	Modulo

	// punctuation
	OpenParenthesis
	CloseParenthesis
	OpenBracket
	CloseBracket
	OpenBrace
	CloseBrace
	Colon
	Comma
	Dot
)

const (
	openExprTag  = "${{"
	closeExprTag = "}}"
)

var (
	tokenNames = []string{
		Error:            "error",
		EOF:              "eof",
		Number:           "number",
		String:           "string",
		Ident:            "ident",
		Null:             "null",
		True:             "true",
		False:            "false",
		Reserved:         "reserved",
		OpenExpr:         openExprTag,
		CloseExpr:        closeExprTag,
		Template:         "template",
		Equal:            "==",
		NotEqual:         "!=",
		And:              "&&",
		Or:               "||",
		LessThanEqual:    "<=",
		LessThan:         "<",
		GreaterThanEqual: ">=",
		GreaterThan:      ">",
		Not:              "!",
		Add:              "+",
		Subtract:         "-",
		Multiply:         "*",
		Divide:           "/",
		Modulo:           "%",
		OpenParenthesis:  "(",
		CloseParenthesis: ")",
		OpenBracket:      "[",
		CloseBracket:     "]",
		OpenBrace:        "{",
		CloseBrace:       "}",
		Colon:            ":",
		Comma:            ",",
		Dot:              ".",
	}

	reserved = map[string]struct{}{
		"array": {}, "as": {}, "break": {}, "case": {}, "const": {},
		"continue": {}, "default": {}, "else": {}, "fallthrough": {},
		"float": {}, "for": {}, "func": {}, "function": {}, "goto": {},
		"if": {}, "import": {}, "in": {}, "int": {}, "let": {}, "loop": {},
		"map": {}, "namespace": {}, "number": {}, "object": {}, "package": {},
		"range": {}, "return": {}, "string": {}, "struct": {}, "switch": {},
		"type": {}, "var": {}, "void": {}, "while": {},
	}
)

type Scanner struct {
	r *strings.Reader

	idx int
	buf strings.Builder

	current Token
	next    Token

	// handle expressions in template strings
	expr       int    // track open expressions
	expectExpr bool   // track when Scan() should return an OpenExpr
	brace      []int  // track open braces
	template   []rune // track template terminations
}

type Token struct {
	Type   TokenType
	Offset int
	Length int
	Lexeme string
}

// NewScanner returns a new scanner.
func NewScanner(expr string) *Scanner {
	s := &Scanner{r: strings.NewReader(expr), brace: make([]int, 1, 10)}
	s.Scan()

	return s
}

// Resets resets the scanner.
func (s *Scanner) Reset(expr string) {
	s.buf.Reset()
	s.r.Reset(expr)
	s.idx = 0
	s.current = Token{}
	s.next = Token{}

	s.expr = 0
	s.expectExpr = false
	s.brace = s.brace[:1]
	clear(s.brace)
	s.template = s.template[:0]

	s.Scan()
}

// Scan returns the next token.
func (s *Scanner) Scan() Token {
	s.current = s.next
	s.next.Type = s.scan()
	s.next.Lexeme = s.buf.String()
	s.next.Length = utf8.RuneCountInString(s.next.Lexeme)
	s.next.Offset = max(0, s.idx-s.next.Length)

	return s.current
}

// Peek peeks the next token.
func (s *Scanner) Peek() Token {
	return s.next
}

// Token returns the current token.
func (s *Scanner) Token() Token {
	return s.current
}

func (s *Scanner) scan() TokenType {
	s.buf.Reset()

	if s.expectExpr {
		return s.handleOpenExpr()
	}

	if s.current.Type == CloseExpr {
		return s.scanString()
	}

	// ignore whitespace
	for unicode.IsSpace(s.read()) {
	}
	s.unread()

	r := s.peek()
	switch r {
	case -1:
		return EOF
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
		return s.scanNumber()
	case '\'', '"':
		return s.scanString()
	case '=':
		return s.scanDigraphOp('=', Equal)
	case '!':
		return s.scanMonoOrDigraphOp(Not, '=', NotEqual)
	case '|':
		return s.scanDigraphOp('|', Or)
	case '&':
		return s.scanDigraphOp('&', And)
	case '<':
		return s.scanMonoOrDigraphOp(LessThan, '=', LessThanEqual)
	case '>':
		return s.scanMonoOrDigraphOp(GreaterThan, '=', GreaterThanEqual)
	case '+':
		return s.scanMonographOp(Add)
	case '-':
		return s.scanMonographOp(Subtract)
	case '*':
		return s.scanMonographOp(Multiply)
	case '/':
		return s.scanMonographOp(Divide)
	case '%':
		return s.scanMonographOp(Modulo)
	case '(':
		return s.scanMonographOp(OpenParenthesis)
	case ')':
		return s.scanMonographOp(CloseParenthesis)
	case '[':
		return s.scanMonographOp(OpenBracket)
	case ']':
		return s.scanMonographOp(CloseBracket)
	case '{':
		return s.scanBrace(OpenBrace)
	case '}':
		return s.scanBrace(CloseBrace)
	case ':':
		return s.scanMonographOp(Colon)
	case ',':
		return s.scanMonographOp(Comma)
	default:
		if r == '_' || unicode.IsLetter(r) {
			return s.scanIdent()
		}
		s.buf.WriteRune(s.read())
		return Error
	}
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	s.idx++
	if err != nil {
		return -1
	}

	return ch
}

func (s *Scanner) unread() {
	_ = s.r.UnreadRune()
	s.idx--
}

func (s *Scanner) peek() rune {
	r := s.read()
	s.unread()

	return r
}

func isIdent(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isNumber(r rune) bool {
	return '0' <= r && r <= '9'
}

func (s *Scanner) handleOpenExpr() TokenType {
	s.expectExpr = false

	s.expr++
	s.brace = append(s.brace, 0)
	s.idx += len(openExprTag)
	s.buf.WriteString(openExprTag)

	return OpenExpr
}

func (s *Scanner) handleCloseExpr() TokenType {
	s.expr--
	s.brace, _ = pop(s.brace)
	s.buf.WriteRune(s.read())

	return CloseExpr
}

func (s *Scanner) scanDigraphOp(next rune, typ TokenType) TokenType {
	s.buf.WriteRune(s.read())
	if s.peek() == next {
		s.buf.WriteRune(s.read())
		return typ
	}
	s.buf.WriteRune(s.read())
	return Error
}

func (s *Scanner) scanMonoOrDigraphOp(monoTyp TokenType, next rune, diTyp TokenType) TokenType {
	s.buf.WriteRune(s.read())
	if s.peek() == next {
		s.buf.WriteRune(s.read())
		return diTyp
	}
	return monoTyp
}

func (s *Scanner) scanMonographOp(typ TokenType) TokenType {
	s.buf.WriteRune(s.read())
	return typ
}

func (s *Scanner) scanBrace(typ TokenType) TokenType {
	s.buf.WriteRune(s.read())

	switch typ {
	case OpenBrace:
		s.brace[s.expr]++
		return OpenBrace

	case CloseBrace:
		if s.brace[s.expr] == 0 && s.expr > 0 && s.peek() == '}' {
			return s.handleCloseExpr()
		}

		s.brace[s.expr]--
		if s.brace[s.expr] < 0 {
			break
		}
		return CloseBrace
	}

	return Error
}

func (s *Scanner) scanNumber() TokenType {
	dot := false
	e := false

	for {
		r := s.peek()

		switch {
		case r == '.' && !e:
			if dot {
				break
			}
			s.buf.WriteRune(s.read())
			dot = true
			continue

		case (r == 'E' || r == 'e') && !e:
			e = true
			s.buf.WriteRune(s.read())
			switch s.peek() {
			case '-', '+':
				s.buf.WriteRune(s.read())
			}
			continue

		case isNumber(r):
			s.buf.WriteRune(s.read())
			continue
		}

		break
	}

	if s.buf.Len() == 1 && dot {
		return Dot
	}

	return Number
}

func (s *Scanner) scanString() TokenType {
	var terminating rune

	// a string must start with " or ', and be closed with the same, on
	// CloseExpr, we pop the terminator from string that started the template
	if s.current.Type == CloseExpr {
		s.template, terminating = pop(s.template)
	} else {
		terminating = s.read()
		s.buf.WriteRune(terminating)
	}

	var idx int
	for {
		r := s.read()
		switch {
		case r == -1:
			s.unread()
			return Error

		case r == terminating:
			s.buf.WriteRune(terminating)
			if s.current.Type == CloseExpr {
				return Template
			}
			return String

		case r == '\\':
			p := s.peek()

			if p == terminating || p == '\\' || p == '$' {
				s.buf.WriteRune(r)
				s.buf.WriteRune(s.read())
				continue
			}

		case byte(r) == openExprTag[idx]:
			idx++
			if idx == len(openExprTag) {
				s.idx -= len(openExprTag)
				s.template = append(s.template, terminating)
				s.expectExpr = true

				// if template string is empty, move to OpenExpr
				if s.buf.Len() == 0 {
					return s.handleOpenExpr()
				}

				return Template
			}
			continue

		case idx > 0:
			s.buf.WriteString(openExprTag[0:idx])
			idx = 0
		}

		s.buf.WriteRune(r)
	}
}

func (s *Scanner) scanIdent() TokenType {
	for isIdent(s.peek()) {
		s.buf.WriteRune(s.read())
	}

	lexeme := s.buf.String()
	switch lexeme {
	case "null":
		return Null
	case "true":
		return True
	case "false":
		return False
	}

	if _, ok := reserved[lexeme]; ok {
		return Reserved
	}

	return Ident
}

func pop[Slice []E, E any](slice []E) ([]E, E) {
	item := slice[len(slice)-1]

	return slice[:len(slice)-1], item
}
