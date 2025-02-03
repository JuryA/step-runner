package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/containerd/platforms"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/compression"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/safearchive/sanitizer"
	"github.com/google/safearchive/tar"
	"golang.org/x/mod/module"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal/version"
)

const (
	StepVersionAnnotation = "com.gitlab.step.version"

	DefaultRegistry  = "registry.gitlab.com"
	DefaultNamespace = "components"

	StepLayerZstd types.MediaType = "application/vnd.gitlab.step.layer.v1.tar+zstd"
)

type Artifact struct {
	ReaderFn []func() (io.ReadCloser, error)
	Platform v1.Platform
}

type TagAlias struct {
	Alias   string
	Version version.Version
}

type Client struct {
	remoteOpts []remote.Option
}

// New returns a new client.
func New() *Client {
	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
		remote.WithJobs(4),
	}

	return &Client{remoteOpts: remoteOpts}
}

func (c *Client) withRemoteOpts(ctx context.Context, opts ...remote.Option) []remote.Option {
	opts = append(c.remoteOpts, opts...)

	return append(opts, remote.WithContext(ctx))
}

// List returns versions from the remote repository that match the provided
// constraint.
func (c *Client) List(ctx context.Context, addr string) ([]version.Version, error) {
	ref, err := ParseReference(addr, name.WithDefaultTag(""))
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %w", addr, err)
	}

	tags, err := remote.List(ref.Context(), c.withRemoteOpts(ctx)...)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	constraint, err := version.NewConstraint(ref.Identifier())
	if err != nil {
		return nil, fmt.Errorf("parsing constraint: %w", err)
	}

	var versions []version.Version
	for _, tag := range tags {
		v, err := version.New(tag)
		if err != nil {
			// ignore invalid versions and aliases
			continue
		}

		versions = append(versions, v)
	}

	return constraint.Match(versions), nil
}

// image is like 'remote.Image()' but uses
// github.com/containerd/platforms for better platform negotiation.
func (c *Client) image(ctx context.Context, ref name.Reference, platform *v1.Platform) (v1.Image, error) {
	idx, err := remote.Index(ref, c.withRemoteOpts(ctx)...)
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}

	indexManifest, err := idx.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("getting index manifest: %w", err)
	}

	var matcher platforms.MatchComparer
	if platform == nil {
		// match system spec... and then "generic".
		matcher = platforms.Ordered(platforms.DefaultSpec(), platforms.Platform{
			OS: "generic",
		})
	} else {
		matcher = platforms.Only(platforms.Platform{
			Architecture: platform.Architecture,
			OS:           platform.OS,
			OSVersion:    platform.OSVersion,
			OSFeatures:   platform.OSFeatures,
			Variant:      platform.Variant,
		})
	}

	for _, manifest := range indexManifest.Manifests {
		platform := platforms.Platform{
			Architecture: manifest.Platform.Architecture,
			OS:           manifest.Platform.OS,
			OSVersion:    manifest.Platform.OSVersion,
			OSFeatures:   manifest.Platform.OSFeatures,
			Variant:      manifest.Platform.Variant,
		}

		if matcher.Match(platform) {
			image, err := idx.Image(manifest.Digest)
			if err != nil {
				log.Fatalf("fetching image for manifest %v: %v", manifest.Digest, err)
			}

			return image, nil
		}
	}

	return nil, fmt.Errorf("didn't find an image matching platform")
}

