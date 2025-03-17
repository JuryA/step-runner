package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/safearchive/sanitizer"
	"github.com/google/safearchive/tar"
)

type LayerWriter interface {
	Write(layers []v1.Layer, dir string) error
}

type DiskLayerWriter struct {
}

func NewDiskLayerWriter() *DiskLayerWriter {
	return &DiskLayerWriter{}
}

func (w *DiskLayerWriter) Write(layers []v1.Layer, outputDir string) error {
	for _, layer := range layers {
		if err := w.writeLayerToDisk(layer, outputDir); err != nil {
			hash, _ := layer.Digest()
			return fmt.Errorf("layer %q: %w", hash, err)
		}
	}

	return nil
}

func (w *DiskLayerWriter) writeLayerToDisk(layer v1.Layer, dir string) error {
	layerRd, err := layer.Uncompressed()
	if err != nil {
		return fmt.Errorf("opening uncompressed reader: %w", err)
	}
	defer layerRd.Close()

	tr := tar.NewReader(layerRd)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return fmt.Errorf("advancing to next tar entry: %w", err)
		}

		filePath := filepath.Join(dir, sanitizer.SanitizePath(hdr.Name))

		switch hdr.Typeflag {
		case tar.TypeDir:
			err = w.writeDir(filePath)
		case tar.TypeReg:
			err = w.writeFile(filePath, tr)
		default:
		}

		if err != nil {
			return err
		}
	}
}

func (w *DiskLayerWriter) writeDir(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating directory %q: %w", dir, err)
	}

	return nil
}

func (w *DiskLayerWriter) writeFile(path string, content io.Reader) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0400)
	if err != nil {
		return fmt.Errorf("creating file %q: %w", path, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, content); err != nil {
		return fmt.Errorf("writing to file %q: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("closing file %q: %w", path, err)
	}

	return nil
}
