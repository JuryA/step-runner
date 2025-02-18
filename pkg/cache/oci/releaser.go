package oci

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal"
)

type Releaser struct {
	client       *internal.Client
	imageFactory *internal.ImageFactory
}

func NewReleaser(downloadDir string) *Releaser {
	return &Releaser{
		client:       internal.NewClient(downloadDir),
		imageFactory: internal.NewImageFactory(),
	}
}

func (r *Releaser) Release(ctx context.Context, imgRef name.Reference, archiveDir string) error {
	createdAt := time.Now()

	layers, err := r.buildImageLayers(archiveDir)
	if err != nil {
		return err
	}

	image, err := r.imageFactory.BuildImage(createdAt, layers...)
	if err != nil {
		return err
	}

	imageIndex := r.imageFactory.BuildImageIndex(createdAt, &v1.Platform{OS: "linux", Architecture: "amd64"}, image)

	err = r.client.PushImageIndex(ctx, imgRef, imageIndex)
	if err != nil {
		return fmt.Errorf("pushing image index: %w", err)
	}

	return nil
}

func (r *Releaser) buildImageLayers(archiveDir string) ([]v1.Layer, error) {
	layers := make([]v1.Layer, 0)
	archiveFS := os.DirFS(path.Join(archiveDir, "dist"))

	commonLayer, err := r.imageFactory.BuildLayer(archiveFS, "common")
	if err != nil {
		return nil, fmt.Errorf("common: %w", err)
	}
	layers = append(layers, commonLayer)

	platformLayer, err := r.imageFactory.BuildLayer(archiveFS, path.Join("linux", "amd64"))
	if err != nil {
		return nil, err
	}
	layers = append(layers, platformLayer)

	return layers, nil
}
