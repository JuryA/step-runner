package runner

import (
	ctx "context"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Step interface {
	Describer
	Run(ctx ctx.Context, globalCtx *GlobalContext, stepDir string, inputs map[string]*structpb.Value, env *Environment, steps map[string]*proto.StepResult) (*proto.StepResult, error)
}
