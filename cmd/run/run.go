package run

import (
	"bytes"
	ctx "context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/report"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

type Options struct {
	Step                 string
	GitURL               string
	GitRev               string
	GitDir               string
	Inputs               map[string]string
	Env                  map[string]string
	Job                  map[string]string
	WriteStepResultsFile bool
}

func NewCmd() *cobra.Command {
	options := &Options{}

	cmd := &cobra.Command{
		Use:   "run [local or remote step, step starting with 'step: [location]', or omit if using git flags]",
		Short: "Run a step locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				options.Step = args[0]
			}

			return run(options)
		},
	}

	cmd.Flags().StringVar(&options.GitURL, "git-url", "", "git url of step")
	cmd.Flags().StringVar(&options.GitRev, "git-rev", "", "git revision of step")
	cmd.Flags().StringVar(&options.GitDir, "git-dir", "", "git directory of step")
	cmd.Flags().StringToStringVar(&options.Inputs, "inputs", make(map[string]string), "provide inputs to step")
	cmd.Flags().StringToStringVar(&options.Env, "env", make(map[string]string), "provide environment to step")
	cmd.Flags().StringToStringVar(&options.Job, "job", make(map[string]string), "provide job variables to step")

	defaultWriteStepsFile, _ := strconv.ParseBool(os.Getenv("CI_STEPS_DEBUG"))
	cmd.Flags().BoolVar(&options.WriteStepResultsFile, "write-steps-results", defaultWriteStepsFile, "write step-results.json file, note this file may contain secrets")
	return cmd
}

func run(options *Options) error {
	yml, err := yamlStep(options)
	if err != nil {
		return err
	}

	stepDef, err := wrapStepsInSpecDef(yml)
	if err != nil {
		return err
	}

	specDef, err := schema.CompileSteps(stepDef)
	if err != nil {
		return err
	}

	stepCache, err := cache.New()
	if err != nil {
		return err
	}

	globalCtx, err := createGlobalCtx(options)
	if err != nil {
		return err
	}

	defer globalCtx.Cleanup()

	step, err := runner.NewParser(globalCtx, stepCache).Parse(specDef, &runner.Params{}, runner.StepDefinedInGitLabJob)
	if err != nil {
		return err
	}

	stepsCtx := runner.NewStepsContext(globalCtx, "", map[string]*structpb.Value{}, map[string]string{})
	result, err := step.Run(ctx.Background(), stepsCtx, specDef)

	if options.WriteStepResultsFile {
		reptErr := report.NewStepResultReport().Write(result)
		if reptErr != nil {
			fmt.Println(reptErr)
		}
	}

	return err
}

func createGlobalCtx(options *Options) (*runner.GlobalContext, error) {
	globalCtx, err := runner.NewGlobalContext()

	if err != nil {
		return nil, err
	}

	workDir, err := os.Getwd()

	if err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}

	globalCtx.WorkDir = workDir
	globalCtx.Job = options.Job
	return globalCtx, nil
}

func wrapStepsInSpecDef(ymlSteps []byte) (*schema.StepDefinition, error) {
	specDef := &schema.StepDefinition{Spec: &schema.Spec{}, Definition: &schema.Definition{}}
	err := yaml.Unmarshal(ymlSteps, &specDef.Definition.Steps)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal step: %w", err)
	}

	return specDef, nil
}

func yamlStep(options *Options) ([]byte, error) {
	yml := bytes.NewBufferString("")

	if strings.HasPrefix(options.Step, "step:") {
		yml.WriteString(fmt.Sprintf("- %s\n", options.Step))
	} else if options.Step != "" {
		yml.WriteString(fmt.Sprintf("- step: %s\n", options.Step))
	} else if options.GitURL != "" {
		yml.WriteString("- step:\n")
		yml.WriteString("    git:\n")
		yml.WriteString(fmt.Sprintf("      url: %s\n", options.GitURL))

		if options.GitRev != "" {
			yml.WriteString(fmt.Sprintf("      rev: %s\n", options.GitRev))
		}

		if options.GitDir != "" {
			yml.WriteString(fmt.Sprintf("      dir: %s\n", options.GitDir))
		}
	} else {
		return nil, fmt.Errorf("no step specified")
	}

	yml.WriteString(yamlObject("inputs", options.Inputs))
	yml.WriteString(yamlObject("env", options.Env))
	return yml.Bytes(), nil
}

func yamlObject(name string, values map[string]string) string {
	if len(values) == 0 {
		return ""
	}

	yml := bytes.NewBufferString(fmt.Sprintf("  %s:\n", name))

	for name, value := range values {
		yml.WriteString(fmt.Sprintf("    %s: %s\n", name, value))
	}

	return yml.String()
}
