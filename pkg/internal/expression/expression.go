package expression

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
)

func Evaluate(stepsCtx *context.Steps, s string) (*structpb.Value, error) {
	s = strings.TrimSpace(s)
	matches := stepsCtx.GetMatches()
	match, ok := matches[s]

	if !ok {
		return nil, fmt.Errorf("%q cannot be evaluated", s)
	}
	return match, nil
}
