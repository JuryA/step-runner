package remote

import (
	"context"
	"os"

	remoteRepo "github.com/google/go-containerregistry/pkg/v1/remote"

	"gitlab.com/gitlab-org/step-runner/dist/steps/oci/fetch/api"
)

func remoteOptions(ctx context.Context) []remoteRepo.Option {
	return []remoteRepo.Option{
		remoteRepo.WithContext(ctx),
		remoteRepo.WithAuthFromKeychain(api.NewAuthLookupKeychain(os.LookupEnv)),
	}
}
