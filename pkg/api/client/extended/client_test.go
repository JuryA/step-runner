package extended

import (
	"context"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"gitlab.com/gitlab-org/step-runner/pkg/api/client"
	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test/server"
)

type testDialer struct {
	dial func() *grpc.ClientConn
}

func (t *testDialer) Dial() (*grpc.ClientConn, error) { return t.dial(), nil }

func cleanup(t *testing.T, paths ...string) {
	os.RemoveAll(path.Join(test.WorkDir(t), ".config"))
	os.RemoveAll(path.Join(test.WorkDir(t), ".cache"))

	for _, p := range paths {
		os.RemoveAll(path.Join(test.WorkDir(t), p))
	}
}

func Test_StepRunnerClient_RunAndFollow_Success(t *testing.T) {
	defer cleanup(t)

	server := server.New(t).Serve()
	srClient, err := New(&testDialer{dial: server.NewConnection})
	require.NoError(t, err)
	//nolint:errcheck
	defer srClient.CloseConn()

	rr := test.RunRequest(t, `steps:
  - name: hello_world
    step: ../../../runner/test_steps/greeting
    inputs: {}
  - name: blabla
    step: ../../testdata/bash
    inputs:
        script: echo "bla bla bla $FOO"
  - name: env
    step: ../../testdata/bash
    inputs:
        script: env
  - name: file_var
    step: ../../testdata/bash
    inputs:
        script: echo ${{ job.MEGA }} && cat ${{ job.MEGA }}
`,
		map[string]string{
			"FOO": "bar",
			"BAZ": "blammo",
		},
		[]client.Variable{
			{
				Key:   "MEGA",
				Value: "this is some secret garbage",
				File:  true,
			},
		},
	)

	ctx := context.Background()

	logs := test.ClosableBuf{}
	out := FollowOutput{Logs: &logs}
	status, err := srClient.RunAndFollow(ctx, rr, &out)

	assert.NoError(t, err)
	assert.Equal(t, client.StateSuccess, status.State)
	assert.Empty(t, status.Message)
	assert.Contains(t, logs.String(), "meet steppy who is 1 likes {\"color\":\"red\"} and is hungry false")
	assert.Contains(t, logs.String(), "bla bla bla bar")
	assert.Contains(t, logs.String(), "FOO=bar")
	assert.Contains(t, logs.String(), "BAZ=blammo")
	assert.Contains(t, logs.String(), path.Join(os.TempDir(), "step-runner-output-"+rr.Id, "MEGA"))
	assert.Contains(t, logs.String(), "this is some secret garbage")
}

func Test_StepRunnerClient_RunAndFollow_Cancelled(t *testing.T) {
	defer cleanup(t)

	server := server.New(t).Serve()
	srClient, err := New(&testDialer{dial: server.NewConnection})
	require.NoError(t, err)
	//nolint:errcheck
	defer srClient.CloseConn()

	rr := test.RunRequest(t, `steps:
  - name: bash
    step: ../../testdata/bash
    inputs:
        script: echo "hello there" && sleep 5 && echo "goodbye"
`, nil, nil)

	// NOTE: this cancels the client side, which ultimately calls Close() on the job and terminates it on the server
	// side.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	logs := test.ClosableBuf{}
	out := FollowOutput{Logs: &logs}
	status, err := srClient.RunAndFollow(ctx, rr, &out)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	// TODO: This is weird: we'd expect the status to be either failed or cancelled, but job timeout is not implemented
	// on the server side yet, and Close() is necessarily called after Status().
	assert.Equal(t, client.StateRunning, status.State)
	assert.Empty(t, status.Message)
	assert.Contains(t, logs.String(), "hello there")
	assert.NotContains(t, logs.String(), "goodbye")
}

