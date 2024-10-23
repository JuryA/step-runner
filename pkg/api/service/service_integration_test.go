//go:build integration

package service_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
	"gitlab.com/gitlab-org/step-runner/pkg/api/service"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

const (
	actionStep = `spec: {}
---
steps:
  - name: action
    action: "mikefarah/yq@master"
    inputs:
      cmd: echo ["foo again!"] | yq .[0]
`
)

const bufSize = 1024 * 1024

func must(e error) {
	if e != nil {
		panic(e)
	}
}

var (
	conn         *grpc.ClientConn
	stepsService *service.StepRunnerService
	apiClient    proto.StepRunnerClient
)

func envWithDefault(key string, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	stepCache, err := cache.New()
	must(err)

	stepsService = service.New(stepCache, runner.NewEmptyEnvironment())

	buflis := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	proto.RegisterStepRunnerServer(server, stepsService)
	go func() { must(server.Serve(buflis)) }()
	defer func() { server.GracefulStop() }()

	bufDialer := func(context.Context, string) (net.Conn, error) { return buflis.Dial() }
	conn, err = grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	must(err)
	defer func() { conn.Close() }()

	apiClient = proto.NewStepRunnerClient(conn)

	code := m.Run()
	os.Exit(code)
}

func cleanup(t *testing.T, paths ...string) {
	os.RemoveAll(path.Join(test.WorkDir(t), ".config"))
	os.RemoveAll(path.Join(test.WorkDir(t), ".cache"))

	for _, p := range paths {
		os.RemoveAll(path.Join(test.WorkDir(t), p))
	}
}

func Test_StepRunnerService_Run_Action_Success(t *testing.T) {
	defer cleanup(t, ".env-file")

	rr := test.ProtoRunRequest(t, actionStep, true)

	dockerHost := envWithDefault("DOCKER_HOST", "unix:///var/run/docker.sock")
	dockerTLSCertDir := envWithDefault("DOCKER_TLS_CERTDIR", "")
	ciProjDir := envWithDefault("CI_PROJECT_DIR", rr.WorkDir)

	rr.Job.Variables = []*proto.Variable{
		{
			Key:   "CI_PROJECT_DIR",
			Value: ciProjDir,
		},
		{
			Key:   "DOCKER_HOST",
			Value: dockerHost,
		},
		{
			Key:   "DOCKER_TLS_CERTDIR",
			Value: dockerTLSCertDir,
		},
	}

	// Add an env var with newlines. I saw this fail at least once during testing.
	rr.Env["CI_COMMIT_MESSAGE"] = `jkdhfgkljhlksjdfg

jksdfgkssdfgsdf
sdfgsdfgsdfg`

	runRequest(t, rr, time.Second*30, proto.StepResult_success)
}

func runRequest(t *testing.T, rr *proto.RunRequest, timeout time.Duration, wantStatus proto.StepResult_Status) {
	ctx := context.Background()

	_, err := apiClient.Run(ctx, rr)
	require.NoError(t, err)

	defer apiClient.Close(ctx, &proto.CloseRequest{Id: rr.Id})

	assert.Eventually(t, func() bool {
		res, err := apiClient.Status(ctx, &proto.StatusRequest{Id: rr.Id})
		assert.NoError(t, err)
		return res.Jobs[0].Status == wantStatus
	}, timeout, time.Millisecond*100)

	// if the test was run with -v, capture and print the logs
	if testing.Verbose() {
		t.Log(string(captureLogs(t, ctx, rr.Id)))
	}

	res, err := apiClient.Status(ctx, &proto.StatusRequest{Id: rr.Id})
	assert.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, wantStatus, res.Jobs[0].Status)
}

func captureLogs(t *testing.T, ctx context.Context, id string) []byte {
	stream, err := apiClient.FollowLogs(ctx, &proto.FollowLogsRequest{Id: id})
	require.NoError(t, err)

	logs := bytes.Buffer{}

	for {
		p, ierr := stream.Recv()
		if ierr == io.EOF {
			err = ierr
			break
		}
		logs.Write(p.Data)
		require.NoError(t, err)
	}

	return logs.Bytes()
}
