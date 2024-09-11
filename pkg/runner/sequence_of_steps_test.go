package runner

import (
	"bytes"
	ctx "context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestSequenceOfSteps_Describe(t *testing.T) {
	steps := NewSequenceOfSteps(StepDefinedInGitLabJob, &Params{}, nil, nil)
	require.Equal(t, "sequence of 2 steps", steps.Describe())
}

func TestSequenceOfSteps_Run(t *testing.T) {
	t.Run("sub-step succeeds", func(t *testing.T) {
		specDef := buildSpecDef()
		steps := NewSequenceOfSteps(StepDefinedInGitLabJob, &Params{}, NewFixedResultStep(buildStepResult(specDef, proto.StepResult_success)))

		result, err := steps.Run(context.Background(), buildStepsCtx(), specDef)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, proto.StepResult_success, result.Status)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, proto.StepResult_success, result.SubStepResults[0].Status)
	})

	t.Run("sub-step fails", func(t *testing.T) {
		specDef := buildSpecDef()
		err := fmt.Errorf("simulated.error")
		steps := NewSequenceOfSteps(StepDefinedInGitLabJob, &Params{}, NewFixedResultStepWithErr(buildStepResult(specDef, proto.StepResult_failure), err))

		result, err := steps.Run(context.Background(), buildStepsCtx(), specDef)
		require.Error(t, err)
		require.Equal(t, "failed to run sequence of steps: simulated.error", err.Error())
		require.NotNil(t, result)
		require.Equal(t, proto.StepResult_failure, result.Status)
		require.Len(t, result.SubStepResults, 1)
		require.Equal(t, proto.StepResult_failure, result.SubStepResults[0].Status)
	})
}

func buildStepResult(specDef *proto.SpecDefinition, status proto.StepResult_Status) *proto.StepResult {
	stepResult := &proto.StepResult{
		SpecDefinition: specDef,
		Status:         status,
		Outputs: map[string]*structpb.Value{
			"value": structpb.NewStringValue("cmd.output"),
		},
		Exports:    make(map[string]string),
		ExecResult: &proto.StepResult_ExecResult{},
	}
	return stepResult
}

func buildStepsCtx() *StepsContext {
	stepsCtx := &StepsContext{
		GlobalContext: buildGlobalCtx(),
		StepDir:       ".",
		OutputFile:    "output",
		Env:           map[string]string{},
		Inputs:        map[string]*structpb.Value{},
		Steps:         map[string]*proto.StepResult{},
	}
	return stepsCtx
}

func buildGlobalCtx() *GlobalContext {
	globalCtx := &GlobalContext{
		WorkDir:    ".",
		Job:        map[string]string{},
		ExportFile: "export",
		Env:        map[string]string{},
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
	}
	return globalCtx
}

func buildSpecDef() *proto.SpecDefinition {
	specDef := &proto.SpecDefinition{
		Spec: &proto.Spec{
			Spec: &proto.Spec_Content{
				Inputs: map[string]*proto.Spec_Content_Input{},
				Outputs: map[string]*proto.Spec_Content_Output{
					"value": {
						Type:      proto.ValueType_string,
						Default:   structpb.NewStringValue("default.output"),
						Sensitive: false,
					},
				},
				OutputMethod: proto.OutputMethod_outputs,
			},
		},
		Definition: &proto.Definition{
			Type: proto.DefinitionType_exec,
			Exec: &proto.Definition_Exec{
				Command: []string{"go", "run", "."},
				WorkDir: "",
			},
			Steps:    nil,
			Outputs:  map[string]*structpb.Value{},
			Env:      map[string]string{},
			Delegate: "",
		},
		Dir: "",
	}
	return specDef
}

type FixedResultStep struct {
	stepResult *proto.StepResult
	err        error
}

func NewFixedResultStep(stepResult *proto.StepResult) *FixedResultStep {
	return &FixedResultStep{
		stepResult: stepResult,
		err:        nil,
	}
}

func NewFixedResultStepWithErr(stepResult *proto.StepResult, err error) *FixedResultStep {
	return &FixedResultStep{
		stepResult: stepResult,
		err:        err,
	}
}

func (s *FixedResultStep) Describe() string {
	return fmt.Sprintf("fixed result step %s", s.stepResult.Status)
}

func (s *FixedResultStep) Run(_ ctx.Context, _ *StepsContext, _ *proto.SpecDefinition) (*proto.StepResult, error) {
	return s.stepResult, s.err
}
