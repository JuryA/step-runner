package oci

import (
	"os"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci/internal/client"
)

func Login(registry, username, password string) (string, error) {
	if registry == "" {
		registry = client.DefaultRegistry
	}

	cf, err := config.Load(os.Getenv("DOCKER_CONFIG"))
	if err != nil {
		return "", err
	}

	creds := cf.GetCredentialsStore(registry)
	if registry == name.DefaultRegistry {
		registry = authn.DefaultAuthKey
	}

	if err := creds.Store(types.AuthConfig{
		ServerAddress: registry,
		Username:      username,
		Password:      password,
	}); err != nil {
		return "", err
	}

	if err := cf.Save(); err != nil {
		return "", err
	}

	return cf.Filename, nil
}
