package runner

import (
	"context"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Cache interface {
	Get(ctx context.Context, parentDir string, step *proto.Step_Reference) (*proto.SpecDefinition, error)
}
