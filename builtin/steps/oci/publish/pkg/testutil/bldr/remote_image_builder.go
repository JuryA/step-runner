package bldr

import (
	"fmt"
	"path"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/builtin/steps/oci/publish/pkg"
)

type RemoteImageBuilder struct {
	t          *testing.T
	registry   string
	repository string
	major      uint64
	minor      uint64
	patch      uint64
	release    string
}

func RemoteImageRef(t *testing.T) *RemoteImageBuilder {
	return &RemoteImageBuilder{
		t:          t,
		registry:   "localhost:5000",
		repository: "my-image",
		major:      1,
		minor:      0,
		patch:      0,
		release:    "",
	}
}

func (b *RemoteImageBuilder) WithRegistry(registry string) *RemoteImageBuilder {
	b.registry = registry
	return b
}

func (b *RemoteImageBuilder) WithRepository(repository string) *RemoteImageBuilder {
	b.registry = repository
	return b
}

func (b *RemoteImageBuilder) WithVersion(major, minor, patch uint64) *RemoteImageBuilder {
	b.major = major
	b.minor = minor
	b.patch = patch
	return b
}

func (b *RemoteImageBuilder) Build() *pkg.RemoteImageRef {
	imgRef, err := name.ParseReference(fmt.Sprintf("%s:%d.%d.%d%s", path.Join(b.registry, b.repository), b.major, b.minor, b.patch, b.release))
	require.NoError(b.t, err)

	return pkg.NewRemoteImageRef(imgRef, b.major, b.minor, b.patch, b.release)
}
