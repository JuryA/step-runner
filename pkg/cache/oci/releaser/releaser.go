package releaser

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/klauspost/compress/zstd"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal/client"
)

type packageConfig struct {
	OS      string
	Arch    string
	Variant string
	Dir     string
}

func (pkg packageConfig) String() string {
	var sb strings.Builder

	sb.WriteString(pkg.OS)

	if pkg.Arch != "" {
		sb.WriteByte('/')
		sb.WriteString(pkg.Arch)
	}

	if pkg.Variant != "" {
		sb.WriteByte('/')
		sb.WriteString(pkg.Variant)
	}

	return sb.String()
}

func Release(ctx context.Context, addr string, opts ...Option) (string, error) {
	var options options

	for _, o := range opts {
		err := o(&options)
		if err != nil {
			return "", err
		}
	}

	if options.dir == "" {
		options.dir = "dist"
	}

	if options.logger == nil {
		options.logger = slog.Default()
	}

	packages := []packageConfig{
		{OS: "generic", Arch: "generic", Dir: "generic"},
		{OS: "linux", Arch: "386", Dir: "linux/386"},
		{OS: "linux", Arch: "amd64", Dir: "linux/amd64"},
		{OS: "linux", Arch: "arm", Variant: "v5", Dir: "linux/armv5"},
		{OS: "linux", Arch: "arm", Variant: "v6", Dir: "linux/armv6"},
		{OS: "linux", Arch: "arm", Variant: "v7", Dir: "linux/armv7"},
		{OS: "linux", Arch: "arm64", Dir: "linux/arm64"},
		{OS: "linux", Arch: "mips64le", Dir: "linux/mips64le"},
		{OS: "linux", Arch: "ppc64le", Dir: "linux/ppc64le"},
		{OS: "linux", Arch: "riscv64", Dir: "linux/riscv64"},
		{OS: "linux", Arch: "s390x", Dir: "linux/s390x"},
		{OS: "freebsd", Arch: "386", Dir: "freebsd/386"},
		{OS: "freebsd", Arch: "amd64", Dir: "freebsd/amd64"},
		{OS: "freebsd", Arch: "arm", Variant: "v5", Dir: "freebsd/armv5"},
		{OS: "freebsd", Arch: "arm", Variant: "v6", Dir: "freebsd/armv6"},
		{OS: "freebsd", Arch: "arm", Variant: "v7", Dir: "freebsd/armv7"},
		{OS: "freebsd", Arch: "arm64", Dir: "freebsd/arm64"},
		{OS: "darwin", Arch: "amd64", Dir: "darwin/amd64"},
		{OS: "darwin", Arch: "arm64", Dir: "darwin/arm64"},
		{OS: "windows", Arch: "386", Dir: "windows/386"},
		{OS: "windows", Arch: "amd64", Dir: "windows/amd64"},
		{OS: "zos", Arch: "s390x", Dir: "zos/s390x"},
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", fmt.Errorf("creating temporary directory: %w", err)
	}

	var artifacts []client.Artifact

	dir := os.DirFS(options.dir)

	commonDir, err := sub(dir, "common")
	if err != nil {
		return "", fmt.Errorf("common sub dir: %w", err)
	}

	for _, pkg := range packages {
		pkgDir, err := sub(dir, pkg.Dir)
		if err != nil {
			return "", fmt.Errorf("%v sub dir: %w", pkg.Dir, err)
		}

		found := pkgDir != nil
		options.logger.Info("packaging", "platform", &pkg, "dir", filepath.Join(options.dir, pkg.Dir), "skipping", !found)
		if !found {
			continue
		}

		var readers []func() (io.ReadCloser, error)
		if commonDir != nil {
			readers = append(readers, archiveReader(tmpDir, "common", commonDir))
		}

		readers = append(readers, archiveReader(tmpDir, pkg.String(), pkgDir))

		artifacts = append(artifacts, client.Artifact{
			ReaderFn: readers,
			Platform: v1.Platform{Architecture: pkg.Arch, OS: pkg.OS, Variant: pkg.Variant},
		})
	}

	return client.New().Push(ctx, addr, artifacts)
}

func archiveReader(tmpDir, name string, fsys fs.FS) func() (io.ReadCloser, error) {
	name = strings.ReplaceAll(name, "/", "-")

	return func() (io.ReadCloser, error) {
		archiveName := filepath.Join(tmpDir, name)

		_, err := os.Stat(archiveName)
		if err == nil {
			// use cached archive
			return os.Open(archiveName)
		}

		// create new archive
		f, err := os.Create(archiveName)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		zw, err := zstd.NewWriter(f)
		if err != nil {
			return nil, err
		}
		defer zw.Close()

		tw := tar.NewWriter(zw)
		defer tw.Close()

		if err := tw.AddFS(fsys); err != nil {
			return nil, err
		}

		if err := tw.Close(); err != nil {
			return nil, err
		}

		if err := zw.Close(); err != nil {
			return nil, err
		}

		if err := f.Close(); err != nil {
			return nil, err
		}

		return os.Open(archiveName)
	}
}

func sub(fsys fs.FS, dir string) (fs.FS, error) {
	_, err := fs.Stat(fsys, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return fs.Sub(fsys, dir)
}
