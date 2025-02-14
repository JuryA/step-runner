package oci_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
)

func TestOCIFetcher_Fetch(t *testing.T) {
	t.Run("invalid image url", func(t *testing.T) {
		fetcher := oci.NewOCIFetcher(t.TempDir())
		_, err := fetcher.Fetch(context.Background(), "registry.gitlab.com/!", "latest")
		require.Error(t, err)
		require.Contains(t, err.Error(), "OCI image: could not parse reference: registry.gitlab.com/!:latest")
	})

	t.Run("invalid tag", func(t *testing.T) {
		fetcher := oci.NewOCIFetcher(t.TempDir())
		_, err := fetcher.Fetch(context.Background(), "registry.gitlab.com/step-runner", "!err!")
		require.Error(t, err)
		require.Contains(t, err.Error(), "OCI image: could not parse reference: registry.gitlab.com/step-runner:!err!")
	})
}
