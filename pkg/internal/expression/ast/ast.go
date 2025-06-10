package ast

import (
	"fmt"
	"slices"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

// Expr represents a node in the AST.
type Expr interface {
	Offset() int
	Length() int
	String() string
	Walk(Visitor) (Expr, error)
}

// Position stores token position information.
type Position struct {
	Off int
	Len int
}

// Offset returns the offset of the associated token.
func (p Position) Offset() int {
	return p.Off
}

// Length returns the length of the associated token.
func (p Position) Length() int {
	return p.Len
}

// Parentheses represents a parenthesized expression.
type Parentheses struct {
	Position
	Expr Expr
}

func (expr *Parentheses) Walk(v Visitor) (Expr, error) {
	if err := walkChild(&expr.Expr, v); err != nil {
		return nil, err
	}

	return v.Visit(expr)
}

func (expr *Parentheses) String() string {
	return fmt.Sprintf("parentheses(%v)", expr.Expr)
}

// Literal represents a literal valuexpr.
type Literal struct {
	Position
	value.Value
}

func (expr *Literal) Walk(v Visitor) (Expr, error) {
	return v.Visit(expr)
}

func (expr Literal) String() string {
	return fmt.Sprintf("literal(%v)", expr.Value.String())
}

// Ident represents an identifier referencexpr.
type Ident struct {
	Position
	Name string
}

func (expr *Ident) Walk(v Visitor) (Expr, error) {
	return v.Visit(expr)
}

func (expr *Ident) String() string {
	return fmt.Sprintf("ident(%v)", expr.Name)
}

// Template represents a string templatexpr.
type Template struct {
	Position
	Exprs []Expr
}

func (expr *Template) Walk(v Visitor) (Expr, error) {
	if err := walkChildren(expr.Exprs, v); err != nil {
		return nil, err
	}

	return v.Visit(expr)
}

func (expr *Template) String() string {
	var sb strings.Builder

	sb.WriteString("template(")
	for idx, expr := range expr.Exprs {
		if idx != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(expr.String())
	}
	sb.WriteString(")")

	return sb.String()
}

// Unary represents a unary operation.
type Unary struct {
	Position
	Op  Op
	RHS Expr
}

func (expr *Unary) Walk(v Visitor) (Expr, error) {
	if err := walkChild(&expr.RHS, v); err != nil {
		return nil, err
	}
	return v.Visit(expr)
}

func (expr *Unary) String() string {
	return fmt.Sprintf("unary(%v, %v)", string(expr.Op), expr.RHS)
}

// Binary represents a binary operation.
type Binary struct {
	Position
	LHS Expr
	Op  Op
	RHS Expr
}

func (expr *Binary) Walk(v Visitor) (Expr, error) {
	if err := walkChild(&expr.LHS, v); err != nil {
		return nil, err
	}
	if err := walkChild(&expr.RHS, v); err != nil {
		return nil, err
	}

	return v.Visit(expr)
}

func (expr *Binary) String() string {
	return fmt.Sprintf("binary(%v, %v, %v)", expr.LHS, string(expr.Op), expr.RHS)
}

// Selector represents property access (obj.prop).
type Selector struct {
	Position
	From   Expr
	Select Expr
}

func (expr *Selector) Walk(v Visitor) (Expr, error) {
	if err := walkChild(&expr.From, v); err != nil {
		return nil, err
	}

	if err := walkChild(&expr.Select, v); err != nil {
		return nil, err
	}

	return v.Visit(expr)
}

func (expr *Selector) String() string {
	return fmt.Sprintf("selector(%v, %v)", expr.From, expr.Select)
}

// Index represents index access (arr[0]).
type Index struct {
	Position
	From  Expr
	Index Expr
}

func (expr *Index) Walk(v Visitor) (Expr, error) {
	if err := walkChild(&expr.From, v); err != nil {
		return nil, err
	}

	if err := walkChild(&expr.Index, v); err != nil {
		return nil, err
	}

	return v.Visit(expr)
}

func (expr *Index) String() string {
	return fmt.Sprintf("index(%v, %v)", expr.From, expr.Index)
}

// Array represents an array constructor [expr, expr, ...].
type Array struct {
	Position
	Items []Expr
}

func (expr *Array) Walk(v Visitor) (Expr, error) {
	if err := walkChildren(expr.Items, v); err != nil {
		return nil, err
	}

	return v.Visit(expr)
}

func (expr *Array) String() string {
	var sb strings.Builder

	sb.WriteString("array(")
	for idx, val := range slices.All(expr.Items) {
		if idx != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(val.String())
	}
	sb.WriteString(")")

	return sb.String()
}

// Object represents an object constructor {expr: expr, ...}.
type Object struct {
	Position
	Items []struct{ Key, Value Expr }
}

func (expr *Object) Walk(v Visitor) (Expr, error) {
	for i := range expr.Items {
		if err := walkChild(&expr.Items[i].Key, v); err != nil {
			return nil, err
		}

		if err := walkChild(&expr.Items[i].Value, v); err != nil {
			return nil, err
		}
	}

	return v.Visit(expr)
}

func (expr *Object) String() string {
	var sb strings.Builder

	sb.WriteString("object(")
	for idx, val := range slices.All(expr.Items) {
		if idx != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(val.Key.String())
		sb.WriteString(":")
		sb.WriteString(val.Value.String())
	}
	sb.WriteString(")")

	return sb.String()
}

// Call represents a function call.
type Call struct {
	Position
	From      Expr
	Arguments []Expr
}

func (expr *Call) Walk(v Visitor) (Expr, error) {
	if err := walkChild(&expr.From, v); err != nil {
		return nil, err
	}

	if err := walkChildren(expr.Arguments, v); err != nil {
		return nil, err
	}

	return v.Visit(expr)
}

func (expr *Call) String() string {
	return fmt.Sprintf("call(%v, %v)", expr.From, expr.Arguments)
}
