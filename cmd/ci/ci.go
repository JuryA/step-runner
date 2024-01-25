package ci

import (
	ctx "context"
	"encoding/json"
	"fmt"
	"os"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/step"
)

const stepsTemplate = `
spec: {}
---
steps:
`

type CI struct{}

func (ci *CI) Run() error {
	steps := os.Getenv("STEPS")
	stepDef, err := step.ReadSteps(stepsTemplate+steps, "")
	if err != nil {
		return fmt.Errorf("reading STEPS %q: %w", steps, err)
	}
	protoStepDef, err := step.CompileSteps(stepDef)
	if err != nil {
		return fmt.Errorf("compiling STEPS: %w", err)
	}

	defs, err := cache.New()
	if err != nil {
		return fmt.Errorf("creating cache: %w", err)
	}
	globalCtx := context.NewGlobal()
	globalCtx.InheritEnv(os.Environ()...)

	execution, err := runner.New(defs)
	if err != nil {
		return fmt.Errorf("creating execution: %w", err)
	}

	params := &runner.Params{}

	result, err := execution.Run(ctx.Background(), protoStepDef, params, globalCtx)
	if err != nil {
		return fmt.Errorf("running execution: %w", err)
	}

	bytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling step results: %w", err)
	}
	outputFile := "step-results.json"
	err = os.WriteFile(outputFile, bytes, 0640)
	if err != nil {
		return fmt.Errorf("writing step results to %v: %w", outputFile, err)
	}
	fmt.Printf("trace written to %v", outputFile)
	return nil
}
