package oci

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
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

func (c Client) Pull(ctx context.Context, ref name.Reference) (string, error) {
	image, err := remote.Image(ref, remote.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("cannot find remote OCI image matching local platform %q: %w", ref.Name(), err)
	}

	layers, err := image.Layers()
	if err != nil {
		return "", fmt.Errorf("getting image layers for OCI image %q: %w", ref.Name(), err)
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
