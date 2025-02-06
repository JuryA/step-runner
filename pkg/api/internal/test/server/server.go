package server

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"gitlab.com/gitlab-org/step-runner/pkg/api/service"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type TestStepRunnerServer struct {
	*service.StepRunnerService
	t      *testing.T
	server *grpc.Server
	port   string
}

func New(t *testing.T, options ...func(*TestStepRunnerServer)) *TestStepRunnerServer {
	port := bldr.TCPPort(t).FindFree()

	server := &TestStepRunnerServer{
		t:      t,
		server: nil,
		port:   port,
	}

	for _, opt := range options {
		opt(server)
	}

	t.Cleanup(server.Stop)
	return server
}

func (s *TestStepRunnerServer) Serve() *TestStepRunnerServer {
	stepCache, err := cache.New()
	require.NoError(s.t, err)

	s.server = grpc.NewServer()
	s.StepRunnerService = service.New(stepCache, runner.NewEmptyEnvironment())
	proto.RegisterStepRunnerServer(s.server, s.StepRunnerService)

	listener, _ := bldr.TCPPort(s.t).Listen(s.port)

	go func() {
		err := s.server.Serve(listener)
		require.NoError(s.t, err)
	}()

	return s
}

func (s *TestStepRunnerServer) NewConnection() *grpc.ClientConn {
	conn, err := grpc.Dial("localhost:"+s.port, grpc.WithTransportCredentials(insecure.NewCredentials())) //nolint:all
	require.NoError(s.t, err)

	s.t.Cleanup(func() { _ = conn.Close() })

	return conn
}

func (s *TestStepRunnerServer) Stop() {
	if s.server != nil {
		s.server.Stop()
		s.server = nil
	}
}
