package server

import (
	"testing"
	"time"

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
	t                  *testing.T
	server             *grpc.Server
	port               string
	executor           func(delegate func())
	jobRunExitWaitTime time.Duration
}

func New(t *testing.T, options ...func(*TestStepRunnerServer)) *TestStepRunnerServer {
	port := bldr.TCPPort(t).FindFree()

	server := &TestStepRunnerServer{
		t:                  t,
		server:             nil,
		port:               port,
		executor:           func(delegate func()) { delegate() }, // sync executor (does not start a goroutine)
		jobRunExitWaitTime: 500 * time.Millisecond,
	}

	for _, opt := range options {
		opt(server)
	}

	t.Cleanup(server.Stop)
	return server
}

func (s *TestStepRunnerServer) Serve() *TestStepRunnerServer {
	svcOptions := []func(*service.StepRunnerService){
		service.WithExecutor(s.executor),
		service.WithJobRunExitWaitTime(s.jobRunExitWaitTime),
	}

	stepCache, err := cache.New()
	require.NoError(s.t, err)

	s.server = grpc.NewServer()
	s.StepRunnerService = service.New(stepCache, runner.NewEmptyEnvironment(), svcOptions...)
	proto.RegisterStepRunnerServer(s.server, s.StepRunnerService)

	listener, _ := bldr.TCPPort(s.t).Listen(s.port)

	go func() {
		err := s.server.Serve(listener)
		require.NoError(s.t, err)
	}()

	return s
}

func (s *TestStepRunnerServer) NewConnection() *grpc.ClientConn {
	conn, err := grpc.NewClient("localhost:"+s.port, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func WithExecutor(executor func(delegate func())) func(*TestStepRunnerServer) {
	return func(server *TestStepRunnerServer) {
		server.executor = executor
	}
}

func WithJobRunExitWaitTime(waitTime time.Duration) func(*TestStepRunnerServer) {
	return func(server *TestStepRunnerServer) {
		server.jobRunExitWaitTime = waitTime
	}
}
