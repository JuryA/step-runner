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
	"gopkg.in/yaml.v3"
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
type: steps
steps:
`

func run(cmd *cobra.Command, args []string) error {
	steps := os.Getenv("STEPS")
	var err error
	steps, err = smashIntoYaml(steps)
	if err != nil {
		return err
	}
	stepDefinition, err := step.Deserialize(stepsTemplate+steps, "")
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

	params := &runner.Params{}

	result, err := execution.Run(ctx.Background(), stepDefinition, params, globalCtx)
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

func smashIntoYaml(yamlOrJson string) (string, error) {
	var a any
	err := yaml.Unmarshal([]byte(yamlOrJson), &a)
	if err != nil {
		return "", fmt.Errorf("smashing into any: %w", err)
	}
	yaml, err := yaml.Marshal(a)
	if err != nil {
		return "", fmt.Errorf("smashing back into yaml: %w", err)
	}
	return string(yaml), nil
}
