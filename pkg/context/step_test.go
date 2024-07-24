package context_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/context/b"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/expression"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestStep_ExpandInputs(t *testing.T) {
	protoStep := b.ProtoStep().WithName("step.name").Build()
	protoSpecDef := b.ProtoSpecDef().Build()
	stepResult := b.ProtoStepResult().
		WithName("my_step").
		WithOutputSpec("first_name", &proto.Spec_Content_Output{Type: proto.ValueType_string}).
		WithOutput("first_name", structpb.NewStringValue("fred")).
		Build()
	stepsCtx := b.StepContext().WithStepResult(stepResult).Build()
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

func TestStep_ExpandEnv(t *testing.T) {
	protoStep := b.ProtoStep().
		WithName("step.name").
		WithEnvVar("welcome", "welcome, ${{steps.my_step.outputs.first_name}}").
		WithEnvVar("name", "Your name is ${{steps.my_step.outputs.first_name}}.").
		Build()
	protoSpecDef := b.ProtoSpecDef().Build()
	stepResult := b.ProtoStepResult().
		WithName("my_step").
		WithOutputSpec("first_name", &proto.Spec_Content_Output{Type: proto.ValueType_string}).
		WithOutput("first_name", structpb.NewStringValue("fred")).
		Build()
	stepsCtx := b.StepContext().WithStepResult(stepResult).Build()
	inputs := map[string]*context.Variable{
		"welcome": context.NewVariable(structpb.NewStringValue("welcome, ${{steps.my_step.outputs.first_name}}"), false),
		"name":    context.NewVariable(structpb.NewStringValue("Your name is ${{steps.my_step.outputs.first_name}}."), false),
	}

	expandedEnv, err := context.NewStep(protoStep, protoSpecDef, inputs).ExpandEnv(stepsCtx, expression.ExpandString)
	require.NoError(t, err)
	require.Len(t, expandedEnv, 2)
	require.Equal(t, "welcome, fred", expandedEnv["welcome"])
	require.Equal(t, "Your name is fred.", expandedEnv["name"])
}
