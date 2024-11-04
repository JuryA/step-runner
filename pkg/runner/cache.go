package runner

import (
	"context"

	"gitlab.com/gitlab-org/step-runner/proto"
)

// The need for this interface to avoid a cyclical dependency
// indicates we've got the package structure wrong
type Cache interface {
	Get(ctx context.Context, parentDir string, stepResource StepResource) (*proto.SpecDefinition, error)
}

// Hacky hack hack. To be set by a magical fairy before we need it.
var CacheSingleton Cache
