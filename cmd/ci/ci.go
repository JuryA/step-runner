package ci

import (
	ctx "context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/report"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

type Options struct {
	WriteStepResultsFile bool
	Steps                string
	WorkDir              string
	JobVariables         map[string]string
}

func NewCmd() *cobra.Command {
	options := &Options{
		Steps:        os.Getenv("STEPS"),
		WorkDir:      findWorkDir(),
		JobVariables: findJobVariables(),
	}

	cmd := &cobra.Command{
		Use:   "ci",
		Short: "Run steps in a CI environment variable STEPS",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(options)
		},
	}

	defaultWriteStepsFile, _ := strconv.ParseBool(os.Getenv("CI_STEPS_DEBUG"))
	cmd.Flags().BoolVar(&options.WriteStepResultsFile, "write-steps-results", defaultWriteStepsFile, "write step-results.json file, note this file may contain secrets")
	return cmd
}

func run(options *Options) error {
	stepDef, err := wrapStepsInSpecDef(options.Steps)
	if err != nil {
		return fmt.Errorf("reading STEPS %q: %w", options.Steps, err)
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

	params := &runner.Params{}

	// Step runner should have no concept of "CI_BUILDS_DIR".
	// However entire `ci` command is a workaround hack because
	// steps are not yet plumbed through runner. Once we receive
	// steps from runner over gRPC we will receive "work_dir"
	// explicitly (set to CI_BUILDS_DIR by runner). Then we can
	// delete this whole command.
	globalCtx.WorkDir = options.WorkDir

	// Add all CI_, GITLAB_ and DOCKER_ environment variables as a
	// workaround until we get an explicit list in the Run gRPC
	// call.
	globalCtx.Job = options.JobVariables

	step, err := runner.NewParser(globalCtx, defs).Parse(protoStepDef, params, runner.StepDefinedInGitLabJob)

	if err != nil {
		return fmt.Errorf("failed to run steps: %w", err)
	}

	env := globalCtx.NewEnvMergedFrom(params.Env)
	inputs := params.NewInputsWithDefault(protoStepDef.Spec.Spec.Inputs)
	stepsCtx := runner.NewStepsContext(globalCtx, protoStepDef.Dir, inputs, env)

	result, err := step.Run(ctx.Background(), stepsCtx, protoStepDef)

	if options.WriteStepResultsFile {
		reptErr := report.NewStepResultReport().Write(result)

		if reptErr != nil {
			fmt.Println(reptErr)
		}
	}

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

func findWorkDir() string {
	workDir := os.Getenv("CI_BUILDS_DIR")

	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	return workDir
}

func findJobVariables() map[string]string {
	variables := map[string]string{}

	prefixes := []string{"CI_", "GITLAB_", "DOCKER_"}

	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")

		if !ok || !slices.ContainsFunc(prefixes, func(prefix string) bool { return strings.HasPrefix(k, prefix) }) {
			continue
		}

		variables[k] = v
	}

	return variables
}
