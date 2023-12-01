package ci

import (
	ctx "context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/step"
	"gitlab.com/gitlab-org/step-runner/proto"
)

var Cmd = &cobra.Command{
	Use:   "ci",
	Short: "Run steps in a CI environment variable STEPS",
	Args:  cobra.ExactArgs(0),
	RunE: run,
		return run()
	},
}

func run(cmd *cobra.Command, args []string) error {
	steps := os.Getenv("STEPS")
	def, err := step.ReadSteps(steps)
	if err != nil {
		return fmt.Errorf("reading STEPS %q: %w", steps, err)
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

	specDefinition := &proto.StepDefinition{
		Spec: &proto.Spec{
			Spec: &proto.Spec_Content{},
		},
		Definition: def,
	}
	stepCall := &proto.StepCall{}

	result, err := execution.Run(ctx.Background(), specDefinition, stepCall, globalCtx)
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
