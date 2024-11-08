package runner

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepsContext struct {
	*GlobalContext

	StepDir    string                       // The path to the YAML definition directory so steps can find their files and sub-steps with relative references know where to start.
	OutputFile *StepFile                    // The path to the output file.
	ExportFile *StepFile                    // The path to the export file.
	Env        *Environment                 // Expanded environment values of the executing step.
	Inputs     map[string]*structpb.Value   // Expanded input values of the executing step.
	Steps      map[string]*proto.StepResult // Results of previously executed steps.
}

func NewStepsContext(globalCtx *GlobalContext, dir string, inputs map[string]*structpb.Value, env *Environment, steps map[string]*proto.StepResult) (*StepsContext, error) {
	outputFile, err := NewStepFileInTmp()

	if err != nil {
		return nil, fmt.Errorf("creating steps context: output file: %w", err)
	}

	exportFile, err := NewStepFileInTmp()

	if err != nil {
		return nil, fmt.Errorf("creating steps context: export file: %w", err)
	}

	return &StepsContext{
		GlobalContext: globalCtx,
		StepDir:       dir,
		Env:           env,
		Inputs:        inputs,
		Steps:         steps,
		OutputFile:    outputFile,
		ExportFile:    exportFile,
	}, nil
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

func (s *StepsContext) ExpandAndApplyEnv(env map[string]string) error {
	expandedEnv := map[string]string{}

	for key, value := range env {
		expanded, err := expression.ExpandString(s.View(), value)

		if err != nil {
			return fmt.Errorf("failed to expand environment variable %q: %w", key, err)
		}

		expandedEnv[key] = expanded
	}

	s.Env = s.Env.AddLexicalScope(expandedEnv)
	return nil
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
		Job:         s.Job,
		OutputFile:  s.OutputFile.Path(),
		StepDir:     s.StepDir,
		StepResults: stepResultViews,
		WorkDir:     s.WorkDir,
	}
}

func (s *StepsContext) Cleanup() {
	_ = s.OutputFile.Remove()
	_ = s.ExportFile.Remove()
}
