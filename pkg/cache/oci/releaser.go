package oci

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal"
)

type Releaser struct {
	client *internal.Client
}

func NewReleaser(downloadDir string) *Releaser {
	return &Releaser{
		client: internal.NewClient(downloadDir),
	}
}

func (r *Releaser) Release(ctx context.Context, imgRef name.Reference, artifacts Artifacts) error {
	factory := internal.NewImageFactory()
	defer factory.CleanUp()

	imagePlatforms := make([]internal.PlatformImage, 0)
	createdAt := time.Now()

	for _, platform := range artifacts.Platforms() {
		layers, err := r.buildImageLayers(factory, artifacts.Generic().Add(artifacts.ForPlatform(platform)))
		if err != nil {
			return err
		}

		image, err := factory.BuildImage(createdAt, layers...)
		if err != nil {
			return err
		}

		imagePlatforms = append(imagePlatforms, internal.PlatformImage{Image: image, Platform: platform})
	}

	imageIndex := factory.BuildImageIndex(createdAt, imagePlatforms...)

	err := r.client.PushImageIndex(ctx, imgRef, imageIndex)
	if err != nil {
		return err
	}

	return nil
}

func (r *Releaser) buildImageLayers(factory *internal.ImageFactory, artifacts Artifacts) ([]v1.Layer, error) {
	layers := make([]v1.Layer, 0)

	for _, artifact := range artifacts {
		layer, err := r.buildImageLayer(factory, artifact)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", artifact, err)
		}

		layers = append(layers, layer)
	}

	return layers, nil
}

func (r *Releaser) buildImageLayer(factory *internal.ImageFactory, artifact *Artifact) (v1.Layer, error) {
	fs, cleanup, err := artifact.FS()
	if err != nil {
		return nil, err
	}
	defer func() { _ = cleanup() }()

	layer, err := factory.BuildLayer(fs)
	if err != nil {
		return nil, err
	}

	if err := cleanup(); err != nil {
		return nil, err
	}

	return layer, nil
}