// Push pushes a new artifact and updates version aliases as necessary.
func (c *Client) Push(ctx context.Context, addr string, artifacts []Artifact) (string, error) {
	dest, err := ParseReference(addr, name.WithDefaultTag(""))
	if err != nil {
		return "", fmt.Errorf("parsing reference %q: %w", addr, err)
	}

	if _, err := version.New(dest.Identifier()); err != nil {
		return "", fmt.Errorf("parsing version: %w", err)
	}

	index := v1.ImageIndex(empty.Index)
	for idx := range artifacts {
		artifact := artifacts[idx]

		image := mutate.ConfigMediaType(empty.Image, types.OCIConfigJSON)

		annotations := map[string]string{
			StepVersionAnnotation:              dest.Identifier(),
			"org.opencontainers.image.version": dest.Identifier(),
			"org.opencontainers.image.created": time.Now().UTC().Format(time.RFC3339),
		}

		image = mutate.Annotations(image, annotations).(v1.Image)

		for _, fn := range artifact.ReaderFn {
			fn := fn
			layer, err := tarball.LayerFromOpener(fn, tarball.WithCompression(compression.ZStd), tarball.WithMediaType(StepLayerZstd))
			if err != nil {
				return "", fmt.Errorf("layer from opener: %w", err)
			}

			image, err = mutate.Append(image,
				mutate.Addendum{
					Layer: layer,
				},
			)
			if err != nil {
				return "", fmt.Errorf("appending content failed: %w", err)
			}
		}

		index = mutate.AppendManifests(index, mutate.IndexAddendum{
			Add: image,
			Descriptor: v1.Descriptor{
				Platform: &artifact.Platform,
			},
		})
	}

	if err := remote.WriteIndex(dest, index, c.withRemoteOpts(ctx)...); err != nil {
		return "", fmt.Errorf("writing index: %w", err)
	}

	if _, err := c.UpdateAliases(ctx, addr, false); err != nil {
		return "", fmt.Errorf("updating aliases: %w", err)
	}

	h, err := index.Digest()
	if err != nil {
		return "", fmt.Errorf("getting digest: %w", err)
	}

	return dest.Context().Digest(h.String()).String(), nil
}

// UpdateAliases updates any out-of-sync version aliases.
func (c *Client) UpdateAliases(ctx context.Context, addr string, dryrun bool) ([]TagAlias, error) {
	ref, err := ParseReference(addr, name.WithDefaultTag(""))
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %w", addr, err)
	}

	tags, err := remote.List(ref.Context(), c.withRemoteOpts(ctx)...)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	constraint, err := version.NewConstraint(ref.Identifier())
	if err != nil {
		return nil, fmt.Errorf("parsing constraint: %w", err)
	}

	type aliasInfo struct {
		Version version.Version
		Digest  *v1.Hash
	}

	aliases := map[string]aliasInfo{"latest": {}}

	var versions []version.Version
	for _, tag := range tags {
		v, err := version.New(tag)
		if err != nil {
			// ignore invalid versions
			continue
		}

		versions = append(versions, v)

		majorAlias := strconv.FormatInt(v.Major, 10)
		minorAlias := strconv.FormatInt(v.Major, 10) + "." + strconv.FormatInt(v.Minor, 10)

		if v.GreaterThan(aliases["latest"].Version) {
			aliases["latest"] = aliasInfo{Version: v}
		}

		if v.GreaterThan(aliases[majorAlias].Version) {
			aliases[majorAlias] = aliasInfo{Version: v}
		}

		if v.GreaterThan(aliases[minorAlias].Version) {
			aliases[minorAlias] = aliasInfo{Version: v}
		}
	}

	var results []TagAlias
	for _, v := range constraint.Match(versions) {
		versionDesc, err := remote.Head(ref.Context().Tag(v.String()), c.withRemoteOpts(ctx)...)
		if err != nil {
			return nil, fmt.Errorf("getting descriptor head for version %v: %w", v, err)
		}

		for alias, info := range aliases {
			if !v.Equal(info.Version) {
				continue
			}

			if info.Digest == nil {
				desc, err := remote.Head(ref.Context().Tag(alias), c.withRemoteOpts(ctx)...)
				if is404(err) {
					desc = &v1.Descriptor{}
					err = nil
				}
				if err != nil {
					return nil, fmt.Errorf("getting descriptor head for alias %v: %w", alias, err)
				}

				info.Digest = &desc.Digest
			}

			if versionDesc.Digest == *info.Digest {
				continue
			}

			results = append(results, TagAlias{
				Alias:   alias,
				Version: v,
			})
		}
	}

	var errs []error
	if !dryrun {
		descriptorCache := make(map[string]*remote.Descriptor)
		for _, result := range results {
			key := result.Version.String()

			if _, ok := descriptorCache[key]; ok {
				continue
			}

			desc, err := remote.Get(ref.Context().Tag(key), c.withRemoteOpts(ctx)...)
			if err != nil {
				return nil, fmt.Errorf("getting descriptor for version %v: %w", result.Version, err)
			}

			descriptorCache[key] = desc
		}

		for _, result := range results {
			err := remote.Tag(ref.Context().Tag(result.Alias), descriptorCache[result.Version.String()], c.withRemoteOpts(ctx)...)
			if err != nil {
				errs = append(errs, fmt.Errorf("tagging alias %v=%v: %w", result.Alias, result.Version, err))
			}
		}
	}

	return results, errors.Join(errs...)
}

