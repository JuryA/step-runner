package basic

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"gitlab.com/gitlab-org/step-runner/pkg/api/client"
	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
	"gitlab.com/gitlab-org/step-runner/pkg/api/service"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/proto"
)

const bufSize = 1024 * 1024

func must(e error) {
	if e != nil {
		panic(e)
	}
}

var conn *grpc.ClientConn

func TestMain(m *testing.M) {
	ctx := context.Background()

	stepCache, err := cache.New()
	must(err)

	stepsService := service.New(stepCache)

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

	code := m.Run()
	os.Exit(code)
}

func Test_StepRunnerClient_Status_ListJobs(t *testing.T) {
	ctx := context.Background()
	srClient := New(conn)

	rr1 := test.RunRequest(t, `steps:
  - name: hello_world
    step: ../../../runner/test_steps/greeting
    inputs: {}
`, nil, nil)
	rr1.Id = rr1.Id + "-1"

	rr2 := test.RunRequest(t, `steps:
  - name: blabla
    step: ../../testdata/bash
    inputs:
        script: echo "bla bla bla"
`, nil, nil)
	rr1.Id = rr1.Id + "-2"

	assert.NoError(t, srClient.Run(ctx, rr1))
	assert.NoError(t, srClient.Run(ctx, rr2))

	jobs, err := srClient.ListJobs(ctx)
	assert.NoError(t, err)

	assert.Len(t, jobs, 2)
	for _, j := range jobs {
		assert.True(t, j.Id == rr1.Id || j.Id == rr2.Id)
		assert.True(t, j.State == client.StateRunning || j.State == client.StateSuccess)
		assert.Empty(t, j.Message)
		assert.True(t, j.StartTime.Before(time.Now()))
		assert.True(t, j.EndTime.IsZero() || j.EndTime.After(j.StartTime))
	}

	job, err := srClient.Status(ctx, rr1.Id)
	assert.NoError(t, err)

	// get the corresponding job from the ListJobs response
	j := jobs[0]
	if j.Id != job.Id {
		j = jobs[1]
	}
	assert.Equal(t, job, j)
	assert.NoError(t, srClient.Close(ctx, rr1.Id))
	assert.NoError(t, srClient.Close(ctx, rr2.Id))
}

const lorem = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

func Test_StepRunnerClient_FollowSteps_Success(t *testing.T) {
	ctx := context.Background()
	srClient := New(conn)

	rr := test.RunRequest(t, `steps:
  - name: lorem
    step: ../../testdata/bash
    inputs:
        script: echo "`+lorem+`"
`, nil, nil)

	assert.NoError(t, srClient.Run(ctx, rr))

	stepResultWriteCloser := test.StepResultWriter{}

	n, err := srClient.FollowSteps(ctx, rr.Id, 0, &stepResultWriteCloser)

	assert.Eventually(t, func() bool { return len(stepResultWriteCloser) > 0 }, time.Millisecond*500, time.Millisecond*100)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)
	assert.NoError(t, srClient.Close(ctx, rr.Id))
}

func Test_StepRunnerClient_FollowLogs_Success(t *testing.T) {
	ctx := context.Background()
	srClient := New(conn)

	rr := test.RunRequest(t, `steps:
  - name: lorem
    step: ../../testdata/bash
    inputs:
        script: echo "`+lorem+`"`, nil, nil)

	assert.NoError(t, srClient.Run(ctx, rr))

	buf := bytes.Buffer{}
	n, err := srClient.FollowLogs(ctx, rr.Id, 0, &buf)

	assert.NoError(t, err)
	assert.Equal(t, int64(len(lorem)+1), n)
	assert.Equal(t, lorem+"\n", buf.String())
	assert.NoError(t, srClient.Close(ctx, rr.Id))
}

type toWriter func([]byte) (int, error)

func (t toWriter) Write(p []byte) (int, error) { return t(p) }

func Test_StepRunnerClient_FollowLogs_Again(t *testing.T) {
	ctx := context.Background()
	srClient := New(conn)

	rr := test.RunRequest(t, `steps:
  - name: lorem
    step: ../../testdata/bash
    inputs:
        script: echo "`+lorem+`"`, nil, nil)

	assert.NoError(t, srClient.Run(ctx, rr))

	buf := bytes.Buffer{}
	bytesToWrite := 66
	writerWithErr := func(p []byte) (int, error) {
		buf.Write(p[:bytesToWrite])
		return bytesToWrite, errors.New("pow")
	}

	n, err := srClient.FollowLogs(ctx, rr.Id, 0, toWriter(writerWithErr))

	assert.Error(t, err)
	assert.Equal(t, int64(bytesToWrite), n)
	assert.Equal(t, lorem[:bytesToWrite], buf.String())

	n, err = srClient.FollowLogs(ctx, rr.Id, n, &buf)

	assert.NoError(t, err)
	assert.Equal(t, int64(len(lorem)+1), n)
	assert.Equal(t, lorem+"\n", buf.String())
	assert.NoError(t, srClient.Close(ctx, rr.Id))
}
