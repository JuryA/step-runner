package internal

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/compression"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/safearchive/tar"
	"github.com/klauspost/compress/zstd"
)

const (
	StepsOCIArtifact  types.MediaType = "application/vnd.gitlab.step.image.v1"
	StepsOCILayerZSTD types.MediaType = "application/vnd.gitlab.step.layer.v1.tar+zstd"
)

type PlatformImage struct {
	Image    v1.Image
	Platform *v1.Platform
}

type ImageFactory struct {
	workDirMu sync.Mutex
	workDir   string
}

func NewImageFactory() *ImageFactory {
	return &ImageFactory{
		workDir: "",
	}
}

func (f *ImageFactory) BuildImageIndex(createdAt time.Time, imagePlatforms ...PlatformImage) v1.ImageIndex {
	annotations := map[string]string{
		"org.opencontainers.image.created": createdAt.UTC().Format(time.RFC3339),
	}

	index := mutate.Annotations(empty.Index, annotations).(v1.ImageIndex)

	for _, imgPlatform := range imagePlatforms {
		index = mutate.AppendManifests(index, mutate.IndexAddendum{
			Add: imgPlatform.Image,
			Descriptor: v1.Descriptor{
				MediaType: types.OCIManifestSchema1,
				Platform:  imgPlatform.Platform,
				// we could add artifact type here too, github.com/google/go-containerregistry does not support this
			},
		})
	}

	return index
}

func (f *ImageFactory) BuildImage(createdAt time.Time, layers ...v1.Layer) (v1.Image, error) {
	var err error
	annotations := map[string]string{
		"org.opencontainers.image.created": createdAt.UTC().Format(time.RFC3339),
	}

	image := mutate.Annotations(empty.Image, annotations).(v1.Image)
	image = mutate.MediaType(image, types.OCIManifestSchema1)

	// technically, we should set Artifact Type here, though it's unsupported by github.com/google/go-containerregistry
	// see https://github.com/opencontainers/image-spec/blob/fbb4662eb53b80bd38f7597406cf1211317768f0/manifest.md#guidelines-for-artifact-usage
	// as a fallback, the config media type should be set to the artifact type instead of using an "empty" media type

	image = mutate.ConfigMediaType(image, StepsOCIArtifact)
	image, err = mutate.ConfigFile(image, &v1.ConfigFile{})
	if err != nil {
		return nil, fmt.Errorf("build image: empty config: %w", err)
	}

	image, err = mutate.AppendLayers(image, layers...)
	if err != nil {
		return nil, fmt.Errorf("build image: appending layers: %w", err)
	}

	return image, nil
}

func (f *ImageFactory) BuildLayer(archiveFS fs.FS) (v1.Layer, error) {
	archive, err := f.archive(archiveFS)
	if err != nil {
		return nil, fmt.Errorf("creating layer: %w", err)
	}

	opener := func() (io.ReadCloser, error) { return os.Open(archive) }

	layer, err := tarball.LayerFromOpener(opener, tarball.WithMediaType(StepsOCILayerZSTD), tarball.WithCompression(compression.ZStd))
	if err != nil {
		return nil, fmt.Errorf("creating layer: %w", err)
	}

	return layer, nil
}

func (f *ImageFactory) CleanUp() {
	f.workDirMu.Lock()
	defer f.workDirMu.Unlock()

	_ = os.RemoveAll(f.workDir)
	f.workDir = ""
}

func (f *ImageFactory) createWorkDir() (string, error) {
	if f.workDir == "" {
		tempDir, err := os.MkdirTemp("", "")
		if err != nil {
			return "", fmt.Errorf("creating work dir: %w", err)
		}

		f.workDir = tempDir
	}

	return f.workDir, nil
}

func (f *ImageFactory) archive(archiveFS fs.FS) (string, error) {
	f.workDirMu.Lock()
	defer f.workDirMu.Unlock()

	workDir, err := f.createWorkDir()
	if err != nil {
		return "", fmt.Errorf("archive %s: %w", archiveFS, err)
	}

	// potentially unreliable, however in practice, fs.FS will be os.dirFS (a string)
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%s", archiveFS)))
	archiveName := filepath.Join(workDir, fmt.Sprintf("%x.tar.zstd", hash.Sum([]byte{})))

	if _, err := os.Stat(archiveName); err == nil {
		return archiveName, nil
	}

	file, err := os.Create(archiveName)
	if err != nil {
		return "", fmt.Errorf("archive %s: creating file: %w", archiveName, err)
	}
	defer file.Close()

	zw, err := zstd.NewWriter(file)
	if err != nil {
		return "", fmt.Errorf("archive %s: creating zstd writer: %w", archiveName, err)
	}
	defer zw.Close()

	tw := tar.NewWriter(zw)
	defer tw.Close()

	if err := tw.AddFS(archiveFS); err != nil {
		return "", fmt.Errorf("archive: taring directory %s: %w", archiveFS, err)
	}

	if err := tw.Close(); err != nil {
		return "", fmt.Errorf("archive %s: closing tar writer: %w", archiveName, err)
	}

	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("archive %s: closing zstd writer: %w", archiveName, err)
	}

	if err := file.Close(); err != nil {
		return "", fmt.Errorf("archive %s: closing file: %w", archiveName, err)
	}

	return archiveName, nil
}
