package remote

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	remoteRepo "github.com/google/go-containerregistry/pkg/v1/remote"
)

func Copy(ctx context.Context, from name.Reference, to name.Reference) error {
	imgIndex, err := remoteRepo.Index(from, remoteOptions(ctx)...)
	if err == nil {
		return copyImageIndex(ctx, imgIndex, to)
	}

	return copyImage(ctx, from, to)
}

func copyImageIndex(ctx context.Context, imgIndex v1.ImageIndex, to name.Reference) error {
	err := remoteRepo.WriteIndex(to, imgIndex, remoteOptions(ctx)...)
	if err != nil {
		return fmt.Errorf("write image index to %q: %w", to, err)
	}

	return nil
}

func copyImage(ctx context.Context, from name.Reference, to name.Reference) error {
	img, err := remoteRepo.Image(from, remoteOptions(ctx)...)
	if err != nil {
		return fmt.Errorf("fetch image metadata %q: %w", from, err)
	}

	err = remoteRepo.Write(to, img, remoteOptions(ctx)...)
	if err != nil {
		return fmt.Errorf("write image to %q: %w", to, err)
	}

	return nil
}

func remoteOptions(ctx context.Context) []remoteRepo.Option {
	return []remoteRepo.Option{
		remoteRepo.WithContext(ctx),
	}
}
