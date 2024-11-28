package runner

import (
	"os"
	"strconv"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

var RunningInDebugMode, _ = strconv.ParseBool(os.Getenv("CI_STEPS_DEBUG"))

// Params are the input and environment parameters for an execution.
type Params struct {
	Inputs map[string]*context.Variable
	Env    map[string]string
}

func (p *Params) NewInputsWithDefault(specInputs map[string]*proto.Spec_Content_Input) map[string]*structpb.Value {
	newInputs := make(map[string]*structpb.Value)

	for key, value := range specInputs {
		if p.Inputs[key] != nil {
			newInputs[key] = p.Inputs[key].Value
		} else {
			newInputs[key] = value.Default
		}
	}

	return newInputs
}
