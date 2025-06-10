package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/ast"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

func TestParser(t *testing.T) {
	tests := []struct {
		input    string
		expected ast.Expr
	}{
		{
			"123 - 123",
			&ast.Binary{
				Position: ast.Position{Off: 4, Len: 1},
				Op:       ast.Subtract,
				LHS: &ast.Literal{
					Position: ast.Position{Len: 3},
					Value:    value.Number(123),
				},
				RHS: &ast.Literal{
					Position: ast.Position{Off: 6, Len: 3},
					Value:    value.Number(123),
				},
			},
		},
		{
			"123",
			&ast.Literal{
				Position: ast.Position{Len: 3},
				Value:    value.Number(123),
			},
		},
		{
			`"hel\"lo"`,
			&ast.Literal{
				Position: ast.Position{Len: 9},
				Value:    value.String("hel\"lo"),
			},
		},
		{
			`'foobar'`,
			&ast.Literal{
				Position: ast.Position{Len: 8},
				Value:    value.String("foobar"),
			},
		},
		{
			`("foo")`,
			&ast.Parentheses{
				Position: ast.Position{Len: 1},
				Expr: &ast.Literal{
					Position: ast.Position{Off: 1, Len: 5},
					Value:    value.String("foo")},
			},
		},
		{
			`foo`,
			&ast.Ident{
				Position: ast.Position{Off: 0, Len: 3},
				Name:     "foo",
			},
		},
		{
			`foo.bar`,
			&ast.Selector{
				Position: ast.Position{Off: 3, Len: 1},
				From: &ast.Ident{
					Position: ast.Position{Off: 0, Len: 3},
					Name:     "foo",
				},
				Select: &ast.Ident{
					Position: ast.Position{Off: 4, Len: 3},
					Name:     "bar",
				},
			},
		},
		{
			`foo.bar.baa`,
			&ast.Selector{
				Position: ast.Position{Off: 7, Len: 1},
				From: &ast.Selector{
					Position: ast.Position{Off: 3, Len: 1},
					From: &ast.Ident{
						Position: ast.Position{Off: 0, Len: 3},
						Name:     "foo",
					},
					Select: &ast.Ident{
						Position: ast.Position{Off: 4, Len: 3},
						Name:     "bar",
					},
				},
				Select: &ast.Ident{
					Position: ast.Position{Off: 8, Len: 3},
					Name:     "baa",
				},
			},
		},
		{
			`foo.bar.baa()`,
			&ast.Call{
				Position: ast.Position{Off: 11, Len: 1},
				From: &ast.Selector{
					Position: ast.Position{Off: 7, Len: 1},
					From: &ast.Selector{
						Position: ast.Position{Off: 3, Len: 1},
						From: &ast.Ident{
							Position: ast.Position{Off: 0, Len: 3},
							Name:     "foo",
						},
						Select: &ast.Ident{
							Position: ast.Position{Off: 4, Len: 3},
							Name:     "bar",
						},
					},
					Select: &ast.Ident{
						Position: ast.Position{Off: 8, Len: 3},
						Name:     "baa",
					},
				},
			},
		},
		{
			`foo.bar.baa(1)`,
			&ast.Call{
				Position: ast.Position{Off: 11, Len: 1},
				From: &ast.Selector{
					Position: ast.Position{Off: 7, Len: 1},
					From: &ast.Selector{
						Position: ast.Position{Off: 3, Len: 1},
						From: &ast.Ident{
							Position: ast.Position{Off: 0, Len: 3},
							Name:     "foo",
						},
						Select: &ast.Ident{
							Position: ast.Position{Off: 4, Len: 3},
							Name:     "bar",
						},
					},
					Select: &ast.Ident{
						Position: ast.Position{Off: 8, Len: 3},
						Name:     "baa",
					},
				},
				Arguments: []ast.Expr{
					&ast.Literal{
						Position: ast.Position{Off: 12, Len: 1},
						Value:    value.Number(1),
					},
				},
			},
		},
		{
			`foo.bar.baa[0]`,
			&ast.Index{
				Position: ast.Position{Off: 11, Len: 1},
				From: &ast.Selector{
					Position: ast.Position{Off: 7, Len: 1},
					From: &ast.Selector{
						Position: ast.Position{Off: 3, Len: 1},
						From: &ast.Ident{
							Position: ast.Position{Off: 0, Len: 3},
							Name:     "foo",
						},
						Select: &ast.Ident{
							Position: ast.Position{Off: 4, Len: 3},
							Name:     "bar",
						},
					},
					Select: &ast.Ident{
						Position: ast.Position{Off: 8, Len: 3},
						Name:     "baa",
					},
				},
				Index: &ast.Literal{
					Position: ast.Position{Off: 12, Len: 1},
					Value:    value.Number(0),
				},
			},
		},
		{
			`foo.bar.baa[0](1)`,
			&ast.Call{
				Position: ast.Position{Off: 14, Len: 1},
				From: &ast.Index{
					Position: ast.Position{Off: 11, Len: 1},
					From: &ast.Selector{
						Position: ast.Position{Off: 7, Len: 1},
						From: &ast.Selector{
							Position: ast.Position{Off: 3, Len: 1},
							From: &ast.Ident{
								Position: ast.Position{Off: 0, Len: 3},
								Name:     "foo",
							},
							Select: &ast.Ident{
								Position: ast.Position{Off: 4, Len: 3},
								Name:     "bar",
							},
						},
						Select: &ast.Ident{
							Position: ast.Position{Off: 8, Len: 3},
							Name:     "baa",
						},
					},
					Index: &ast.Literal{
						Position: ast.Position{Off: 12, Len: 1},
						Value:    value.Number(0),
					},
				},
				Arguments: []ast.Expr{
					&ast.Literal{
						Position: ast.Position{Off: 15, Len: 1},
						Value:    value.Number(1),
					},
				},
			},
		},
		{
			`[]`,
			&ast.Array{
				Position: ast.Position{Off: 0, Len: 1},
			},
		},
		{
			`[1, ]`,
			&ast.Array{
				Position: ast.Position{Off: 0, Len: 1},
				Items: []ast.Expr{
					&ast.Literal{
						Position: ast.Position{Off: 1, Len: 1},
						Value:    value.Number(1),
					},
				},
			},
		},
		{
			`[1, 2, 3]`,
			&ast.Array{
				Position: ast.Position{Off: 0, Len: 1},
				Items: []ast.Expr{
					&ast.Literal{
						Position: ast.Position{Off: 1, Len: 1},
						Value:    value.Number(1),
					},
					&ast.Literal{
						Position: ast.Position{Off: 4, Len: 1},
						Value:    value.Number(2),
					},
					&ast.Literal{
						Position: ast.Position{Off: 7, Len: 1},
						Value:    value.Number(3),
					},
				},
			},
		},
		{
			`{}`,
			&ast.Object{
				Position: ast.Position{Off: 0, Len: 1},
			},
		},
		{
			`{"a": "b", }`,
			&ast.Object{
				Position: ast.Position{Off: 0, Len: 1},
				Items: []struct{ Key, Value ast.Expr }{
					{
						&ast.Literal{
							Position: ast.Position{Off: 1, Len: 3},
							Value:    value.String("a"),
						},
						&ast.Literal{
							Position: ast.Position{Off: 6, Len: 3},
							Value:    value.String("b"),
						},
					},
				},
			},
		},
		{
			`{"a": "b", "b": 1, "array": [1,2], "true": true, "false": false}`,
			&ast.Object{
				Position: ast.Position{Off: 0, Len: 1},
				Items: []struct{ Key, Value ast.Expr }{
					{
						&ast.Literal{
							Position: ast.Position{Off: 1, Len: 3},
							Value:    value.String("a"),
						},
						&ast.Literal{
							Position: ast.Position{Off: 6, Len: 3},
							Value:    value.String("b"),
						},
					},
					{
						&ast.Literal{
							Position: ast.Position{Off: 11, Len: 3},
							Value:    value.String("b"),
						},
						&ast.Literal{
							Position: ast.Position{Off: 16, Len: 1},
							Value:    value.Number(1),
						},
					},
					{
						&ast.Literal{
							Position: ast.Position{Off: 19, Len: 7},
							Value:    value.String("array"),
						},
						&ast.Array{
							Position: ast.Position{Off: 28, Len: 1},
							Items: []ast.Expr{
								&ast.Literal{
									Position: ast.Position{Off: 29, Len: 1},
									Value:    value.Number(1),
								},
								&ast.Literal{
									Position: ast.Position{Off: 31, Len: 1},
									Value:    value.Number(2),
								},
							},
						},
					},
					{
						&ast.Literal{
							Position: ast.Position{Off: 35, Len: 6},
							Value:    value.String("true"),
						},
						&ast.Literal{
							Position: ast.Position{Off: 43, Len: 4},
							Value:    value.Bool(true),
						},
					},
					{
						&ast.Literal{
							Position: ast.Position{Off: 49, Len: 7},
							Value:    value.String("false"),
						},
						&ast.Literal{
							Position: ast.Position{Off: 58, Len: 5},
							Value:    value.Bool(false),
						},
					},
				},
			},
		},
		{
			`({'hello': [1 + 10]})['hello']`,
			&ast.Index{
				Position: ast.Position{Off: 21, Len: 1},
				From: &ast.Parentheses{
					Position: ast.Position{Off: 0, Len: 1},
					Expr: &ast.Object{
						Position: ast.Position{Off: 1, Len: 1},
						Items: []struct{ Key, Value ast.Expr }{
							{
								&ast.Literal{
									Position: ast.Position{Off: 2, Len: 7},
									Value:    value.String("hello"),
								},
								&ast.Array{
									Position: ast.Position{Off: 11, Len: 1},
									Items: []ast.Expr{
										&ast.Binary{
											Position: ast.Position{Off: 14, Len: 1},
											Op:       ast.Add,
											LHS: &ast.Literal{
												Position: ast.Position{Off: 12, Len: 1},
												Value:    value.Number(1),
											},
											RHS: &ast.Literal{
												Position: ast.Position{Off: 16, Len: 2},
												Value:    value.Number(10),
											},
										},
									},
								},
							},
						},
					},
				},
				Index: &ast.Literal{
					Position: ast.Position{Off: 22, Len: 7},
					Value:    value.String("hello"),
				},
			},
		},
		{
			`({'hello': [1 + 10]}).hello`,
			&ast.Selector{
				Position: ast.Position{Off: 21, Len: 1},
				From: &ast.Parentheses{
					Position: ast.Position{Off: 0, Len: 1},
					Expr: &ast.Object{
						Position: ast.Position{Off: 1, Len: 1},
						Items: []struct{ Key, Value ast.Expr }{
							{
								&ast.Literal{
									Position: ast.Position{Off: 2, Len: 7},
									Value:    value.String("hello"),
								},
								&ast.Array{
									Position: ast.Position{Off: 11, Len: 1},
									Items: []ast.Expr{
										&ast.Binary{
											Position: ast.Position{Off: 14, Len: 1},
											Op:       ast.Add,
											LHS: &ast.Literal{
												Position: ast.Position{Off: 12, Len: 1},
												Value:    value.Number(1),
											},
											RHS: &ast.Literal{
												Position: ast.Position{Off: 16, Len: 2},
												Value:    value.Number(10),
											},
										},
									},
								},
							},
						},
					},
				},
				Select: &ast.Ident{
					Position: ast.Position{Off: 22, Len: 5},
					Name:     "hello",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			p := New(tc.input)
			expr, err := p.Parse()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, expr)
		})
	}
}

func FuzzParser(f *testing.F) {
	seed := []string{
		"123", "123 - 123", `"hello"`, `'world'`, "foo", "foo.bar", "foo.bar.baz", "foo()",
		"foo(1, 2, 3)", "foo[0]", "[]", "[1, 2, 3]", "{}", `{"key": "value"}`, `{"a": 1, "b": [1, 2], "c": true}`,
		"(foo + bar)", "foo.bar[0](1, 2)", `({'hello': [1 + 10]})['hello']`, "true", "false",
		"null", "1 + 2 * 3", "foo && bar || baz", "!true", "a == b", "x < y", "x >= y",
	}

	for _, tc := range seed {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, input string) {
		p := New(input)
		_, _ = p.Parse()
	})
}
