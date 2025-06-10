package eval

import (
	"errors"
	"fmt"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/ast"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

type Context struct {
	Env value.Value
}

// EvalError wraps evaluation errors with position information
type EvalError struct {
	Expr ast.Expr
	Err  error
}

func (e EvalError) Error() string {
	return fmt.Sprintf("runtime: %v at %d:%d", e.Err.Error(), e.Expr.Offset(), e.Expr.Length())
}

// eval evaluates an AST expression with the given data context
func Eval(ctx *Context, expr ast.Expr) (value.Value, error) {
	return eval(ctx, expr)
}

func eval(ctx *Context, expr ast.Expr) (value.Value, error) {
	switch e := expr.(type) {
	case *ast.Literal:
		return e.Value, nil

	case *ast.Ident:
		return wrapError(ctx.Env.Get(value.String(e.Name)))(e, "identifier: %w")

	case *ast.Parentheses:
		return eval(ctx, e.Expr)

	case *ast.Array:
		return array(ctx, e)

	case *ast.Object:
		return object(ctx, e)

	case *ast.Template:
		return template(ctx, e)

	case *ast.Selector:
		return selector(ctx, e)

	case *ast.Index:
		return index(ctx, e)

	case *ast.Unary:
		return unary(ctx, e)

	case *ast.Binary:
		return binary(ctx, e)

	case *ast.Call:
		return call(ctx, e)

	default:
		return value.Null(), fmt.Errorf("unknown expression type: %T", expr)
	}
}

func wrapError(val value.Value, err error) func(expr ast.Expr, format string) (value.Value, error) {
	if err != nil {
		return func(expr ast.Expr, format string) (value.Value, error) {
			return val, &EvalError{expr, fmt.Errorf(format, err)}
		}
	}

	return func(expr ast.Expr, format string) (value.Value, error) {
		return val, nil
	}
}

func array(ctx *Context, expr *ast.Array) (value.Value, error) {
	items := make(value.List, 0, len(expr.Items))

	for _, itemExpr := range expr.Items {
		item, err := eval(ctx, itemExpr)
		if err != nil {
			return value.Null(), err
		}

		items = append(items, item)
	}

	return value.NewList(items...), nil
}

func object(ctx *Context, expr *ast.Object) (value.Value, error) {
	items := make([]value.Value, 0, len(expr.Items))

	for _, kv := range expr.Items {
		key, err := eval(ctx, kv.Key)
		if err != nil {
			return value.Null(), err
		}

		if key.Kind() != value.StringKind {
			return wrapError(value.Null(), fmt.Errorf("must evaluate to string"))(kv.Key, "%w")
		}

		val, err := eval(ctx, kv.Value)
		if err != nil {
			return value.Null(), err
		}

		items = append(items, key, val)
	}

	obj, err := value.NewMap(items...)
	if err != nil {
		return value.Null(), err
	}

	return obj, nil
}

func template(ctx *Context, expr *ast.Template) (value.Value, error) {
	var concat string
	var marks uint16
	for _, item := range expr.Exprs {
		result, err := eval(ctx, item)
		if err != nil {
			return value.Null(), err
		}

		if result.Kind() != value.StringKind {
			return wrapError(value.Null(), fmt.Errorf("must evaluate to string"))(item, "%w")
		}

		concat += result.String()
		marks |= result.Marks(false)
	}

	return value.String(concat).WithMarks(marks), nil
}

func selector(ctx *Context, expr *ast.Selector) (value.Value, error) {
	from, err := eval(ctx, expr.From)
	if err != nil {
		return value.Null(), err
	}

	sel := ast.Expr(expr.Select)
	if ident, ok := expr.Select.(*ast.Ident); ok {
		sel = &ast.Literal{Position: expr.Position, Value: value.String(ident.Name)}
	}

	selVal, err := eval(ctx, sel)
	if err != nil {
		return value.Null(), err
	}

	return wrapError(from.Get(selVal))(sel, "selector: %w")
}

func index(ctx *Context, expr *ast.Index) (value.Value, error) {
	from, err := eval(ctx, expr.From)
	if err != nil {
		return value.Null(), err
	}

	selVal, err := eval(ctx, expr.Index)
	if err != nil {
		return value.Null(), err
	}

	return wrapError(from.Get(selVal))(expr.Index, "index: %w")
}

func unary(ctx *Context, expr *ast.Unary) (value.Value, error) {
	rhs, err := eval(ctx, expr.RHS)
	if err != nil {
		return value.Null(), err
	}

	return wrapError(UnaryOp(expr.Op, rhs))(expr, "%w")
}

func binary(ctx *Context, expr *ast.Binary) (value.Value, error) {
	lhs, err := eval(ctx, expr.LHS)
	switch {
	// we ignore AttributeNotFound and IndexOutOfRange errors for the Or operation immediately returning the RHS.
	case expr.Op == ast.Or && (errors.Is(err, value.ErrAttributeNotFound) || errors.Is(err, value.ErrArrayIndexOutOfRange)):
		return eval(ctx, expr.RHS)

	// for other operations, or other errors, the error is propogated
	case err != nil:
		return lhs, err
	}

	rhs, err := eval(ctx, expr.RHS)
	if err != nil {
		return rhs, err
	}

	return wrapError(BinaryOp(lhs, expr.Op, rhs))(expr, "%w")
}

func call(ctx *Context, expr *ast.Call) (value.Value, error) {
	val, err := eval(ctx, expr.From)
	if err != nil {
		return value.Null(), err
	}

	args := make(value.List, 0, len(expr.Arguments))
	for _, argExpr := range expr.Arguments {
		val, err := eval(ctx, argExpr)
		if err != nil {
			return value.Null(), err
		}

		args = append(args, val)
	}

	return wrapError(val.Call(args...))(expr, "calling func: %w")
}

func UnaryOp(op ast.Op, rhs value.Value) (value.Value, error) {
	switch op {
	case ast.Add:
		return rhs, nil

	case ast.Subtract:
		return rhs.Negate()

	case ast.Not:
		return rhs.Not(), nil

	default:
		return value.Null(), nil
	}
}

func BinaryOp(lhs value.Value, op ast.Op, rhs value.Value) (value.Value, error) {
	switch op {
	case ast.Or:
		ok, err := lhs.IsTrue()
		if err != nil {
			return value.Null(), err
		}
		if ok {
			return lhs, nil
		}
		return rhs, nil
	case ast.And:
		ok, err := lhs.IsTrue()
		if err != nil {
			return value.Null(), err
		}
		if ok {
			return rhs, nil
		}
		return lhs, nil

	case ast.Equal:
		return lhs.Equal(rhs), nil
	case ast.NotEqual:
		return lhs.Equal(rhs).Negate()

	case ast.LessThan:
		return lhs.LessThan(rhs), nil
	case ast.LessThanEqual:
		return lhs.LessThanEqual(rhs), nil

	case ast.GreaterThan:
		return lhs.GreaterThan(rhs), nil
	case ast.GreaterThanEqual:
		return lhs.GreaterThanEqual(rhs), nil

	case ast.Add:
		return lhs.Add(rhs)
	case ast.Subtract:
		return lhs.Subtract(rhs)

	case ast.Multiply:
		return lhs.Multiply(rhs)
	case ast.Divide:
		return lhs.Divide(rhs)

	default:
		return value.Null(), nil
	}
}
