package ci

import (
	ctx "context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/pkg/di"
	"gitlab.com/gitlab-org/step-runner/pkg/report"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
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

	argUsage := "write step-results.json file, note this file may contain secrets"
	cmd.Flags().BoolVar(&options.WriteStepResultsFile, "write-steps-results", runner.RunningInDebugMode, argUsage)
	return cmd
}

func run(options *Options) error {
	def, err := wrapStepsInSpecDef(options.Steps)
	if err != nil {
		return fmt.Errorf("reading STEPS %q: %w", options.Steps, err)
	}

	protoDef, err := def.Compile()
	if err != nil {
		return fmt.Errorf("compiling STEPS: %w", err)
	}
	protoStepDef := &proto.SpecDefinition{
		Spec: &proto.Spec{
			Spec: &proto.Spec_Content{},
		},
		Definition: protoDef,
	}

	diContainer := di.NewContainer()

	osEnv, err := runner.NewEnvironmentFromOSWithKnownVars()
	if err != nil {
		return err
	}

	globalEnv, err := runner.GlobalEnvironment(osEnv, options.JobVariables)
	if err != nil {
		return err
	}

	// Step runner should have no concept of "CI_BUILDS_DIR".
	// However entire `ci` command is a workaround hack because
	// steps are not yet plumbed through runner. Once we receive
	// steps from runner over gRPC we will receive "work_dir"
	// explicitly (set to CI_BUILDS_DIR by runner). Then we can
	// delete this whole command.
	// Add all CI_, GITLAB_ and DOCKER_ environment variables as a
	// workaround until we get an explicit list in the Run gRPC
	// call.
	globalCtx := runner.NewGlobalContext(options.WorkDir, options.JobVariables, globalEnv, os.Stdout, os.Stderr)
	params := &runner.Params{}

	stepParser, err := diContainer.StepParser()
	if err != nil {
		return err
	}

	step, err := stepParser.Parse(globalCtx, protoStepDef, params, runner.StepDefinedInGitLabJob)
	if err != nil {
		return fmt.Errorf("failed to run steps: %w", err)
	}

	env := globalCtx.EnvWithLexicalScope(params.Env)
	inputs := params.NewInputsWithDefault(protoStepDef.Spec.Spec.Inputs)
	stepsCtx, err := runner.NewStepsContext(globalCtx, protoStepDef.Dir, inputs, env)

	if err != nil {
		return fmt.Errorf("creating steps context: %w", err)
	}

	defer stepsCtx.Cleanup()

	result, err := step.Run(ctx.Background(), stepsCtx)

	if options.WriteStepResultsFile {
		reptErr := report.NewStepResultReport("", report.FormatJSON).Write(result)

		if reptErr != nil {
			fmt.Println(reptErr)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to run steps: %w", err)
	}

	return nil
}

func wrapStepsInSpecDef(steps string) (*schema.Step, error) {
	def := &schema.Step{}
	err := yaml.Unmarshal([]byte(steps), &def.Run)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling steps: %w", err)
	}
	return def, nil
}

func findWorkDir() string {
	workDir := os.Getenv("CI_PROJECT_DIR")

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
