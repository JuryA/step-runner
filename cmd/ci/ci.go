package ci

import (
	ctx "context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

var Cmd = &cobra.Command{
	Use:   "ci",
	Short: "Run steps in a CI environment variable STEPS",
	Args:  cobra.ExactArgs(0),
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {
	steps := os.Getenv("STEPS")
	stepDef, err := wrapStepsInSpecDef(steps)
	if err != nil {
		return fmt.Errorf("reading STEPS %q: %w", steps, err)
	}
	protoStepDef, err := schema.CompileSteps(stepDef)
	if err != nil {
		return fmt.Errorf("compiling STEPS: %w", err)
	}

	defs, err := cache.New()
	if err != nil {
		return fmt.Errorf("creating cache: %w", err)
	}
	globalCtx, err := runner.NewGlobalContext()
	if err != nil {
		return fmt.Errorf("creating global context: %w", err)
	}
	defer globalCtx.Cleanup()

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

	// Add all CI_, GITLAB_ and DOCKER_ environment variables as a
	// workaround until we get an explicit list in the Run gRPC
	// call.
	globalCtx.Job = map[string]string{}
	prefixes := []string{"CI_", "GITLAB_", "DOCKER_"}
	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")
		if !ok || !slices.ContainsFunc(prefixes, func(prefix string) bool {
			return strings.HasPrefix(k, prefix)
		}) {
			continue
		}
		globalCtx.Job[k] = v
	}

	step, err := schema.NewParser(defs, execution.Run).Parse(protoStepDef)

	if err != nil {
		return fmt.Errorf("failed to run steps: %w", err)
	}

	result, err := execution.Run(ctx.Background(), globalCtx, params, step, protoStepDef)
	writeResultToFile(result)

	if err != nil {
		return fmt.Errorf("failed to run steps: %w", err)
	}

	return nil
}

func wrapStepsInSpecDef(steps string) (*schema.StepDefinition, error) {
	specDef := &schema.StepDefinition{
		Spec:       &schema.Spec{},
		Definition: &schema.Definition{},
	}
	err := yaml.Unmarshal([]byte(steps), &specDef.Definition.Steps)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling steps: %w", err)
	}
	runningSteps, _ := yaml.Marshal(specDef)
	fmt.Printf("running steps:\n%v", string(runningSteps))
	return specDef, nil
}

func writeResultToFile(result *proto.StepResult) {
	bytes, err := protojson.Marshal(result)

	if err != nil {
		fmt.Println(fmt.Errorf("failed to write step results to file: %w", err))
		return
	}

	outputFile := "step-results.json"
	err = os.WriteFile(outputFile, bytes, 0640)

	if err != nil {
		fmt.Println(fmt.Errorf("failed to write step results to file: %w", err))
		return
	}

	fmt.Printf("step results written to %v\n", outputFile)
}
