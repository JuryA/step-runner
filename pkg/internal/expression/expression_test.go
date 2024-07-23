package expression

import (
	"bytes"
	"errors"
	"testing"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"

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
			got, err := Evaluate(textContextSteps(), c.value)
			if c.wantErr != nil {
				require.Equal(t, c.wantErr, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, c.want, got.Value)
			}
		})
	}
}

func TestEvaluateSensitivity(t *testing.T) {
	tests := map[string]struct {
		sensitive           bool
		wantSensitiveReason string
	}{
		"sensitive": {
			sensitive:           true,
			wantSensitiveReason: "steps.secret_factory.outputs.secret",
		},
		"not sensitive": {
			sensitive: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stepResult := b.protoStepResult().
				withName("secret_factory").
				withOutputSpec("secret", &proto.Spec_Content_Output{Type: proto.ValueType_string, Sensitive: test.sensitive}).
				withOutput("secret", structpb.NewStringValue("secret.value")).
				build()
			stepContext := b.stepContext().withStepResult(stepResult).build()

			value, err := Evaluate(stepContext, "steps.secret_factory.outputs.secret")
			require.NoError(t, err)
			require.Equal(t, structpb.NewStringValue("secret.value"), value.Value)
			require.Equal(t, test.sensitive, value.Sensitive)

			if test.sensitive {
				require.Equal(t, test.wantSensitiveReason, value.SensitiveReason)
			}
		})
	}
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
