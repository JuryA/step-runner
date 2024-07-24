package context_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestStep_ExpandInputs(t *testing.T) {
	protoStep := &proto.Step{
		Name:   "step.name",
		Step:   &proto.Step_Reference{Filename: "step.yml"},
		Env:    map[string]string{},
		Inputs: map[string]*structpb.Value{},
	}

	protoSpecDef := &proto.SpecDefinition{
		Spec: &proto.Spec{
			Spec: &proto.Spec_Content{
				Inputs:       map[string]*proto.Spec_Content_Input{},
				Outputs:      map[string]*proto.Spec_Content_Output{},
				OutputMethod: proto.OutputMethod_outputs,
			},
		},
	}

	stepResult := b.protoStepResult().
		withName("my_step").
		withOutputSpec("first_name", &proto.Spec_Content_Output{Type: proto.ValueType_string}).
		withOutput("first_name", structpb.NewStringValue("fred")).
		build()
	stepsCtx := b.stepContext().withStepResult(stepResult).build()
	inputs := map[string]*context.Variable{
		"welcome": context.NewVariable(structpb.NewStringValue("welcome, ${{steps.my_step.outputs.first_name}}"), false),
		"name":    context.NewVariable(structpb.NewStringValue("Your name is ${{steps.my_step.outputs.first_name}}."), false),
	}

	expandedInputs, err := context.NewStep(protoStep, protoSpecDef, inputs).ExpandInputs(stepsCtx, expression.Expand)
	require.NoError(t, err)
	require.Len(t, expandedInputs, 2)
	require.Equal(t, "welcome, fred", expandedInputs["welcome"].Value.GetStringValue())
	require.Equal(t, "Your name is fred.", expandedInputs["name"].Value.GetStringValue())
}

type builders struct{}

var b = &builders{}

type stepContextBuilder struct {
	stepResults map[string]*proto.StepResult
}

func (*builders) stepContext() *stepContextBuilder {
	return &stepContextBuilder{
		stepResults: map[string]*proto.StepResult{},
	}
}

func (bldr *stepContextBuilder) withStepResult(stepResult *proto.StepResult) *stepContextBuilder {
	bldr.stepResults[stepResult.Step.Name] = stepResult
	return bldr
}

func (bldr *stepContextBuilder) build() *context.Steps {
	return &context.Steps{
		Global:     b.globalContext().build(),
		StepDir:    ".",
		OutputFile: "output",
		Env:        map[string]string{},
		Inputs:     map[string]*structpb.Value{},
		Steps:      bldr.stepResults,
	}
}

type globalContextBuilder struct {
}

func (*builders) globalContext() *globalContextBuilder {
	return &globalContextBuilder{}
}

func (*globalContextBuilder) build() *context.Global {
	return &context.Global{
		WorkDir:    ".",
		Job:        map[string]string{},
		ExportFile: "export",
		Env:        map[string]string{},
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
	}
}

type protoStepResultBuilder struct {
	name       string
	outputSpec map[string]*proto.Spec_Content_Output
	output     map[string]*structpb.Value
}

func (*builders) protoStepResult() *protoStepResultBuilder {
	return &protoStepResultBuilder{
		name:       "my-step",
		outputSpec: map[string]*proto.Spec_Content_Output{},
		output:     map[string]*structpb.Value{},
	}
}

func (bldr *protoStepResultBuilder) withName(name string) *protoStepResultBuilder {
	bldr.name = name
	return bldr
}

func (bldr *protoStepResultBuilder) withOutputSpec(name string, spec *proto.Spec_Content_Output) *protoStepResultBuilder {
	bldr.outputSpec[name] = spec
	return bldr
}

func (bldr *protoStepResultBuilder) withOutput(name string, value *structpb.Value) *protoStepResultBuilder {
	bldr.output[name] = value
	return bldr
}

func (bldr *protoStepResultBuilder) build() *proto.StepResult {
	return &proto.StepResult{
		Step: &proto.Step{
			Name:   bldr.name,
			Step:   &proto.Step_Reference{Filename: "step.yml"},
			Env:    map[string]string{},
			Inputs: map[string]*structpb.Value{},
		},
		SpecDefinition: &proto.SpecDefinition{
			Spec: &proto.Spec{
				Spec: &proto.Spec_Content{
					Inputs:       map[string]*proto.Spec_Content_Input{},
					Outputs:      bldr.outputSpec,
					OutputMethod: proto.OutputMethod_outputs,
				},
			},
			Definition: &proto.Definition{
				Type:     proto.DefinitionType_exec,
				Exec:     &proto.Definition_Exec{Command: []string{"go", "run", "."}, WorkDir: ""},
				Steps:    nil,
				Outputs:  map[string]*structpb.Value(nil),
				Env:      map[string]string{},
				Delegate: "",
			},
			Dir: "",
		},
		Status:         proto.StepResult_success,
		Outputs:        bldr.output,
		Exports:        map[string]string{},
		Env:            map[string]string{},
		ExecResult:     &proto.StepResult_ExecResult{Command: []string{"go", "run", "."}, WorkDir: "", ExitCode: 0},
		SubStepResults: nil,
	}
}
