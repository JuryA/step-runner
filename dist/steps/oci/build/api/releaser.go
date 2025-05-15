package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

	"gitlab.com/gitlab-org/step-runner/dist/steps/oci/fetch/api"

	"gitlab.com/gitlab-org/step-runner/dist/steps/oci/build/internal"
)

type Releaser struct {
	logger *slog.Logger
}

func NewReleaser() *Releaser {
	return &Releaser{
		logger: slog.Default(),
	}
}

func (r *Releaser) Release(ctx context.Context, remoteImgRef *internal.RemoteImageRef, common internal.Artifacts, platformSpecific internal.Artifacts) (v1.ImageIndex, error) {
	factory := internal.NewImageFactory(internal.WithLogger(r.logger))
	defer factory.CleanUp()

	if r.alreadyPublished(ctx, remoteImgRef.MajorMinorPatch()) {
		return nil, fmt.Errorf("image already published: %s", remoteImgRef.MajorMinorPatch())
	}

	imageIndex, err := r.buildImageIndex(factory, common, platformSpecific)
	if err != nil {
		return nil, err
	}

	tags, err := r.listPublishedTags(ctx, remoteImgRef)
	if err != nil {
		return nil, err
	}

	semVerRefs, err := remoteImgRef.SemVerRefs(tags)
	if err != nil {
		return nil, err
	}

	for _, ref := range semVerRefs {
		if err := r.pushImageIndex(ctx, ref, imageIndex); err != nil {
			return nil, err
		}
	}

	return imageIndex, nil
}

func (r *Releaser) buildImageIndex(factory *internal.ImageFactory, common, platformSpecific internal.Artifacts) (v1.ImageIndex, error) {
	imagePlatforms := make([]internal.PlatformImage, 0)
	createdAt := time.Now()

	for _, platform := range platformSpecific.Platforms() {
		r.logger.Info("building image", "platform", platform)

		layers, err := r.buildImageLayers(factory, common.Add(platformSpecific.ForPlatform(platform)))
		if err != nil {
			return nil, err
		}

		image, err := factory.BuildImage(createdAt, layers...)
		if err != nil {
			return nil, err
		}

		imagePlatforms = append(imagePlatforms, internal.PlatformImage{Image: image, Platform: platform})
	}

	r.logger.Info("building image index")
	return factory.BuildImageIndex(createdAt, imagePlatforms...), nil
}

func (r *Releaser) buildImageLayers(factory *internal.ImageFactory, artifacts internal.Artifacts) ([]v1.Layer, error) {
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

func (r *Releaser) buildImageLayer(factory *internal.ImageFactory, artifact *internal.Artifact) (v1.Layer, error) {
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

func (r *Releaser) pushImageIndex(ctx context.Context, ref name.Reference, imageIndex v1.ImageIndex) error {
	digest, err := imageIndex.Digest()
	if err != nil {
		return fmt.Errorf("getting digest of image index: %w", err)
	}

	r.logger.Info("pushing image index", "image_digest", digest.String(), "destination", ref.Name())

	err = remote.WriteIndex(ref, imageIndex, r.remoteOptions(ctx)...)
	if err != nil {
		return fmt.Errorf("push index image: %w", err)
	}

	return nil
}

func (r *Releaser) alreadyPublished(ctx context.Context, imgRef name.Reference) bool {
	r.logger.Debug("checking if image has already been published")

	descriptor, _ := remote.Head(imgRef, r.remoteOptions(ctx)...)

	if descriptor == nil {
		r.logger.Debug("image has not been published")
	}

	return descriptor != nil
}

// listPublishedTags lists the tags for the remote image
// for example, for registry.gitlab.com/gitlab-org/gitlab-runner it will return v17.7.0, v17.7.1, v17.8.0, v17.8.1, etc
// returns no tags if the repository is not found
func (r *Releaser) listPublishedTags(ctx context.Context, remoteImgRef *internal.RemoteImageRef) ([]string, error) {
	r.logger.Debug("listing published tags")

	tags, err := remote.List(remoteImgRef.Repository(), r.remoteOptions(ctx)...)

	if err != nil {
		if isErrorNameUnknown(err) {
			r.logger.Debug("no published tags, repository not found")
			return []string{}, nil
		}

		return nil, fmt.Errorf("listing published tags: %w", err)
	}

	if r.logger.Enabled(ctx, slog.LevelDebug) {
		for chunk := range slices.Chunk(tags, 20) {
			r.logger.Debug("published tags", "tags", strings.Join(chunk, ","))
		}

		if len(tags) == 0 {
			r.logger.Debug("no published tags")
		}
	}

	return tags, nil
}

func (r *Releaser) remoteOptions(ctx context.Context) []remote.Option {
	return []remote.Option{
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(api.NewAuthLookupKeychain(os.LookupEnv)),
	}
}

func isErrorNameUnknown(err error) bool {
	for err != nil {
		transportErr, ok := err.(*transport.Error)
		if ok {
			for _, diagnostic := range transportErr.Errors {
				if diagnostic.Code == transport.NameUnknownErrorCode {
					return true
				}
			}
		}

		err = errors.Unwrap(err)
	}

	return false
}