// Pull finds and extracts the step from the repository and unpacks it to the
// provided directory.
func (c *Client) Pull(ctx context.Context, addr, dir string) error {
	os.MkdirAll(filepath.Join(dir, "temp"), 0o777)

	ref, err := ParseReference(addr)
	if err != nil {
		return fmt.Errorf("parsing reference: %v", err)
	}

	image, err := c.image(ctx, ref, nil)
	if err != nil {
		return fmt.Errorf("getting image: %w", err)
	}

	manifest, err := image.Manifest()
	if err != nil {
		return fmt.Errorf("getting image manifest: %w", err)
	}

	layers, err := image.Layers()
	if err != nil {
		return fmt.Errorf("getting image layers: %w", err)
	}

	ver, ok := manifest.Annotations[StepVersionAnnotation]
	if !ok {
		return fmt.Errorf("step version not found")
	}
	if _, err := version.New(ver); err != nil {
		return fmt.Errorf("step version is invalid: %w", err)
	}

	stepDir, stepVersion, err := stepPath(ref.Context().Name(), ver)
	if err != nil {
		return err
	}
	stepDir = filepath.Join(dir, stepDir, stepVersion)

	// check if already exists
	if _, err := os.Stat(stepDir); err == nil {
		return nil
	}

	tmpDir, err := os.MkdirTemp(dir, "")
	if err != nil {
		return fmt.Errorf("creating temporary directory for step: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// todo: extract layers
	for _, layer := range layers {
		digest, _ := layer.Digest()

		rc, err := layer.Uncompressed()
		if err != nil {
			return fmt.Errorf("opening uncompressed reader %v: %w", digest, err)
		}
		defer rc.Close()

		tr := tar.NewReader(rc)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("tar next %v: %w", digest, err)
			}

			hdr.Name = sanitizer.SanitizePath(hdr.Name)

			if hdr.Typeflag != tar.TypeReg {
				continue
			}

			f, err := os.OpenFile(filepath.Join(tmpDir, hdr.Name), os.O_CREATE|os.O_RDWR, hdr.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("creating file: %w", err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("copying file: %w", err)
			}

			if err := f.Close(); err != nil {
				return fmt.Errorf("closing file: %w", err)
			}
		}

		rc.Close()
	}

	if err := os.MkdirAll(filepath.Dir(stepDir), 0o777); err != nil {
		return fmt.Errorf("creating step directory: %w", err)
	}

	return os.Rename(tmpDir, stepDir)
}

func is404(err error) bool {
	var terr *transport.Error
	if errors.As(err, &terr) {
		return terr.StatusCode == http.StatusNotFound
	}

	return false
}

func stepPath(dir, ver string) (string, string, error) {
	stepDir, err := module.EscapePath(dir)
	if err != nil {
		return "", "", fmt.Errorf("escaping step path: %w", err)
	}

	stepVersion, err := module.EscapeVersion(ver)
	if err != nil {
		return "", "", fmt.Errorf("escaping step version: %w", err)
	}

	return stepDir, stepVersion, nil
}

func ParseReference(addr string, opts ...name.Option) (name.Reference, error) {
	parts := strings.Split(addr, "/")
	switch {
	case len(parts) == 1:
		addr = path.Join(DefaultRegistry, DefaultNamespace, addr)
	case len(parts) > 1 && !strings.ContainsRune(parts[0], '.'):
		addr = path.Join(DefaultRegistry, addr)
	}

	ref, err := name.ParseReference(addr, opts...)
	if err != nil {
		return ref, err
	}

	return ref, nil
}
