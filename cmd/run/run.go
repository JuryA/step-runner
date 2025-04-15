package run

import (
	"bytes"
	ctx "context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"
	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/pkg/di"
	"gitlab.com/gitlab-org/step-runner/pkg/report"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

type Options struct {
	Step              string
	OCIRegistry       string
	OCIRepository     string
	OCITag            string
	OCIDir            string
	OCIFilename       string
	GitURL            string
	GitRev            string
	GitDir            string
	Inputs            map[string]string
	Env               map[string]string
	Job               map[string]string
	TextProtoStepFile string
	WriteStepResults  bool
	StepResultsFile   string
	StepResultsFormat string
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

	cmd.Flags().StringVar(&options.OCIRegistry, "oci-registry", "", "oci registry of step, <host>[:<port>]")
	cmd.Flags().StringVar(&options.OCIRepository, "oci-repository", "", "oci repository of step")
	cmd.Flags().StringVar(&options.OCITag, "oci-tag", "", "oci tag of step")
	cmd.Flags().StringVar(&options.OCIDir, "oci-dir", "", "oci dir of step")
	cmd.Flags().StringVar(&options.OCIFilename, "oci-filename", "", "oci filename of step yaml")
	cmd.Flags().StringVar(&options.GitURL, "git-url", "", "git url of step")
	cmd.Flags().StringVar(&options.GitRev, "git-rev", "", "git revision of step")
	cmd.Flags().StringVar(&options.GitDir, "git-dir", "", "git directory of step")
	cmd.Flags().StringToStringVar(&options.Inputs, "inputs", make(map[string]string), "provide inputs to step")
	cmd.Flags().StringToStringVar(&options.Env, "env", make(map[string]string), "provide environment to step")
	cmd.Flags().StringToStringVar(&options.Job, "job", make(map[string]string), "provide job variables to step")
	cmd.Flags().StringVar(&options.TextProtoStepFile, "text-proto-step-file", "", "file containing a text protobuf definition of a step")

	defaultWriteStepsFile, _ := strconv.ParseBool(os.Getenv("CI_STEPS_DEBUG"))
	cmd.Flags().BoolVar(&options.WriteStepResults, "write-steps-results", defaultWriteStepsFile, "write step-results.json file, note this file may contain secrets")
	cmd.Flags().StringVar(&options.StepResultsFile, "step-results-file", "", "file to write step results")
	cmd.Flags().StringVar(&options.StepResultsFormat, "step-results-format", "json", "format in which to write step results (`json` or `prototext`)")
	return cmd
}

func run(options *Options) error {

	var specDef *runner.SpecDefinition

	if options.TextProtoStepFile != "" {
		data, err := os.ReadFile(options.TextProtoStepFile)
		if err != nil {
			return err
		}

		defn := &proto.Definition{}
		err = prototext.Unmarshal(data, defn)
		if err != nil {
			return err
		}

		spec := &proto.Spec{Spec: &proto.Spec_Content{}}
		dir := filepath.Dir(options.TextProtoStepFile)
		specDef = runner.NewSpecDefinition(spec, defn, dir)
	} else {

		yml, err := yamlStep(options)
		if err != nil {
			return err
		}

		def, err := wrapStepsInSingleStep(yml)
		if err != nil {
			return err
		}

		protoDef, err := def.Compile()
		if err != nil {
			return err
		}

		spec := &proto.Spec{Spec: &proto.Spec_Content{}}
		specDef = runner.NewSpecDefinition(spec, protoDef, "")
	}

	diContainer := di.NewContainer()

	globalCtx, err := createGlobalCtx(options)
	if err != nil {
		return err
	}

	stepParser, err := diContainer.StepParser()
	if err != nil {
		return err
	}

	params := &runner.Params{}
	step, err := stepParser.Parse(globalCtx, specDef, params, runner.StepDefinedInGitLabJob)
	if err != nil {
		return err
	}

	inputs := params.NewInputsWithDefault(specDef.SpecInputs())
	stepsCtx, err := runner.NewStepsContext(globalCtx, specDef.Dir(), inputs, globalCtx.Env())
	if err != nil {
		return err
	}

	defer stepsCtx.Cleanup()

	result, err := step.Run(ctx.Background(), stepsCtx)

	if options.WriteStepResults || options.StepResultsFile != "" {
		reptErr := report.NewStepResultReport(
			options.StepResultsFile,
			report.Format(options.StepResultsFormat),
		).Write(result)
		if reptErr != nil {
			fmt.Println(reptErr)
		}
	}

	return err
}

func createGlobalCtx(options *Options) (*runner.GlobalContext, error) {
	env, err := runner.NewEnvironmentFromOS(excludeJobVars)
	if err != nil {
		return nil, err
	}

	env, err = runner.GlobalEnvironment(env, options.Job)
	if err != nil {
		return nil, err
	}

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}

	globalCtx := runner.NewGlobalContext(workDir, options.Job, env, os.Stdout, os.Stderr)
	return globalCtx, nil
}

func excludeJobVars(envName string) bool {
	return strings.HasPrefix(envName, "CI_") ||
		strings.HasPrefix(envName, "GITLAB_") ||
		strings.HasPrefix(envName, "FF_") ||
		strings.HasPrefix(envName, "DOCKER_ENV_")
}

func wrapStepsInSingleStep(ymlSteps []byte) (*schema.Step, error) {
	def := &schema.Step{}
	err := yaml.Unmarshal(ymlSteps, &def.Run)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal step: %w", err)
	}

	return def, nil
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
	} else if options.OCIRegistry != "" {
		yml.WriteString("- step:\n")
		yml.WriteString("    oci:\n")
		yml.WriteString(fmt.Sprintf("      registry: %s\n", options.OCIRegistry))
		yml.WriteString(fmt.Sprintf("      repository: %s\n", options.OCIRepository))
		yml.WriteString(fmt.Sprintf("      tag: %s\n", options.OCITag))

		if options.OCIDir != "" {
			yml.WriteString(fmt.Sprintf("      dir: %s\n", options.OCIDir))
		}

		if options.OCIFilename != "" {
			yml.WriteString(fmt.Sprintf("      file: %s\n", options.OCIFilename))
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
