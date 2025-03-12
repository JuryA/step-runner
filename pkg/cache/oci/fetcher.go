package oci

import (
	"context"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal"
)

type OCIFetcher struct {
	client *internal.Client
}

func NewOCIFetcher(downloadDir string) *OCIFetcher {
	return &OCIFetcher{
		client: internal.NewClient(downloadDir),
	}
}

func (f *OCIFetcher) Fetch(ctx context.Context, imgRef name.Reference, opts ...func(*internal.PullOption)) (string, error) {
	dir, err := f.client.Pull(ctx, imgRef, opts...)
	if err != nil {
		return "", err
	}

	return dir, nil
}

func WithPlatforms(platforms ...*v1.Platform) func(*internal.PullOption) {
	return internal.WithPlatforms(platforms...)
}
