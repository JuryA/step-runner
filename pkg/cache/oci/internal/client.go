package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/containerd/platforms"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Client struct {
	cacheDir    string
	layerWriter LayerWriter
}

func NewClient(cacheDir string) *Client {
	return &Client{
		cacheDir:    cacheDir,
		layerWriter: NewDiskLayerWriter(),
	}
}

type PullOption struct {
	Platforms []platforms.Platform
}

func (c *Client) Pull(ctx context.Context, ref name.Reference, opts ...func(*PullOption)) (string, error) {
	options := &PullOption{Platforms: []platforms.Platform{platforms.DefaultSpec(), {OS: "generic"}}}

	for _, opt := range opts {
		opt(options)
	}

	image, err := c.fetchImage(ctx, ref, options.Platforms)
	if err != nil {
		return "", fmt.Errorf("fetching OCI image %q: %w", ref.Name(), err)
	}

	layers, err := image.Layers()
	if err != nil {
		return "", fmt.Errorf("getting layers for OCI image %q: %w", ref.Name(), err)
	}

	stepDir, err := os.MkdirTemp(c.cacheDir, "oci-image-*")
	if err != nil {
		return "", fmt.Errorf("creating download directory for OCI image %q: %w", ref.Name(), err)
	}

	err = c.layerWriter.Write(layers, stepDir)
	if err != nil {
		return "", fmt.Errorf("writing layers for OCI image %q: %w", ref.Name(), err)
	}

	return stepDir, nil
}

// fetchImage is like 'remote.Image()' but uses github.com/containerd/platforms for better platform negotiation.
func (c *Client) fetchImage(ctx context.Context, ref name.Reference, findForPlatform []platforms.Platform) (v1.Image, error) {
	idx, err := remote.Index(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}

	indexManifest, err := idx.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("getting index manifest: %w", err)
	}

	manifest := FindManifestForPlatforms(findForPlatform, indexManifest.Manifests)

	if manifest == nil {
		return nil, fmt.Errorf("didn't find an image matching platform %s", DescribePlatforms(findForPlatform...))
	}

	image, err := idx.Image(manifest.Digest)
	if err != nil {
		return nil, fmt.Errorf("fetching image for manifest %v: %v", manifest.Digest, err)
	}

	return image, nil
}

func WithPlatforms(v1Platforms ...*v1.Platform) func(*PullOption) {
	return func(opt *PullOption) {
		opt.Platforms = make([]platforms.Platform, len(v1Platforms))

		for i := range v1Platforms {
			opt.Platforms[i] = ConvertPlatformV1ToCtrd(v1Platforms[i])
		}
	}
}
