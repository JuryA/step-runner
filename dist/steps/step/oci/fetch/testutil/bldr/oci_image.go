package bldr

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/require"
)

type OCIImageBuilder struct {
	t      *testing.T
	layers []v1.Layer
}

func OCIImage(t *testing.T) *OCIImageBuilder {
	return &OCIImageBuilder{
		t:      t,
		layers: []v1.Layer{},
	}
}

func (b *OCIImageBuilder) WithLayer(layer v1.Layer) *OCIImageBuilder {
	b.layers = append(b.layers, layer)
	return b
}

func (b *OCIImageBuilder) WithFile(path string, content []byte) *OCIImageBuilder {
	b.layers = append(b.layers, OCIImageLayer(b.t).WithFile(path, content).Build())
	return b
}

func (b *OCIImageBuilder) WithEmptyFile(path string) *OCIImageBuilder {
	return b.WithFile(path, []byte{})
}

func (b *OCIImageBuilder) Build() v1.Image {
	img := mutate.MediaType(empty.Image, types.OCIManifestSchema1)

	img, err := mutate.AppendLayers(img, b.layers...)
	require.NoError(b.t, err)

	diffIDs := make([]v1.Hash, len(b.layers))

	for i, layer := range b.layers {
		diffIDs[i], err = layer.DiffID()
		require.NoError(b.t, err)
	}

	cfg, err := img.ConfigFile()
	require.NoError(b.t, err)

	cfg.Config.WorkingDir = "/"
	cfg.Config.Env = []string{"PATH=/usr/bin"}
	cfg.RootFS.DiffIDs = diffIDs

	img, err = mutate.ConfigFile(img, cfg)
	require.NoError(b.t, err)

	return img
}

type fileInfo struct {
	content []byte
	perm    os.FileMode
}

type OCIImageLayerBuilder struct {
	t     *testing.T
	files map[string]fileInfo
}

func OCIImageLayer(t *testing.T) *OCIImageLayerBuilder {
	return &OCIImageLayerBuilder{
		t:     t,
		files: make(map[string]fileInfo),
	}
}

func (b *OCIImageLayerBuilder) WithFile(path string, fileContent []byte) *OCIImageLayerBuilder {
	return b.WithFileWithPerms(path, fileContent, 0644)
}

func (b *OCIImageLayerBuilder) WithFileWithPerms(path string, fileContent []byte, perms os.FileMode) *OCIImageLayerBuilder {
	b.files[path] = fileInfo{content: fileContent, perm: perms}
	return b
}

func (b *OCIImageLayerBuilder) Build() v1.Layer {
	dirsWritten := map[string]struct{}{}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for path, info := range b.files {
		dirs := b.findUnwrittenDirsInPath(path, dirsWritten)

		for _, dir := range dirs {
			err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: dir, Mode: 0777})
			require.NoError(b.t, err)
		}

		err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: path, Size: int64(len(info.content)), Mode: int64(info.perm)})
		require.NoError(b.t, err)

		_, err = tw.Write(info.content)
		require.NoError(b.t, err)
	}

	err := tw.Close()
	require.NoError(b.t, err)

	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
	})
	require.NoError(b.t, err)

	return layer
}

func (b *OCIImageLayerBuilder) findUnwrittenDirsInPath(path string, dirsSeen map[string]struct{}) []string {
	dirs := make([]string, 0)
	dir, _ := filepath.Split(path)

	for {
		if _, seen := dirsSeen[dir]; seen || len(dir) == 0 || dir == "/" {
			break
		}

		dirs = append(dirs, dir)
		dirsSeen[dir] = struct{}{}
		dir, _ = filepath.Split(strings.TrimRight(dir, "/"))
	}

	slices.Reverse(dirs)
	return dirs
}