func Test_StepRunnerClient_RunAndFollow_Step_Fails(t *testing.T) {
	defer cleanup(t)

	server := server.New(t).Serve()
	srClient, err := New(&testDialer{dial: server.NewConnection})
	require.NoError(t, err)
	//nolint:errcheck
	defer srClient.CloseConn()

	rr := test.RunRequest(t, `steps:
  - name: bash
    step: ../../testdata/bash
    inputs:
        script: kjhdfdhlkf
`, nil, nil)

	ctx := context.Background()

	logs := test.ClosableBuf{}
	out := FollowOutput{Logs: &logs}
	status, err := srClient.RunAndFollow(ctx, rr, &out)

	assert.NoError(t, err)
	assert.Equal(t, client.StateFailure, status.State)
	assert.Equal(t, `step "bash": exec: exit status 127`, status.Message)
	assert.Contains(t, logs.String(), "kjhdfdhlkf: command not found")
}

func Test_StepRunnerClient_RunAndFollow_Concurrent(t *testing.T) {
	defer cleanup(t)

	ctx := context.Background()

	server := server.New(t).Serve()
	srClient, err := New(&testDialer{dial: server.NewConnection})
	require.NoError(t, err)
	//nolint:errcheck
	defer srClient.CloseConn()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		rr := test.RunRequest(t, `steps:
  - name: hello_world
    step: ../../../runner/test_steps/greeting
    inputs: {}
`, nil, nil)
		rr.Id = rr.Id + "-1"

		logs := test.ClosableBuf{}
		out := FollowOutput{Logs: &logs}
		status, err := srClient.RunAndFollow(ctx, rr, &out)

		assert.NoError(t, err)
		assert.Equal(t, client.StateSuccess, status.State)
		assert.Empty(t, status.Message)
		assert.Contains(t, logs.String(), "meet steppy who is 1 likes {\"color\":\"red\"} and is hungry false")
		assert.NotContains(t, logs.String(), "FOO=bar")
		assert.NotContains(t, logs.String(), "BAZ=blammo")
	}()

	go func() {
		defer wg.Done()
		rr := test.RunRequest(t, `steps:
  - name: bash
    step: ../../testdata/bash
    inputs:
        script: env
`,
			map[string]string{
				"FOO": "bar",
				"BAZ": "blammo",
			}, nil)
		rr.Id = rr.Id + "-2"

		logs := test.ClosableBuf{}
		out := FollowOutput{Logs: &logs}
		status, err := srClient.RunAndFollow(ctx, rr, &out)

		assert.NoError(t, err)
		assert.Equal(t, client.StateSuccess, status.State)
		assert.Empty(t, status.Message)
		assert.Contains(t, logs.String(), "FOO=bar")
		assert.Contains(t, logs.String(), "BAZ=blammo")
		assert.NotContains(t, logs.String(), "meet steppy who is 1 likes {\"color\":\"red\"} and is hungry false")
	}()

	wg.Wait()
}

func Test_StepRunnerClient_RunAndFollow_LogsOnly(t *testing.T) {
	defer cleanup(t)

	server := server.New(t).Serve()
	srClient, err := New(&testDialer{dial: server.NewConnection})
	require.NoError(t, err)
	//nolint:errcheck
	defer srClient.CloseConn()

	rr := test.RunRequest(t, `steps:
  - name: blabla
    step: ../../testdata/bash
    inputs:
        script: echo "bla bla bla $FOO"
  - name: env
    step: ../../testdata/bash
    inputs:
        script: env
`,
		map[string]string{
			"FOO": "bar",
			"BAZ": "blammo",
		}, nil)

	ctx := context.Background()

	logs := test.ClosableBuf{}
	out := FollowOutput{Logs: &logs}
	status, err := srClient.RunAndFollow(ctx, rr, &out)

	assert.NoError(t, err)
	assert.Equal(t, client.StateSuccess, status.State)
	assert.Empty(t, status.Message)
	assert.Contains(t, logs.String(), "bla bla bla bar")
	assert.Contains(t, logs.String(), "FOO=bar")
	assert.Contains(t, logs.String(), "BAZ=blammo")
}
