package oci

import (
	"context"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal/client"
)

func List(ctx context.Context, addr string) ([]string, error) {
	c := client.New()

	versions, err := c.List(ctx, addr)
	if err != nil {
		return nil, err
	}

	vers := make([]string, len(versions))
	for _, version := range versions {
		vers = append(vers, version.String())
	}

	return vers, nil
}
