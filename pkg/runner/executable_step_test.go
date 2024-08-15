package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestExecutableStep_Describe(t *testing.T) {
	specDef := &proto.SpecDefinition{
		Spec: &proto.Spec{
			Spec: &proto.Spec_Content{
				Inputs:       map[string]*proto.Spec_Content_Input{},
				Outputs:      map[string]*proto.Spec_Content_Output{},
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
			Outputs:  map[string]*structpb.Value(nil),
			Env:      map[string]string{},
			Delegate: "",
		},
		Dir: "",
	}

	step := NewExecutableStep(specDef)
	require.Equal(t, `executable step "go run ."`, step.Describe())
}
