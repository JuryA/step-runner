package runner

import (
	"fmt"
	"io"

	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/pkg/precond"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepsContext struct {
	globalCtx  *GlobalContext
	stepDir    string                       // The path to the YAML definition directory so steps can find their files and sub-steps with relative references know where to start.
	outputFile *StepFile                    // The path to the output file.
	exportFile *StepFile                    // The path to the export file.
	env        *Environment                 // Expanded environment values of the executing step.
	inputs     map[string]*structpb.Value   // Expanded input values of the executing step.
	steps      map[string]*proto.StepResult // Results of previously executed steps.
}

func NewStepsContext(globalCtx *GlobalContext, stepDir string, inputs map[string]*structpb.Value, env *Environment, options ...func(*StepsContext)) (*StepsContext, error) {
	outputFile, err := NewStepFileInTmp()

	if err != nil {
		return nil, fmt.Errorf("creating steps context: output file: %w", err)
	}

	exportFile, err := NewStepFileInTmp()

	if err != nil {
		return nil, fmt.Errorf("creating steps context: export file: %w", err)
	}

	stepsCtx := &StepsContext{
		globalCtx:  globalCtx,
		stepDir:    stepDir,
		env:        env,
		inputs:     inputs,
		steps:      map[string]*proto.StepResult{},
		outputFile: outputFile,
		exportFile: exportFile,
	}

	for _, option := range options {
		option(stepsCtx)
	}

	precond.MustNotBeNil(stepsCtx.env, "steps context must have an environment")
	return stepsCtx, nil
}

func (s *StepsContext) GetEnvs() map[string]string {
	return s.env.Values()
}

func (s *StepsContext) GetEnvList() []string {
	r := []string{}
	for k, v := range s.GetEnvs() {
		r = append(r, k+"="+v)
	}
	return r
}

func (s *StepsContext) ExpandAndApplyEnv(env map[string]string) (*Environment, error) {
	expandedEnv := map[string]string{}

	for key, value := range env {
		expanded, err := expression.ExpandString(s.View(), value)

		if err != nil {
			return nil, fmt.Errorf("env variable %q: %w", key, err)
		}

		expandedEnv[key] = expanded
	}

	s.env = s.env.AddLexicalScope(expandedEnv)
	return s.env, nil
}

func (s *StepsContext) View() *expression.InterpolationContext {
	stepResultViews := make(map[string]*expression.StepResultView)

	for name, step := range s.steps {
		stepResultViews[name] = &expression.StepResultView{Outputs: step.Outputs}
	}

	return &expression.InterpolationContext{
		Env:         s.env.Values(),
		ExportFile:  s.exportFile.Path(),
		Inputs:      s.inputs,
		Job:         s.globalCtx.Job(),
		OutputFile:  s.outputFile.Path(),
		StepDir:     s.stepDir,
		StepResults: stepResultViews,
		WorkDir:     s.globalCtx.WorkDir(),
	}
}

// RecordResult captures the result of a step even if it failed
func (s *StepsContext) RecordResult(stepResult *proto.StepResult) {
	if stepResult == nil || stepResult.Step == nil {
		return
	}

	s.steps[stepResult.Step.Name] = stepResult
}

func (s *StepsContext) StepResults() []*proto.StepResult {
	return maps.Values(s.steps)
}

func (s *StepsContext) Cleanup() {
	_ = s.outputFile.Remove()
	_ = s.exportFile.Remove()
}

func (s *StepsContext) AddGlobalEnv(env *Environment) {
	s.globalCtx.AddGlobalEnv(env)
}

func (s *StepsContext) Logln(format string, v ...any) error {
	return s.globalCtx.Logln(format, v...)
}

func (s *StepsContext) WorkDir() string {
	return s.globalCtx.WorkDir()
}

func (s *StepsContext) Pipe() (io.Writer, io.Writer) {
	return s.globalCtx.Pipe()
}

func (s *StepsContext) ReadOutputStepResult() (*proto.StepResult, error) {
	return s.outputFile.ReadStepResult()
}

func (s *StepsContext) ReadOutputValues(specOutputs map[string]*proto.Spec_Content_Output) (map[string]*structpb.Value, error) {
	return s.outputFile.ReadValues(specOutputs)
}

func (s *StepsContext) ReadExportedEnv() (*Environment, error) {
	return s.exportFile.ReadEnvironment()
}

func (s *StepsContext) EnvWithLexicalScope(envVars map[string]string) *Environment {
	return s.env.AddLexicalScope(envVars)
}

func WithStepsCtxOutputFile(outputFile *StepFile) func(*StepsContext) {
	return func(s *StepsContext) {
		s.outputFile = outputFile
	}
}

func WithStepsCtxExportFile(exportFile *StepFile) func(*StepsContext) {
	return func(s *StepsContext) {
		s.exportFile = exportFile
	}
}

func WithStepsCtxStepResults(results map[string]*proto.StepResult) func(*StepsContext) {
	return func(s *StepsContext) {
		s.steps = results
	}
}
