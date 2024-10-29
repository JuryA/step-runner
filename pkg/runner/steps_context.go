package runner

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepsContext struct {
	*GlobalContext

	StepDir     string                       // The path to the YAML definition directory so steps can find their files and sub-steps with relative references know where to start.
	OutputFile  string                       // The path to the output file.
	ContextFile string                       // The path to the context file.
	Env         *Environment                 // Expanded environment values of the executing step.
	Inputs      map[string]*structpb.Value   // Expanded input values of the executing step.
	Steps       map[string]*proto.StepResult // Results of previously executed steps.
}

func NewStepsContext(globalCtx *GlobalContext, dir string, inputs map[string]*structpb.Value, env map[string]string) *StepsContext {
	return &StepsContext{
		GlobalContext: globalCtx,
		StepDir:       dir,
		Env:           NewEnvironment(env),
		Inputs:        inputs,
		Steps:         map[string]*proto.StepResult{},
	}
}

func (s *StepsContext) GetEnvs() map[string]string {
	r := make(map[string]string)
	for k, v := range s.GlobalContext.Env.Values() {
		r[k] = v
	}
	for k, v := range s.Env.Values() {
		r[k] = v
	}
	return r
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
		ExportFile:  s.ExportFile,
		Inputs:      s.Inputs,
		Job:         s.Job,
		OutputFile:  s.OutputFile,
		ContextFile: s.ContextFile,
		StepDir:     s.StepDir,
		StepResults: stepResultViews,
		WorkDir:     s.WorkDir,
	}
}

func (s *StepsContext) Proto() *proto.Context {
	job := []*proto.Variable{}
	for k, v := range s.Job {
		job = append(job, &proto.Variable{
			Key:   k,
			Value: v,
			// We've lost sensitive and file metadata!
		})
	}
	return &proto.Context{
		Env:     s.GetEnvs(),
		Job:     job,
		Steps:   s.Steps,
		Inputs:  s.Inputs,
		WorkDir: s.WorkDir,
		StepDir: s.StepDir,
	}
}
