package ci

import (
	ctx "context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/di"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

var Cmd = &cobra.Command{
	Use:   "ci",
	Short: "Run steps in a CI environment variable STEPS",
	Args:  cobra.ExactArgs(0),
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {
	container, err := di.Initialize()
	defer container.CleanUp()

	if err != nil {
		return fmt.Errorf("failed to run steps: %w", err)
	}

	steps, err := convertToYAML(os.Getenv("STEPS"))

	if err != nil {
		return fmt.Errorf("failed to run steps: %w", err)
	}

	fmt.Printf("running steps:\n%s\n", steps)

	_, protoStepDef, err := container.StepParser.Parse(steps, "")

	if err != nil {
		return fmt.Errorf("failed to run steps: %w", err)
	}

	defs, err := cache.New()
	if err != nil {
		return fmt.Errorf("creating cache: %w", err)
	}

	execution, err := runner.New(defs)
	if err != nil {
		return fmt.Errorf("creating execution: %w", err)
	}

	params := &runner.Params{}

	result, err := execution.Run(ctx.Background(), container.GlobalCtx, params, protoStepDef)

	writeResults := func() error {
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

	if err != nil {
		_ = writeResults()
		fmt.Printf("unable to write results: %v", err)
		return fmt.Errorf("running execution: %w", err)
	}

	return writeResults()
}

func convertToYAML(envSteps string) (string, error) {
	return fmt.Sprintf("spec:\n---\nsteps:\n%s", strings.TrimSpace(envSteps)), nil

	//specDef := &schema.StepDefinition{
	//	Spec:       &schema.Spec{},
	//	Definition: &schema.Definition{},
	//}
	//
	//err := yaml.Unmarshal([]byte(envSteps), &specDef.Definition.Steps)
	//
	//if err != nil {
	//	return "", fmt.Errorf("failed to convert CI/CD steps to YAML: %w", err)
	//}
	//
	//runningSteps, err := yaml.Marshal(specDef)
	//
	//if err != nil {
	//	return "", fmt.Errorf("failed to convert CI/CD steps to YAML: %w", err)
	//}
	//
	//return string(runningSteps), nil
}
