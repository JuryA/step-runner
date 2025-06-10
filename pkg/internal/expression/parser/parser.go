package parser

import (
	"errors"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/ast"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/lexer"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

type Parser struct {
	s *lexer.Scanner

	err error
}

var (
	ErrUnexpectedToken = errors.New("unexpected token")
)

type ParseError struct {
	Offset   int
	Length   int
	Type     lexer.TokenType
	Err      error
	Expected []lexer.TokenType
}

func (e ParseError) Error() string {
	switch {
	case e.Err != nil:
		return fmt.Sprintf("parser: %v at %d:%d", e.Err.Error(), e.Offset, e.Length)
	case e.Expected == nil:
		return fmt.Sprintf("parser: unexpected %v at %d:%d", e.Type.String(), e.Offset, e.Length)
	default:
		var sb strings.Builder
		for idx, expected := range e.Expected {
			if idx != 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(expected.String())
		}

		return fmt.Sprintf("parser: expected %v, got %v at %d:%d", sb.String(), e.Type, e.Offset, e.Length)
	}
}

func New(input string) *Parser {
	return &Parser{s: lexer.NewScanner(input)}
}

func (p *Parser) Parse() (ast.Expr, error) {
	expr := p.expression(1)
	p.match(lexer.EOF)

	return expr, p.err
}

func (p *Parser) expression(precedence int) ast.Expr {
	expr := p.unary()

	for {
		if p.err != nil {
			break
		}

		op := p.binaryOperator()
		if op.Precedence() < precedence {
			break
		}

		p.match(p.peek())

		pos := p.pos()
		rhs := p.expression(op.Precedence() + 1)
		expr = &ast.Binary{Position: pos, Op: op, LHS: expr, RHS: rhs}
	}

	return expr
}

func (p *Parser) unary() ast.Expr {
	switch p.peek() {
	case lexer.Subtract:
		p.match(lexer.Subtract)
		return &ast.Unary{Position: p.pos(), Op: ast.Subtract, RHS: p.unary()}

	case lexer.Add:
		p.match(lexer.Add)
		return &ast.Unary{Position: p.pos(), Op: ast.Add, RHS: p.unary()}

	case lexer.Not:
		p.match(lexer.Not)
		return &ast.Unary{Position: p.pos(), Op: ast.Not, RHS: p.unary()}

	case lexer.Null:
		p.match(lexer.Null)
		return &ast.Literal{Position: p.pos(), Value: value.Null()}

	case lexer.True:
		p.match(lexer.True)
		return &ast.Literal{Position: p.pos(), Value: value.Bool(true)}

	case lexer.False:
		p.match(lexer.False)
		return &ast.Literal{Position: p.pos(), Value: value.Bool(false)}

	case lexer.String:
		raw := p.match(lexer.String)
		str, err := value.ParseString(raw)
		if err != nil {
			return p.error(p.s.Token(), fmt.Errorf("parsing string: %w", err))
		}

		return p.postfix(&ast.Literal{Position: p.pos(), Value: str})

	case lexer.Number:
		raw := p.match(lexer.Number)
		num, err := value.ParseNumber(raw)
		if err != nil {
			return p.error(p.s.Token(), fmt.Errorf("parsing number: %w", err))
		}

		return p.postfix(&ast.Literal{Position: p.pos(), Value: num})

	case lexer.OpenBracket:
		return p.postfix(p.array())

	case lexer.OpenBrace:
		return p.postfix(p.object())

	case lexer.OpenParenthesis:
		return p.postfix(p.paren())

	case lexer.Ident:
		return p.postfix(&ast.Ident{Position: p.nextPos(), Name: p.match(lexer.Ident)})

	case lexer.Template:
		return p.template()

	default:
		if strings.HasPrefix(p.s.Peek().Lexeme, "$") {
			// hack: lexer.OpenExpr isn't scanned outside of string literals,
			// so we cannot recieve it above, so we check here instead.
			return p.error(p.s.Peek(), fmt.Errorf("template expression can only appear in strings"))
		}
		return p.error(p.s.Peek(), fmt.Errorf("expecting unary expression"))
	}
}

func (p *Parser) template() ast.Expr {
	type item struct {
		token lexer.Token
		expr  []ast.Expr
	}

	var builder []item
	for p.is(lexer.Template) {
		builder = append(builder, item{token: p.s.Scan()})

		if !p.is(lexer.OpenExpr) {
			break
		}

		for p.is(lexer.OpenExpr) {
			p.match(lexer.OpenExpr)
			if !p.is(lexer.CloseExpr) {
				builder[len(builder)-1].expr = append(builder[len(builder)-1].expr, p.expression(1))
			}
			p.match(lexer.CloseExpr)
		}
	}

	template := &ast.Template{}
	var terminating string
	for idx, item := range builder {
		pos := ast.Position{Off: item.token.Offset, Len: item.token.Length}

		var str string
		switch idx {
		case 0:
			terminating = string(item.token.Lexeme[0])
			template.Position = pos
			str = item.token.Lexeme + terminating
		case len(builder) - 1:
			str = terminating + item.token.Lexeme
		default:
			str = terminating + item.token.Lexeme + terminating
		}

		val, err := value.ParseString(str)
		if err != nil {
			return p.error(item.token, fmt.Errorf("parsing string: %w", err))
		}

		template.Exprs = append(template.Exprs, &ast.Literal{Position: pos, Value: val})
		template.Exprs = append(template.Exprs, item.expr...)
	}

	return template
}

// postfix: .ident, .ident["attr"], ["attr"], [123] with (args...) for calls
func (p *Parser) postfix(expr ast.Expr) ast.Expr {
	for {
		switch p.peek() {
		case lexer.Dot:
			p.match(lexer.Dot)

			switch p.peek() {
			case lexer.Ident:
				expr = &ast.Selector{Position: p.pos(), From: expr, Select: &ast.Ident{
					Position: p.nextPos(), Name: p.match(lexer.Ident),
				}}

			default:
				return p.error(p.s.Peek(), nil, lexer.Ident)
			}

		case lexer.OpenBracket:
			p.match(lexer.OpenBracket)
			selector := &ast.Index{Position: p.pos(), From: expr}

			selector.Index = p.expression(1)
			p.match(lexer.CloseBracket)

			expr = selector

		case lexer.OpenParenthesis:
			expr = &ast.Call{Position: p.nextPos(), From: expr, Arguments: p.call()}

		default:
			return expr
		}
	}
}

// binaryOperator: ||, &&, ==, !=, <, <=, >, >=, +, -, *, /
func (p *Parser) binaryOperator() ast.Op {
	switch p.peek() {
	case lexer.Or:
		return ast.Or
	case lexer.And:
		return ast.And
	case lexer.Equal:
		return ast.Equal
	case lexer.NotEqual:
		return ast.NotEqual
	case lexer.LessThan:
		return ast.LessThan
	case lexer.LessThanEqual:
		return ast.LessThanEqual
	case lexer.GreaterThan:
		return ast.GreaterThan
	case lexer.GreaterThanEqual:
		return ast.GreaterThanEqual
	case lexer.Add:
		return ast.Add
	case lexer.Subtract:
		return ast.Subtract
	case lexer.Multiply:
		return ast.Multiply
	case lexer.Divide:
		return ast.Divide
	}

	return ast.Nop
}

// paren: ( expr )
func (p *Parser) paren() *ast.Parentheses {
	p.match(lexer.OpenParenthesis)
	expr := &ast.Parentheses{Position: p.pos(), Expr: p.expression(1)}
	p.match(lexer.CloseParenthesis)

	return expr
}

// object: { "a": "b", ... }
func (p *Parser) object() ast.Expr {
	p.match(lexer.OpenBrace)

	obj := &ast.Object{Position: p.pos()}

	for p.not(lexer.CloseBrace) {
		key := p.expression(1)
		p.match(lexer.Colon)
		val := p.expression(1)

		obj.Items = append(obj.Items, struct{ Key, Value ast.Expr }{Key: key, Value: val})
		if p.not(lexer.Comma) {
			break
		}
		p.match(lexer.Comma)

		// allow trailing comma
		if p.is(lexer.CloseBrace) {
			break
		}
	}

	p.match(lexer.CloseBrace)

	return obj
}

// array: [ "a", "b", ... ]
func (p *Parser) array() ast.Expr {
	p.match(lexer.OpenBracket)

	arr := &ast.Array{Position: p.pos()}

	for p.not(lexer.CloseBracket) {
		arr.Items = append(arr.Items, p.expression(1))

		if p.not(lexer.Comma) {
			break
		}
		p.match(lexer.Comma)

		// allow trailing comma
		if p.is(lexer.CloseBracket) {
			break
		}
	}

	p.match(lexer.CloseBracket)

	return arr
}

// call: ( 1, 2, "c", ... )
func (p *Parser) call() []ast.Expr {
	if p.not(lexer.OpenParenthesis) {
		return nil
	}

	p.match(lexer.OpenParenthesis)

	var arguments []ast.Expr
	for p.not(lexer.CloseParenthesis) {
		arguments = append(arguments, p.expression(1))

		if p.not(lexer.Comma) {
			break
		}
		p.match(lexer.Comma)
	}

	p.match(lexer.CloseParenthesis)

	return arguments
}

func (p *Parser) pos() ast.Position {
	return ast.Position{
		Off: p.s.Token().Offset,
		Len: p.s.Token().Length,
	}
}

func (p *Parser) nextPos() ast.Position {
	return ast.Position{
		Off: p.s.Peek().Offset,
		Len: p.s.Peek().Length,
	}
}

func (p *Parser) match(token lexer.TokenType) string {
	next := p.s.Scan()
	if next.Type != token {
		if token == lexer.EOF {
			p.error(p.s.Token(), nil)
			return next.Lexeme
		}
		p.error(p.s.Token(), nil, token)
	}

	return next.Lexeme
}

func (p *Parser) peek() lexer.TokenType {
	if p.err != nil {
		return lexer.Error
	}

	return p.s.Peek().Type
}

func (p *Parser) is(token lexer.TokenType) bool {
	return p.peek() == token
}

func (p *Parser) not(token lexer.TokenType) bool {
	return p.peek() != token
}

func (p *Parser) error(token lexer.Token, err error, expected ...lexer.TokenType) ast.Expr {
	if p.err != nil {
		return nil
	}

	p.err = &ParseError{
		Err:      err,
		Offset:   token.Offset,
		Length:   token.Length,
		Type:     token.Type,
		Expected: expected,
	}

	return nil
}
