package remote

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	remoteRepo "github.com/google/go-containerregistry/pkg/v1/remote"
)

func CopyAll(ctx context.Context, from name.Reference, toRefs ...name.Reference) error {
	for _, to := range toRefs {
		if err := Copy(ctx, from, to); err != nil {
			return err
		}
	}

	return nil
}

func Copy(ctx context.Context, from name.Reference, to name.Reference) error {
	slog.Debug("fetching reference to image index", "repository", from)

	imgIndex, err := remoteRepo.Index(from, remoteOptions(ctx)...)
	if err == nil {
		return copyImageIndex(ctx, imgIndex, from, to)
	}

	return copyImage(ctx, from, to)
}

func copyImageIndex(ctx context.Context, imgIndex v1.ImageIndex, from name.Reference, to name.Reference) error {
	slog.Debug("issuing remote copy of image index", "from", from, "to", to)

	err := remoteRepo.WriteIndex(to, imgIndex, remoteOptions(ctx)...)
	if err != nil {
		return fmt.Errorf("write image index to %q: %w", to, err)
	}

	slog.Info("promoted image index", "repository", to)
	return nil
}

func copyImage(ctx context.Context, from name.Reference, to name.Reference) error {
	slog.Debug("fetching reference to image", "repository", from)

	img, err := remoteRepo.Image(from, remoteOptions(ctx)...)
	if err != nil {
		return fmt.Errorf("fetch image metadata %q: %w", from, err)
	}

	slog.Debug("issuing remote copy of image", "from", from, "to", to)

	err = remoteRepo.Write(to, img, remoteOptions(ctx)...)
	if err != nil {
		return fmt.Errorf("write image to %q: %w", to, err)
	}

	slog.Info("promoted image", "repository", to)
	return nil
}
