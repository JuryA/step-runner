package oci

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"

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

func (f *OCIFetcher) Fetch(ctx context.Context, url, tag string, opts ...func(*internal.PullOption)) (string, error) {
	urlAndTag := fmt.Sprintf("%s:%s", url, tag)
	imgRef, err := name.ParseReference(urlAndTag)
	if err != nil {
		return "", fmt.Errorf("OCI image: %w", err)
	}

	dir, err := f.client.Pull(ctx, imgRef, opts...)
	if err != nil {
		return "", err
	}

	return dir, nil
}
