package runner

import (
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// NamedStepReference is a step that is loaded using a name and a reference
type NamedStepReference struct {
	name string // if name is not specified, it may be empty
	ref  *proto.Step_Reference
}

func NewNamedStepReference(name string, ref *proto.Step_Reference) *NamedStepReference {
	return &NamedStepReference{
		name: name,
		ref:  ref,
	}
}

func (sr *NamedStepReference) ToProtoStep(params *Params) *proto.Step {
	return &proto.Step{
		Name:   sr.name,
		Step:   sr.ref,
		Inputs: sr.formatInputs(params.Inputs),
		Env:    params.Env,
	}
}

func (sr *NamedStepReference) formatInputs(inputs map[string]*context.Variable) map[string]*structpb.Value {
	formatted := make(map[string]*structpb.Value, len(inputs))

	for k, v := range inputs {
		formatted[k] = v.Value
	}

	return formatted
}

func (sr *NamedStepReference) Describe() string {
	return sr.name
}
