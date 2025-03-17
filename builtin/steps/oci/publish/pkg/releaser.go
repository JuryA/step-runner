package pkg

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Releaser struct {
	logger *slog.Logger
}

func NewReleaser() *Releaser {
	return &Releaser{
		logger: slog.Default(),
	}
}

func (r *Releaser) Release(ctx context.Context, imgRef name.Reference, common Artifacts, platformSpecific Artifacts) error {
	factory := NewImageFactory(WithLogger(r.logger))
	defer factory.CleanUp()

	imagePlatforms := make([]PlatformImage, 0)
	createdAt := time.Now()

	for _, platform := range platformSpecific.Platforms() {
		r.logger.Info("building image", "platform", platform)

		layers, err := r.buildImageLayers(factory, common.Add(platformSpecific.ForPlatform(platform)))
		if err != nil {
			return err
		}

		image, err := factory.BuildImage(createdAt, layers...)
		if err != nil {
			return err
		}

		imagePlatforms = append(imagePlatforms, PlatformImage{Image: image, Platform: platform})
	}

	r.logger.Info("building image index")
	imageIndex := factory.BuildImageIndex(createdAt, imagePlatforms...)

	r.logger.Info("pushing image index")
	return r.pushImageIndex(ctx, imgRef, imageIndex)
}

func (r *Releaser) buildImageLayers(factory *ImageFactory, artifacts Artifacts) ([]v1.Layer, error) {
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

func (r *Releaser) buildImageLayer(factory *ImageFactory, artifact *Artifact) (v1.Layer, error) {
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

func (r *Releaser) pushImageIndex(ctx context.Context, ref name.Reference, index v1.ImageIndex) error {
	err := remote.WriteIndex(ref, index, remote.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("push index image: %w", err)
	}

	return nil
}
