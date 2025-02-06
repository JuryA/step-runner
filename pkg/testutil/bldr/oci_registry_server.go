package bldr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"github.com/stretchr/testify/require"
)

func StartOCIRegistryServer(t *testing.T) string {
	port := TCPPort(t).FindFree()

	ociRegistry := NewOCIRegistryServer(t, port)
	ociRegistry.Serve(context.Background())
	t.Cleanup(ociRegistry.Stop)

	return fmt.Sprintf("127.0.0.1:%s", port)
}

type OCIRegistryServer struct {
	server   *registry.Registry
	cancelFn context.CancelFunc
	port     string
	t        *testing.T
}

func NewOCIRegistryServer(t *testing.T, port string) *OCIRegistryServer {
	return &OCIRegistryServer{
		t:    t,
		port: port,
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
	config.HTTP.Addr = fmt.Sprintf(":%s", s.port)

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

func (s *OCIRegistryServer) Stop() {
	s.cancelFn()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.server.Shutdown(ctx)
	require.NoError(s.t, err)
}
