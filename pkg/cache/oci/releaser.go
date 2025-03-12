package oci

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal"
)

type Releaser struct {
	client *internal.Client
	logger *slog.Logger
}

func NewReleaser(downloadDir string) *Releaser {
	return &Releaser{
		client: internal.NewClient(downloadDir),
		logger: slog.Default(),
	}
}

func (r *Releaser) Release(ctx context.Context, imgRef name.Reference, artifacts Artifacts) error {
	factory := internal.NewImageFactory(internal.WithLogger(r.logger))
	defer factory.CleanUp()

	imagePlatforms := make([]internal.PlatformImage, 0)
	createdAt := time.Now()

	for _, platform := range artifacts.Platforms() {
		r.logger.Info("building image", "platform", platform)

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

	r.logger.Info("building image index")
	imageIndex := factory.BuildImageIndex(createdAt, imagePlatforms...)

	r.logger.Info("pushing image index")
	return r.client.PushImageIndex(ctx, imgRef, imageIndex)
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
	r.logger.Debug("copying files", "source", artifact.Src, "destination", artifact.Dst)
	fs, cleanup, err := artifact.FS()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := cleanup(); err != nil {
			r.logger.Warn("failed to clean up files used to build image layer", "err", err)
		}
	}()

	r.logger.Debug("adding files to layer", "path", artifact.Dst)
	return factory.BuildLayer(fs)
}
