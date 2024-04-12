package ci

import (
	ctx "context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/step"
	"google.golang.org/protobuf/encoding/protojson"
)

var Cmd = &cobra.Command{
	Use:   "ci",
	Short: "Run steps in a CI environment variable STEPS",
	Args:  cobra.ExactArgs(0),
	RunE:  run,
}

const stepsTemplate = `
spec: {}
---
steps:
`

func run(cmd *cobra.Command, args []string) error {
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

	// Step runner should have no concept of "CI_BUILDS_DIR".
	// However entire `ci` command is a workaround hack because
	// steps are not yet plumbed through runner. Once we receive
	// steps from runner over gRPC we will receive "work_dir"
	// explicitly (set to CI_BUILDS_DIR by runner). Then we can
	// delete this whole command.
	workDir := os.Getenv("CI_BUILDS_DIR")
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	globalCtx.WorkDir = workDir
	result, err := execution.Run(ctx.Background(), globalCtx, params, protoStepDef)
	if err != nil {
		return fmt.Errorf("running execution: %w", err)
	}

	bytes, err := protojson.Marshal(result)
	if err != nil {
		return fmt.Errorf("error marshaling step results: %w", err)
	}
	outputFile := "step-results.json"
	err = os.WriteFile(outputFile, bytes, 0640)
	if err != nil {
		return fmt.Errorf("writing step results to %v: %w", outputFile, err)
	}
	fmt.Printf("trace written to %v\n", outputFile)
	return nil
}
