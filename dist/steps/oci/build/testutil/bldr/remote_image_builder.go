package bldr

import (
	"fmt"
	"path"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/require"
)

type RemoteImageBuilder struct {
	t          *testing.T
	registry   string
	repository string
	tag        string
}

func RemoteImageRef(t *testing.T) *RemoteImageBuilder {
	return &RemoteImageBuilder{
		t:          t,
		registry:   "localhost:5000",
		repository: "my-image",
		tag:        "1.0.0",
	}
}

func (b *RemoteImageBuilder) WithRegistry(registry string) *RemoteImageBuilder {
	b.registry = registry
	return b
}

func (b *RemoteImageBuilder) WithRepository(repository string) *RemoteImageBuilder {
	b.repository = repository
	return b
}

func (b *RemoteImageBuilder) WithTag(tag string) *RemoteImageBuilder {
	b.tag = tag
	return b
}

func (b *RemoteImageBuilder) WithRepositoryRef(imgRef name.Reference) *RemoteImageBuilder {
	b.registry = imgRef.Context().RegistryStr()
	b.repository = imgRef.Context().RepositoryStr()
	return b
}

func (b *RemoteImageBuilder) Build() name.Reference {
	imageRef, err := name.ParseReference(fmt.Sprintf("%s:%s", path.Join(b.registry, b.repository), b.tag))
	require.NoError(b.t, err)

	return imageRef
}
