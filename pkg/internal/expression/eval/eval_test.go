package eval

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression/parser"
	. "gitlab.com/gitlab-org/step-runner/pkg/internal/expression/value"
)

func TestEval(t *testing.T) {
	const (
		sensitiveMark = 1 << iota
	)

	ctx := &Context{
		Env: MustMap(
			// funcs
			String("now"), Func(func(v ...Value) (Value, error) {
				if len(v) > 0 {
					return Null(), fmt.Errorf("now() don't take no args from anybody")
				}

				return String(time.Now().Format(time.RFC3339)), nil
			}),
			String("contains"), Func(func(v ...Value) (Value, error) {
				if len(v) != 2 {
					return Null(), fmt.Errorf("contains takes 2 arguments: (s string, substr string)")
				}

				contains := Bool(strings.Contains(v[0].String(), v[1].String()))
				contains = contains.WithMarks(v[0].Marks(false) | v[1].Marks(false))

				return contains, nil
			}),
			String("sensitive"), Func(func(v ...Value) (Value, error) {
				if len(v) != 1 {
					return Null(), fmt.Errorf("sensitive takes 1 argument")
				}

				return v[0].WithMarks(sensitiveMark), nil
			}),

			// Data
			String("job"), MustMap(
				String("foo"), MustMap(
					String("outputs"), NewList(
						MustMap(String("hello"), String("world")),
						MustMap(String("foo"), String("bar")),
						MustMap(String("success"), Bool(true)),
						MustMap(String("percent"), Number(99.99)),
						MustMap(String("strawberry?"), Null()),
					),
				),
			),
		),
	}

	input := `[ "Hello ${{ job.foo.outputs[0].hello }}, your password is ${{ sensitive("password") }}", job.foo.outputs[3].percent ]`

	p := parser.New(input)
	tree, err := p.Parse()
	require.NoError(t, err)

	val, err := Eval(ctx, tree)
	require.NoError(t, err)

	idx0, err := val.Index(0)
	require.NoError(t, err)

	idx1, err := val.Index(1)
	require.NoError(t, err)

	require.True(t, idx0.HasMarks(sensitiveMark))
	require.Equal(t, "Hello world, your password is password", idx0.String())

	require.False(t, idx1.HasMarks(sensitiveMark))
	require.True(t, Number(99.99).Equal(idx1).MustIsTrue())
}
