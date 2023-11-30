package runner

import (
	"fmt"
	"os/exec"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/output"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Execution struct {
	defs      *cache.Definitions
	globalCtx *context.Global
	stepsCtx  *context.Steps
	steps     []*proto.Step
}

func New(defs *cache.Definitions, globalCtx *context.Global, steps []*proto.Step) (*Execution, error) {
	return &Execution{
		defs:      defs,
		globalCtx: globalCtx,
		stepsCtx:  context.NewSteps(),
		steps:     steps,
	}, nil
}

type Return func(*proto.StepResult, string)

func (e *Execution) Run(fn Return) error {
	return e.run(e.stepsCtx, e.steps, fn)
}

func (e *Execution) run(stepsCtx *context.Steps, steps []*proto.Step, trace Return) error {
	for _, s := range steps {
		err := expression.InterpolateInputs(e.globalCtx, e.stepsCtx, s)
		if err != nil {
			return fmt.Errorf("interpolating step %q: %w", s.Name, err)
		}
		spec, def, dir, err := e.defs.Get(s.Step)
		if err != nil {
			return fmt.Errorf("getting step %q definition: %w", s.Name, err)
		}
		switch def.Type {
		case proto.DefinitionType_exec:
			err = expression.InterpolateExec(e.globalCtx, s.Inputs, spec.Spec, def.Exec)
			if err != nil {
				return fmt.Errorf("interpolating definition of step %q: %w", s.Name, err)
			}
			files, err := output.New(s)
			if err != nil {
				return err
			}
			var (
				result *proto.StepResult
				log    string
			)
			err = func() error {
				defer files.Cleanup(result)
				result, log, err = e.runExec(s, spec, def, dir)
				if err != nil {
					return fmt.Errorf("running step %q: %w", s.Name, err)
				}
				err = files.OutputTo(stepsCtx, result)
				if err != nil {
					return fmt.Errorf("outputting: %w", err)
				}
				err = files.ExportTo(e.globalCtx, result)
				if err != nil {
					return fmt.Errorf("exporting: %w", err)
				}
				return nil
			}()
			if err != nil {
				return err
			}
			trace(result, log)
		case proto.DefinitionType_steps:
			result, log, err := e.runSteps(s, spec, def)
			if err != nil {
				return fmt.Errorf("running step %q: %w", s.Name, err)
			}
			trace(result, string(log))
		default:
			return fmt.Errorf("unsupported type: %v", def.Type)
		}
	}
	return nil
}

func (e *Execution) runExec(s *proto.Step, spec *proto.Spec, def *proto.Definition, dir string) (*proto.StepResult, string, error) {
	cmd := exec.Command(def.Exec.Command[0], def.Exec.Command[1:]...)
	cmd.Dir = dir
	// Only explicitly provided environment variables
	cmd.Env = []string{}
	for k, v := range s.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	for k, v := range e.globalCtx.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	log, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("exec: %w: %v", err, string(log))
	}
	exitCode := cmd.ProcessState.ExitCode()
	status := proto.StepResult_failure
	if exitCode == 0 {
		status = proto.StepResult_success
	}
	return &proto.StepResult{
		Step:     s,
		Spec:     spec,
		Def:      def,
		Status:   status,
		ExitCode: int32(exitCode),
	}, string(log), nil
}

func (e *Execution) runSteps(s *proto.Step, spec *proto.Spec, def *proto.Definition) (*proto.StepResult, string, error) {
	var log strings.Builder
	result := &proto.StepResult{
		Step: s,
		Spec: spec,
		Def:  def,
	}
	fn := func(child *proto.StepResult, childLog string) {
		result.ChildrenStepResults = append(result.ChildrenStepResults, child)
		log.WriteString(childLog)
	}
	stepsCtx := context.NewSteps()
	err := e.run(stepsCtx, def.Steps, fn)
	if err != nil {
		return nil, "", fmt.Errorf("steps: %w", err)
	}
	result.Outputs, err = expression.InterpolateOutputs(e.globalCtx, stepsCtx, spec.Spec, def)
	if err != nil {
		return nil, "", fmt.Errorf("interpolating output: %w", err)
	}
	e.stepsCtx.Outputs[s.Name] = result.Outputs
	return result, log.String(), nil
}
