package bldr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	"github.com/distribution/distribution/v3/registry/auth"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
)

type OCIRegistryServerOptions struct {
	port        string
	requireAuth bool
	user        string
	password    string
}

func StartOCIRegistryServer(t *testing.T, options ...func(*OCIRegistryServerOptions)) *OCIRegistryServer {
	port := TCPPort(t).FindFree()

	opts := &OCIRegistryServerOptions{
		requireAuth: false,
		port:        port,
	}

	for _, option := range options {
		option(opts)
	}

	ociRegistry := NewOCIRegistryServer(t, opts)
	ociRegistry.Serve(context.Background())
	t.Cleanup(ociRegistry.Stop)

	return ociRegistry
}

func WithRequireAuth(user, password string) func(*OCIRegistryServerOptions) {
	return func(opts *OCIRegistryServerOptions) {
		opts.requireAuth = true
		opts.user = user
		opts.password = password
	}
}

type OCIRegistryServer struct {
	server   *registry.Registry
	cancelFn context.CancelFunc
	opts     *OCIRegistryServerOptions
	t        *testing.T
}

func NewOCIRegistryServer(t *testing.T, opts *OCIRegistryServerOptions) *OCIRegistryServer {
	return &OCIRegistryServer{
		t:    t,
		opts: opts,
	}
}

func (s *OCIRegistryServer) Serve(ctx context.Context) {
	ctx, cancelFn := context.WithCancel(ctx)
	s.cancelFn = cancelFn

	config := &configuration.Configuration{}
	config.Storage = configuration.Storage{}
	config.Storage["inmemory"] = configuration.Parameters{}
	config.Storage["maintenance"] = configuration.Parameters{"uploadpurging": map[any]any{"enabled": false}}
	config.Log.Level = "debug"
	config.Log.Formatter = "text"
	config.HTTP.Secret = "secrety-secret"
	config.HTTP.Addr = fmt.Sprintf(":%s", s.opts.port)

	if s.opts.requireAuth {
		err := auth.Register("basicauth", NewOCIBasicAuthAccess)
		require.NoError(s.t, err)

		config.Auth = configuration.Auth{"basicauth": configuration.Parameters{"username": s.opts.user, "password": s.opts.password}}
	}

	var err error
	s.server, err = registry.NewRegistry(ctx, config)
	require.NoError(s.t, err)

	var errchan chan error
	go func() {
		errchan <- s.server.ListenAndServe()
	}()

	select {
	case err = <-errchan:
		s.t.Fatalf("error serving registry: %v", err)
	default:
	}
}

func (s *OCIRegistryServer) Address() string {
	return fmt.Sprintf("127.0.0.1:%s", s.opts.port)
}

func (s *OCIRegistryServer) RefToImage(imageName, imageTag string) name.Reference {
	remoteImgRef, err := name.ParseReference(fmt.Sprintf("%s/%s:%s", s.Address(), imageName, imageTag))
	require.NoError(s.t, err)

	return remoteImgRef
}

func (s *OCIRegistryServer) Push(remoteImgRef name.Reference, img v1.Image) {
	err := remote.Write(remoteImgRef, img)
	require.NoError(s.t, err)
}

func (s *OCIRegistryServer) PushImageIndex(remoteImgRef name.Reference, img v1.ImageIndex) {
	err := remote.WriteIndex(remoteImgRef, img)
	require.NoError(s.t, err)
}

func (s *OCIRegistryServer) Stop() {
	s.cancelFn()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.server.Shutdown(ctx)
}
