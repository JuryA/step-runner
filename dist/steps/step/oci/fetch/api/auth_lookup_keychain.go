package api

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
)

type LookupValue func(key string) (string, bool)

const dockerEnvConfigVar = "DOCKER_AUTH_CONFIG"

type AuthLookupKeychain struct {
	lookup LookupValue
}

func NewAuthLookupKeychain(lookup LookupValue) *AuthLookupKeychain {
	return &AuthLookupKeychain{
		lookup: lookup,
	}
}

func (ak *AuthLookupKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	authConfig, found := ak.lookup(dockerEnvConfigVar)

	if !found {
		slog.Debug("auth environment variable not found, using anonymous authentication", "env_variable", dockerEnvConfigVar)
		return authn.Anonymous, nil
	}

	if authConfig == "" {
		slog.Debug("auth environment variable is empty, using anonymous authentication", "env_variable", dockerEnvConfigVar)
		return authn.Anonymous, nil
	}

	configFile, err := config.LoadFromReader(strings.NewReader(authConfig))
	if err != nil {
		return nil, fmt.Errorf("resolving OCI registry authentication: %w", err)
	}

	authCfg, err := ak.authConfig(target, configFile)
	if err != nil {
		return nil, fmt.Errorf("resolving OCI registry authentication: %w", err)
	}

	if authCfg == nil {
		slog.Debug("auth environment variable does not contain auth for target, using anonymous authentication", "env_variable", dockerEnvConfigVar, "target", target.RegistryStr())
		return authn.Anonymous, nil
	}

	slog.Debug("auth config found, publishing image using credentials")
	return authn.FromConfig(*authCfg), nil
}

func (ak *AuthLookupKeychain) authConfig(target authn.Resource, configFile *configfile.ConfigFile) (*authn.AuthConfig, error) {
	var empty types.AuthConfig

	for _, key := range []string{target.String(), target.RegistryStr()} {
		cfg, err := configFile.GetAuthConfig(key)
		if err != nil {
			return nil, fmt.Errorf("getting auth config for %s: %w", key, err)
		}

		if cfg != empty {
			return &authn.AuthConfig{
				Username:      cfg.Username,
				Password:      cfg.Password,
				Auth:          cfg.Auth,
				IdentityToken: cfg.IdentityToken,
				RegistryToken: cfg.RegistryToken,
			}, nil
		}
	}

	return nil, nil
}
