package expression

import (
	"bytes"
	"errors"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestEvaluate(t *testing.T) {
	cases := []struct {
		value   string
		want    *structpb.Value
		wantErr error
	}{{
		value: "job.job_id",
		want:  structpb.NewStringValue("1982"),
	}, {
		value: "  job.job_id  ",
		want:  structpb.NewStringValue("1982"),
	}, {
		value:   "job.undefined_key",
		wantErr: errors.New(`job.undefined_key: the "undefined_key" was not found`),
	}}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			got, _, err := Evaluate(textContextSteps(), c.value)
			if c.wantErr != nil {
				require.Equal(t, c.wantErr, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, c.want, got)
			}
		})
	}
}

func TestEvaluateReturnsValueSensitivity(t *testing.T) {
	stepResult := b.ProtoStepResult().
		WithName("secret_factory").
		WithOutputSpec("secret", &proto.Spec_Content_Output{Type: proto.ValueType_string, Sensitive: true}).
		WithOutput("secret", structpb.NewStringValue("secret.value")).
		Build()
	stepContext := b.StepContext().WithStepResult(stepResult).Build()

	value, isSensitive, err := Evaluate(stepContext, "steps.secret_factory.outputs.secret")
	require.NoError(t, err)
	require.True(t, isSensitive)
	require.Equal(t, structpb.NewStringValue("secret.value"), value)
}

type Builders struct{}

var b = &Builders{}

type StepContextBuilder struct {
	stepResults map[string]*proto.StepResult
}

func (b *Builders) StepContext() *StepContextBuilder {
	return &StepContextBuilder{
		stepResults: map[string]*proto.StepResult{},
	}
}

func (bldr *StepContextBuilder) WithStepResult(stepResult *proto.StepResult) *StepContextBuilder {
	bldr.stepResults[stepResult.Step.Name] = stepResult
	return bldr
}

func (bldr *StepContextBuilder) Build() *context.Steps {
	return &context.Steps{
		Global:     b.GlobalContext().Build(),
		StepDir:    ".",
		OutputFile: "output",
		Env:        map[string]string{},
		Inputs:     map[string]*structpb.Value{},
		Steps:      bldr.stepResults,
	}
}

type GlobalContextBuilder struct {
}

func (b *Builders) GlobalContext() *GlobalContextBuilder {
	return &GlobalContextBuilder{}
}

func (bldr *GlobalContextBuilder) Build() *context.Global {
	return &context.Global{
		WorkDir:    ".",
		Job:        map[string]string{},
		ExportFile: "export",
		Env:        map[string]string{},
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
	}
}

type ProtoStepResultBuilder struct {
	name       string
	outputSpec map[string]*proto.Spec_Content_Output
	output     map[string]*structpb.Value
}

func (b *Builders) ProtoStepResult() *ProtoStepResultBuilder {
	return &ProtoStepResultBuilder{
		name:       "my-step",
		outputSpec: map[string]*proto.Spec_Content_Output{},
		output:     map[string]*structpb.Value{},
	}
}

func (bldr *ProtoStepResultBuilder) WithName(name string) *ProtoStepResultBuilder {
	bldr.name = name
	return bldr
}

func (bldr *ProtoStepResultBuilder) WithOutputSpec(name string, spec *proto.Spec_Content_Output) *ProtoStepResultBuilder {
	bldr.outputSpec[name] = spec
	return bldr
}

func (bldr *ProtoStepResultBuilder) WithOutput(name string, value *structpb.Value) *ProtoStepResultBuilder {
	bldr.output[name] = value
	return bldr
}

func (bldr *ProtoStepResultBuilder) Build() *proto.StepResult {
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
