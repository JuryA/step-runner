package basic

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"

	"gitlab.com/gitlab-org/step-runner/pkg/api/client"
	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test/server"
)

func Test_StepRunnerClient_Status_ListJobs(t *testing.T) {
	rr1 := test.RunRequest(t, `run:
  - name: hello_world
    step: ../../../runner/test_steps/greeting
    inputs: {}
`, nil, nil)
	rr1.Id = rr1.Id + "-1"

	rr2 := test.RunRequest(t, `run:
  - name: blabla
    step: ../../testdata/bash
    inputs:
        script: echo "bla bla bla"
`, nil, nil)
	rr1.Id = rr1.Id + "-2"

	server := server.New(t).Serve()
	srClient := New(server.NewConnection())

	ctx := context.Background()
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

const (
	runStepMsg = "Running step \"lorem\"\n"
	lorem      = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`
)

func Test_StepRunnerClient_FollowLogs_Success(t *testing.T) {
	rr := test.RunRequest(t, `run:
  - name: lorem
    step: ../../testdata/bash
    inputs:
        script: echo "`+lorem+`"`, nil, nil)

	server := server.New(t).Serve()
	srClient := New(server.NewConnection())

	ctx := context.Background()
	assert.NoError(t, srClient.Run(ctx, rr))

	buf := bytes.Buffer{}
	n, err := srClient.FollowLogs(ctx, rr.Id, 0, &buf)

	assert.NoError(t, err)
	assert.Equal(t, int64(len(runStepMsg)+len(lorem)+1), n)
	assert.Equal(t, runStepMsg+lorem+"\n", buf.String())
	assert.NoError(t, srClient.Close(ctx, rr.Id))
}

type toWriter func([]byte) (int, error)

func (t toWriter) Write(p []byte) (int, error) { return t(p) }

func Test_StepRunnerClient_FollowLogs_Again(t *testing.T) {
	rr := test.RunRequest(t, `run:
  - name: lorem
    step: ../../testdata/bash
    inputs:
        script: echo "`+lorem+`"`, nil, nil)

	server := server.New(t).Serve()
	srClient := New(server.NewConnection())

	ctx := context.Background()
	assert.NoError(t, srClient.Run(ctx, rr))

	buf := bytes.Buffer{}
	bytesToWrite := 20
	writerWithErr := func(p []byte) (int, error) {
		if len(p) < bytesToWrite {
			bytesToWrite = len(p)
		}

		buf.Write(p[:bytesToWrite])
		return bytesToWrite, errors.New("pow")
	}

	n, err := srClient.FollowLogs(ctx, rr.Id, 0, toWriter(writerWithErr))

	assert.Error(t, err)
	assert.Equal(t, int64(bytesToWrite), n)
	assert.Equal(t, runStepMsg[:bytesToWrite], buf.String())

	n, err = srClient.FollowLogs(ctx, rr.Id, n, &buf)

	assert.NoError(t, err)
	assert.Equal(t, int64(bytesToWrite+len(lorem)+1), n)
	assert.Equal(t, runStepMsg[:bytesToWrite]+lorem+"\n", buf.String())
	assert.NoError(t, srClient.Close(ctx, rr.Id))
}

func Test_StepRunnerClient_WaitForReady(t *testing.T) {
	srvr := server.New(t)
	t.Run("aborts waiting for ready when deadline has exceeded", func(t *testing.T) {
		conn := srvr.NewConnection()
		require.Eventually(t, func() bool { return conn.GetState() == connectivity.TransientFailure }, 2*time.Second, 100*time.Millisecond)

		step := "run:\n  - name: hello_world\n    step: ../../../runner/test_steps/greeting"
		runRequest := test.RunRequest(t, step, nil, nil)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()

		srClient := New(conn)
		err := srClient.Run(ctx, runRequest)
		require.Error(t, err)
		require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	})

	t.Run("blocks on send until server is ready", func(t *testing.T) {
		conn := srvr.NewConnection()
		require.Eventually(t, func() bool { return conn.GetState() == connectivity.TransientFailure }, 2*time.Second, 100*time.Millisecond)

		srvr.Serve()
		step := "run:\n  - name: hello_world\n    step: ../../../runner/test_steps/greeting"
		runRequest := test.RunRequest(t, step, nil, nil)

		srClient := New(conn)
		require.NoError(t, srClient.Run(context.Background(), runRequest))
	})
}
