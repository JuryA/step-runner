package ci

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func run() error {
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
	// Inherit process environment
	// TODO: require all steps to set all required environment variables from job context
	for _, e := range os.Environ() {
		fields := strings.Split(e, "=")
		if len(fields) != 2 {
			continue
		}
		globalCtx.Env[fields[0]] = fields[1]
	}
	execution, err := runner.New(defs, globalCtx, def.Steps)
	if err != nil {
		return fmt.Errorf("creating execution: %w", err)
	}
	var results []*proto.StepResult
	fn := func(r *proto.StepResult, log string) {
		results = append(results, r)
		fmt.Println(log)
	}
	err = execution.Run(fn)
	if err != nil {
		return fmt.Errorf("running execution: %w", err)
	}
	bytes, err := json.MarshalIndent(results, "", "  ")
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
