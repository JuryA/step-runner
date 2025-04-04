package runner

import (
	"context"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Cache interface {
	Get(ctx context.Context, stepResource StepResource) (*proto.SpecDefinition, error)
}
