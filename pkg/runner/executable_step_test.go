package runner_test

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestExecutableStep_Run(t *testing.T) {
	if _, err := os.Stat("/bin/bash"); errors.Is(err, os.ErrNotExist) {
		t.Skip("skipping test because /bin/bash doesn't exist")
	}

	t.Run("executes command", func(t *testing.T) {
		tests := map[string]struct {
			outputType     proto.ValueType
			outputValue    string
			expected       interface{}
			extractValueFn func(value *structpb.Value) interface{}
		}{
			"string output type": {
				outputType:     proto.ValueType_string,
				outputValue:    `value="hello world"`,
				expected:       "hello world",
				extractValueFn: func(value *structpb.Value) interface{} { return value.GetStringValue() },
			},
			"number output type": {
				outputType:     proto.ValueType_number,
				outputValue:    "value=56.77",
				expected:       56.77,
				extractValueFn: func(value *structpb.Value) interface{} { return value.GetNumberValue() },
			},
			"boolean output type": {
				outputType:     proto.ValueType_boolean,
				outputValue:    "value=true",
				expected:       true,
				extractValueFn: func(value *structpb.Value) interface{} { return value.GetBoolValue() },
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				protoSpec := bldr.ProtoSpec().
					WithOutputSpec(map[string]*proto.Spec_Content_Output{"value": {Type: test.outputType, Default: nil}}).
					Build()

				outputValueB64 := base64.StdEncoding.EncodeToString([]byte(test.outputValue))

				protoDef := bldr.ProtoDef().
					WithExecType("", []string{"/bin/bash", "-c", "echo " + outputValueB64 + " | base64 -d > ${{output_file}}"}).
					Build()
				specDef := bldr.ProtoSpecDef().WithSpec(protoSpec).WithDefinition(protoDef).Build()
				stepsCtx := bldr.StepsContext(t).Build()

				step := runner.NewExecutableStep(runner.StepDefinedInGitLabJob, &runner.Params{}, specDef)
				execStepResult, err := step.Run(context.Background(), stepsCtx.GlobalContext, stepsCtx.StepDir, stepsCtx.Inputs, stepsCtx.Env, nil)
				require.NoError(t, err)
				require.Equal(t, proto.StepResult_success, execStepResult.Status)
				require.Equal(t, test.expected, test.extractValueFn(execStepResult.Outputs["value"]))
			})
		}
	})

	t.Run("delegates output", func(t *testing.T) {
		stepResult := bldr.StepResult().
			WithOutput("name", structpb.NewStringValue("amanda")).
			WithSuccessStatus().
			Build()
		jsonStepResult, err := protojson.Marshal(stepResult)
		require.NoError(t, err)

		protoSpec := bldr.ProtoSpec().WithOutputMethod(proto.OutputMethod_delegate).Build()
		protoDef := bldr.ProtoDef().
			WithEnvVar("STEP_RESULT", base64.StdEncoding.EncodeToString(jsonStepResult)).
			WithExecType("", []string{"/bin/bash", "-c", `echo ${{env.STEP_RESULT}} | base64 -d >${{output_file}}`}).
			Build()
		specDef := bldr.ProtoSpecDef().WithSpec(protoSpec).WithDefinition(protoDef).Build()
		stepsCtx := bldr.StepsContext(t).Build()

		step := runner.NewExecutableStep(runner.StepDefinedInGitLabJob, &runner.Params{}, specDef)
		execStepResult, err := step.Run(context.Background(), stepsCtx.GlobalContext, stepsCtx.StepDir, stepsCtx.Inputs, stepsCtx.Env, nil)
		require.NoError(t, err)
		require.Equal(t, proto.StepResult_success, execStepResult.Status)
		require.Equal(t, "amanda", execStepResult.Outputs["name"].GetStringValue())
	})
}

func TestExecutableStep_Describe(t *testing.T) {
	protoDef := bldr.ProtoDef().WithExecType("", []string{"go", "run", "."}).Build()
	specDef := bldr.ProtoSpecDef().WithDefinition(protoDef).Build()

	step := runner.NewExecutableStep(runner.StepDefinedInGitLabJob, &runner.Params{}, specDef)
	require.Equal(t, `executable step "go run ."`, step.Describe())
}
