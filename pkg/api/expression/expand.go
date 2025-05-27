package expression

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	internalExpr "gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
)

type StepsJobInput struct {
	Key       string
	Value     *structpb.Value
	Sensitive bool
}

func Expand(jobInputs []*StepsJobInput, expr *structpb.Value) (*structpb.Value, error) {
	// Having separate Input types for Steps and Runner is likely sensible
	inputs := map[string]*structpb.Value{}

	for _, jobInput := range jobInputs {
		inputs[jobInput.Key] = jobInput.Value
	}

	// Need to create a constrained interpolation context instead of this mess
	interpolationCtx := &internalExpr.InterpolationContext{
		Env:         map[string]string{},
		ExportFile:  "/dev/null",
		Inputs:      inputs,
		Job:         map[string]string{},
		OutputFile:  "/dev/null",
		StepDir:     "/dev/null",
		StepResults: map[string]*internalExpr.StepResultView{},
		WorkDir:     "/dev/null",
	}

	value, err := internalExpr.Expand(interpolationCtx, expr)
	if err != nil {
		return nil, fmt.Errorf("failed to expand expression '%s': %w", expr.String(), err)
	}

	return value.Value, err
}
