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
	StepDir    string                       // The path to the YAML definition directory so steps can find their files and sub-steps with relative references know where to start.
	OutputFile *StepFile                    // The path to the output file.
	ExportFile *StepFile                    // The path to the export file.
	Env        *Environment                 // Expanded environment values of the executing step.
	Inputs     map[string]*structpb.Value   // Expanded input values of the executing step.
	Steps      map[string]*proto.StepResult // Results of previously executed steps.
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
		StepDir:    stepDir,
		Env:        env,
		Inputs:     inputs,
		Steps:      map[string]*proto.StepResult{},
		OutputFile: outputFile,
		ExportFile: exportFile,
	}

	for _, option := range options {
		option(stepsCtx)
	}

	precond.MustNotBeNil(stepsCtx.Env, "steps context must have an environment")
	return stepsCtx, nil
}

func (s *StepsContext) GetEnvs() map[string]string {
	return s.Env.Values()
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

	s.Env = s.Env.AddLexicalScope(expandedEnv)
	return s.Env, nil
}

func (s *StepsContext) View() *expression.InterpolationContext {
	stepResultViews := make(map[string]*expression.StepResultView)

	for name, step := range s.Steps {
		stepResultViews[name] = &expression.StepResultView{Outputs: step.Outputs}
	}

	return &expression.InterpolationContext{
		Env:         s.Env.Values(),
		ExportFile:  s.ExportFile.Path(),
		Inputs:      s.Inputs,
		Job:         s.globalCtx.Job,
		OutputFile:  s.OutputFile.Path(),
		StepDir:     s.StepDir,
		StepResults: stepResultViews,
		WorkDir:     s.globalCtx.WorkDir,
	}
}

// RecordResult captures the result of a step even if it failed
func (s *StepsContext) RecordResult(stepResult *proto.StepResult) {
	if stepResult == nil || stepResult.Step == nil {
		return
	}

	s.Steps[stepResult.Step.Name] = stepResult
}

func (s *StepsContext) StepResults() []*proto.StepResult {
	return maps.Values(s.Steps)
}

func (s *StepsContext) Cleanup() {
	_ = s.OutputFile.Remove()
	_ = s.ExportFile.Remove()
}

func (s *StepsContext) AddGlobalEnv(env *Environment) {
	s.globalCtx.Env.Mutate(env)
}

func (s *StepsContext) Logln(format string, v ...any) error {
	return s.globalCtx.Logln(format, v...)
}

func (s *StepsContext) WorkDir() string {
	return s.globalCtx.WorkDir
}

func (s *StepsContext) Pipe() (io.Writer, io.Writer) {
	return s.globalCtx.Stdout, s.globalCtx.Stderr
}

func WithStepsCtxOutputFile(outputFile *StepFile) func(*StepsContext) {
	return func(s *StepsContext) {
		s.OutputFile = outputFile
	}
}

func WithStepsCtxExportFile(exportFile *StepFile) func(*StepsContext) {
	return func(s *StepsContext) {
		s.ExportFile = exportFile
	}
}

func WithStepsCtxStepResults(results map[string]*proto.StepResult) func(*StepsContext) {
	return func(s *StepsContext) {
		s.Steps = results
	}
}
