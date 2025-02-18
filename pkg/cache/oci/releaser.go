package oci

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/compression"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/safearchive/tar"
	"github.com/klauspost/compress/zstd"

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

func (r *Releaser) Release(ctx context.Context, imgRef name.Reference, outputDir string) error {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}

	layers, err := r.buildImageLayers(outputDir, tempDir)
	if err != nil {
		return err
	}

	image := mutate.ConfigMediaType(empty.Image, types.OCIConfigJSON)

	for _, layer := range layers {
		image, err = mutate.Append(image, mutate.Addendum{Layer: layer})
		if err != nil {
			return fmt.Errorf("appending layer to image: %w", err)
		}
	}

	index := v1.ImageIndex(empty.Index)
	index = mutate.AppendManifests(index, mutate.IndexAddendum{
		Add: image,
		Descriptor: v1.Descriptor{
			Platform: &v1.Platform{OS: "linux", Architecture: "amd64"},
		},
	})

	err = r.client.PushImageIndex(ctx, imgRef, index)
	if err != nil {
		return fmt.Errorf("pushing image index: %w", err)
	}

	return nil
}

func (r *Releaser) buildImageLayers(archiveDir string, tempDir string) ([]v1.Layer, error) {
	layers := make([]v1.Layer, 0)
	archiveFS := os.DirFS(path.Join(archiveDir, "dist"))

	commonLayer, err := r.buildLayer(archiveFS, "common", tempDir)
	if err != nil {
		return nil, err
	}
	layers = append(layers, commonLayer)

	platformLayer, err := r.buildLayer(archiveFS, path.Join("linux", "amd64"), tempDir)
	if err != nil {
		return nil, err
	}
	layers = append(layers, platformLayer)

	return layers, nil
}

func (r *Releaser) buildLayer(archiveFS fs.FS, subDir, outputDir string) (v1.Layer, error) {
	archive, err := r.platformArchive(archiveFS, subDir, outputDir)
	if err != nil {
		return nil, fmt.Errorf("archiving %s: %w", subDir, err)
	}

	opener := func() (io.ReadCloser, error) { return os.Open(archive) }

	layer, err := tarball.LayerFromOpener(opener, tarball.WithCompression(compression.ZStd))
	if err != nil {
		return nil, fmt.Errorf("creating layer: %w", err)
	}

	return layer, nil
}

func (r *Releaser) platformArchive(archiveFS fs.FS, subDir string, outputDir string) (string, error) {
	subDirFS, err := fs.Sub(archiveFS, subDir)
	if err != nil {
		return "", fmt.Errorf("reading directory %q: %w", subDir, err)
	}

	archiveFile := fmt.Sprintf("%s.tar.zstd", strings.ReplaceAll(subDir, "/", "_"))
	archiveName := filepath.Join(outputDir, archiveFile)

	f, err := os.Create(archiveName)
	if err != nil {
		return "", fmt.Errorf("creating archive: %w", err)

	}
	defer f.Close()

	zw, err := zstd.NewWriter(f)
	if err != nil {
		return "", fmt.Errorf("creating zstd writer: %w", err)
	}
	defer zw.Close()

	tw := tar.NewWriter(zw)
	defer tw.Close()

	if err := tw.AddFS(subDirFS); err != nil {
		return "", fmt.Errorf("taring directory %s: %w", subDirFS, err)
	}

	if err := tw.Close(); err != nil {
		return "", fmt.Errorf("closing tar writer: %w", err)
	}

	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("closing zstd writer: %w", err)
	}

	if err := f.Close(); err != nil {
		return "", fmt.Errorf("closing file %s: %w", archiveName, err)
	}

	return archiveName, nil
}
