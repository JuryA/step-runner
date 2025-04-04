package runner_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestOCIStepResource_NamedReference(t *testing.T) {
	tests := []struct {
		name       string
		registry   string
		repository string
		tag        string
		expect     string
		expectErr  string
	}{
		{
			name:       "registry, repository, and tag",
			registry:   "registry.gitlab.com",
			repository: "group/project",
			tag:        "1.0.0",
			expect:     "registry.gitlab.com/group/project:1.0.0",
		},
		{
			name:       "removes extra slashes",
			registry:   "registry.gitlab.com//",
			repository: "/group/project/",
			tag:        "latest",
			expect:     "registry.gitlab.com/group/project:latest",
		},
		{
			name:       "registry with port",
			registry:   "registry.gitlab.com:8080",
			repository: "project",
			tag:        "latest",
			expect:     "registry.gitlab.com:8080/project:latest",
		},
		{
			name:       "invalid registry",
			registry:   "registry.gitlab.com/!",
			repository: "project",
			tag:        "latest",
			expectErr:  "could not parse reference: registry.gitlab.com/!/project:latest",
		},
		{
			name:       "invalid tag",
			registry:   "registry.gitlab.com",
			repository: "project",
			tag:        "!err!",
			expectErr:  "could not parse reference: registry.gitlab.com/project:!err!",
		},
		{
			name:       "registry, repository, and digest",
			registry:   "registry.gitlab.com",
			repository: "project",
			tag:        "sha256:f271d3fd90442470614813bd422ad3c1a8286e79904ba4faeca94a3fd0fb5b24",
			expect:     "registry.gitlab.com/project@sha256:f271d3fd90442470614813bd422ad3c1a8286e79904ba4faeca94a3fd0fb5b24",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resource := runner.NewOCIStepResource(nil, test.registry, test.repository, test.tag, "", "step.yml")
			reference, err := resource.NamedReference()
			if test.expectErr == "" {
				require.NoError(t, err)
				require.Equal(t, test.expect, reference.String())
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectErr)
			}
		})
	}
}

func TestOCIStepResource_Fetch(t *testing.T) {
	t.Run("loads OCI step", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		layer := bldr.OCIImageLayer(t).WithFile("/step.yml", []byte("spec:\n---\nexec: {command: [bash]}")).Build()
		img := bldr.OCIImage(t).WithLayer(layer).Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(remoteImgRef, imgIndex)

		fetcher := oci.NewOCIFetcher(t.TempDir())
		res := runner.NewOCIStepResource(fetcher, registry.Address(), "my-image", "latest", "", "step.yml")
		specDef, err := res.Fetch(context.Background())
		require.NoError(t, err)
		require.Equal(t, []string{"bash"}, specDef.Definition.Exec.Command)
	})

	t.Run("loads OCI step in sub-directory", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		remoteImgRef := registry.RefToImage("my-image", "latest")

		layer := bldr.
			OCIImageLayer(t).
			WithFile("/foo/bar/bob/step.yml", []byte("spec:\n---\nexec: {command: [bash]}")).
			Build()
		img := bldr.OCIImage(t).WithLayer(layer).Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(remoteImgRef, imgIndex)

		fetcher := oci.NewOCIFetcher(t.TempDir())
		res := runner.NewOCIStepResource(fetcher, registry.Address(), "my-image", "latest", "foo/bar/bob", "step.yml")
		specDef, err := res.Fetch(context.Background())
		require.NoError(t, err)
		require.Equal(t, []string{"bash"}, specDef.Definition.Exec.Command)
	})

	t.Run("loads OCI step using a digest", func(t *testing.T) {
		registry := bldr.StartOCIRegistryServer(t)
		layer := bldr.
			OCIImageLayer(t).
			WithFile("/step.yml", []byte("spec:\n---\nexec: {command: [sh]}")).
			Build()
		img := bldr.OCIImage(t).WithLayer(layer).Build()
		imgIndex := bldr.OCIImageIndex(t).WithImageForThisPlatform(img).Build()
		registry.PushImageIndex(registry.RefToImage("image", "latest"), imgIndex)

		digest, err := imgIndex.Digest()
		require.NoError(t, err)

		fetcher := oci.NewOCIFetcher(t.TempDir())
		res := runner.NewOCIStepResource(fetcher, registry.Address(), "image", digest.String(), "", "step.yml")
		specDef, err := res.Fetch(context.Background())
		require.NoError(t, err)
		require.Equal(t, []string{"sh"}, specDef.Definition.Exec.Command)
	})
}
