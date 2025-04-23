package internal

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/require"
)

func TestAuthKeyChain_Resolve(t *testing.T) {
	t.Run("anon when config not found", func(t *testing.T) {
		kc := NewAuthLookupKeychain(func(string) (string, bool) { return "", false })

		authenticator, err := kc.Resolve(name.MustParseReference("registry.gitlab.com/image:1.0.0").Context())
		require.NoError(t, err)
		require.Equal(t, authn.Anonymous, authenticator)
	})

	t.Run("anon when config is empty", func(t *testing.T) {
		kc := NewAuthLookupKeychain(func(string) (string, bool) { return "", true })

		authenticator, err := kc.Resolve(name.MustParseReference("registry.gitlab.com/image:1.0.0").Context())
		require.NoError(t, err)
		require.Equal(t, authn.Anonymous, authenticator)
	})

	t.Run("anon when config does not define target auth", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("foo:bar"))
		configData := fmt.Sprintf(`{"auths":{"registry.labgit.com":{"auth":"%s"}}}`, encoded)
		kc := NewAuthLookupKeychain(func(string) (string, bool) { return configData, true })

		authenticator, err := kc.Resolve(name.MustParseReference("registry.gitlab.com/image:1.0.0").Context())
		require.NoError(t, err)
		require.Equal(t, authn.Anonymous, authenticator)
	})

	t.Run("returns auth when config defines target registry", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("foo:bar"))
		configData := fmt.Sprintf(`{"auths":{"registry.gitlab.com":{"auth":"%s"}}}`, encoded)
		kc := NewAuthLookupKeychain(func(string) (string, bool) { return configData, true })

		authenticator, err := kc.Resolve(name.MustParseReference("registry.gitlab.com/image:1.0.0").Context())
		require.NoError(t, err)

		auth, err := authenticator.Authorization()
		require.NoError(t, err)
		require.Equal(t, "foo", auth.Username)
		require.Equal(t, "bar", auth.Password)
	})

	t.Run("returns auth when config defines target repository", func(t *testing.T) {
		encodedA := base64.StdEncoding.EncodeToString([]byte("user_a:passwd"))
		encodedB := base64.StdEncoding.EncodeToString([]byte("user_b:passwd"))

		cfgData := `{"auths":{
			"registry.gitlab.com":{"auth":"%s"},
			"registry.gitlab.com/image":{"auth":"%s"}
		}}`
		configData := fmt.Sprintf(cfgData, encodedA, encodedB)
		kc := NewAuthLookupKeychain(func(string) (string, bool) { return configData, true })

		authenticator, err := kc.Resolve(name.MustParseReference("registry.gitlab.com/image:1.0.0").Context())
		require.NoError(t, err)

		auth, err := authenticator.Authorization()
		require.NoError(t, err)
		require.Equal(t, "user_b", auth.Username)
	})

	t.Run("supports authenticating to DockerHub", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("user:passwd"))
		configData := fmt.Sprintf(`{"auths":{"https://index.docker.io/v1/":{"auth":"%s"}}}`, encoded)
		kc := NewAuthLookupKeychain(func(string) (string, bool) { return configData, true })

		authenticator, err := kc.Resolve(name.MustParseReference("index.docker.io/image:1.0.0").Context())
		require.NoError(t, err)

		auth, err := authenticator.Authorization()
		require.NoError(t, err)
		require.Equal(t, "user", auth.Username)
	})
}
