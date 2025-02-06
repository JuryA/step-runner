package oci

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/safearchive/sanitizer"
	"github.com/google/safearchive/tar"
)

type Client struct {
	cacheDir string
}

func NewClient(cacheDir string) *Client {
	return &Client{
		cacheDir: cacheDir,
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

	for _, layer := range layers {
		if err := c.writeLayerToDisk(layer, stepDir); err != nil {
			return "", fmt.Errorf("writing layer to disk for OCI image %q: %w", ref.Name(), err)
		}
	}

	return stepDir, nil
}

func (c Client) writeLayerToDisk(layer v1.Layer, dir string) error {
	digest, err := layer.Digest()
	if err != nil {
		return fmt.Errorf("getting layer digest: %w", err)
	}

	layerRd, err := layer.Uncompressed()
	if err != nil {
		return fmt.Errorf("opening uncompressed reader %v: %w", digest, err)
	}
	defer layerRd.Close()

	tr := tar.NewReader(layerRd)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("advancing to next entry in tar archive %v: %w", digest, err)
		}

		filePath := filepath.Join(dir, sanitizer.SanitizePath(hdr.Name))
		filePerm := hdr.FileInfo().Mode()

		switch hdr.Typeflag {
		case tar.TypeDir:
			err = c.writeDir(filePath, filePerm)
		case tar.TypeReg:
			err = c.writeFile(filePath, tr, filePerm)
		default:
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c Client) writeDir(dir string, perm fs.FileMode) error {
	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("creating directory %q: %w", dir, err)
	}

	return nil
}

func (c Client) writeFile(path string, content io.Reader, perm fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, perm)
	if err != nil {
		return fmt.Errorf("creating file %q: %w", path, err)
	}

	if _, err := io.Copy(file, content); err != nil {
		_ = file.Close()
		return fmt.Errorf("writing to file %q: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("closing file %q: %w", path, err)
	}

	return nil
}
