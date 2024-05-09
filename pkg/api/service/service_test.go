package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/pkg/jobs"
	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	id        = "853"
	helloStep = `spec: {}
---
steps:
  - name: hello_world
    step: ../../runner/test_steps/greeting
    inputs: {}
`
	bashStep = `spec: {}
---
steps:
  - name: bash
    step: ./testdata/bash
    inputs:
        script: %s
`
)

func makeBashStep(cmd string) string {
	return fmt.Sprintf(bashStep, cmd)
}

func testDirName(t *testing.T) string {
	return path.Join(os.TempDir(), t.Name())
}

func makeRunRequest(t *testing.T, step string, withJob bool) *proto.RunRequest {
	testDir := testDirName(t)
	runReq := proto.RunRequest{
		Id:    id,
		Steps: step,
		Env:   map[string]string{},
	}

	if withJob {
		runReq.Job = &proto.Job{BuildDir: testDir}
	} else {
		runReq.WorkDir = testDir
	}

	return &runReq
}

const bufSize = 1024 * 1024

func startService(t *testing.T) (*StepRunnerService, proto.StepRunnerClient, func()) {
	srs, err := New()
	require.NoError(t, err)

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	proto.RegisterStepRunnerServer(srv, srs)

	bufDialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }

	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	go func() { require.NoError(t, srv.Serve(lis)) }()

	cleanup := func() {
		srv.GracefulStop()
		conn.Close()
	}

	return srs, proto.NewStepRunnerClient(conn), cleanup
}

func Test_StepRunnerService_Run_Success(t *testing.T) {
	bg := context.Background()
	srs, client, cleanup := startService(t)
	defer cleanup()

	rr := makeRunRequest(t, helloStep, false)

	_, err := client.Run(bg, rr)
	require.NoError(t, err)

	job, ok := srs.jobs.Get(id)
	require.True(t, ok)
	defer os.RemoveAll(job.WorkDir)

	assert.Eventually(t, job.Finished, time.Second*20, time.Millisecond*50)
	assert.NoError(t, job.Ctx.Err())

	res, err := job.Result()
	assert.Nil(t, err)
	require.NotNil(t, res)

	assert.Equal(t, int32(0), res.ExitCode)

	job.Close()
	assert.NoDirExists(t, job.TmpDir)
}

func Test_StepRunnerService_Run_Cancelled(t *testing.T) {
	defer os.RemoveAll(testDirName(t))

	tests := map[string]struct {
		id       string
		script   string
		finish   func(*jobs.Job, *StepRunnerService, *sync.WaitGroup)
		validate func(*jobs.Job)
	}{
		"Close called before request executed": {
			script: "sleep 1",
			finish: func(j *jobs.Job, srs *StepRunnerService, wg *sync.WaitGroup) {
				defer wg.Done()
				j.Close()
			},
			validate: func(j *jobs.Job) {
				res, err := j.Result()
				assert.True(t, errors.Is(err, context.Canceled))
				assert.Nil(t, res)
			},
		},
		"Close called after request finished": {
			script: "sleep 1",
			finish: func(j *jobs.Job, srs *StepRunnerService, wg *sync.WaitGroup) {
				defer wg.Done()
				// Make sure the step execution finished
				assert.Eventually(t, j.Finished, time.Millisecond*1900, time.Millisecond*100)
				j.Close()
			},
			validate: func(j *jobs.Job) {
				res, err := j.Result()
				assert.NoError(t, err)
				assert.NotNil(t, res)
			},
		},
		"Close called before request finishes": {
			script: "sleep 1",
			finish: func(j *jobs.Job, srs *StepRunnerService, wg *sync.WaitGroup) {
				defer wg.Done()
				// Make sure the step sub-process was executed before calling Finish()
				time.Sleep(time.Millisecond * 50)
				j.Close()
			},
			validate: func(j *jobs.Job) {
				res, err := j.Result()
				assert.True(t, errors.Is(err, context.Canceled))
				assert.Nil(t, res)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			bg := context.Background()
			srs, client, cleanup := startService(t)
			defer cleanup()

			wg := sync.WaitGroup{}
			wg.Add(1)

			_, err := client.Run(bg, makeRunRequest(t, makeBashStep(tt.script), false))
			require.NoError(t, err)

			job, ok := srs.jobs.Get(id)
			require.True(t, ok)
			defer os.RemoveAll(job.WorkDir)

			go tt.finish(job, srs, &wg)

			assert.Eventually(t, job.Finished, time.Millisecond*5500, time.Millisecond*100)
			wg.Wait()

			assert.Error(t, job.Ctx.Err())
			tt.validate(job)

			assert.NoDirExists(t, job.TmpDir)
		})
	}
}

func Test_StepRunnerService_Run_Vars(t *testing.T) {
	defer os.RemoveAll(testDirName(t))

	tests := map[string]struct {
		jobWorkDir bool
		id         string
		script     string
		setup      func(*proto.RunRequest)
	}{
		"env vars": {
			id:     "111",
			script: "echo ${{ env.BAR}} > ${{ env.FOO }}",
			setup: func(rr *proto.RunRequest) {
				rr.Id = "111"
				rr.Env = map[string]string{
					"FOO": "blammo.txt",
					"BAR": "foobarbaz",
				}
			},
		},
		"job vars": {
			jobWorkDir: true,
			id:         "222",
			script:     "echo ${{ job.BAR}} > ${{ job.FOO }}",
			setup: func(rr *proto.RunRequest) {
				rr.Id = "222"
				rr.Job.Variables = []*proto.Variable{
					{Key: "FOO", Value: "blammo.txt"},
					{Key: "BAR", Value: "foobarbaz"},
				}
			},
		},
		"job file vars": {
			jobWorkDir: true,
			id:         "333",
			script:     "cat ${{ job.BAR}} > ${{ job.FOO }}",
			setup: func(rr *proto.RunRequest) {
				rr.Id = "333"
				rr.Job.Variables = []*proto.Variable{
					{Key: "FOO", Value: "blammo.txt"},
					{Key: "BAR", Value: "foobarbaz", File: true},
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			bg := context.Background()
			srs, client, cleanup := startService(t)
			defer cleanup()

			rr := makeRunRequest(t, makeBashStep(tt.script), tt.jobWorkDir)
			tt.setup(rr)

			_, err := client.Run(bg, rr)
			require.NoError(t, err)

			job, ok := srs.jobs.Get(tt.id)
			require.True(t, ok)
			defer os.RemoveAll(job.WorkDir)

			assert.Eventually(t, job.Finished, time.Millisecond*500, time.Millisecond*50)
			assert.NoError(t, job.Ctx.Err())

			res, err := job.Result()
			assert.Nil(t, err)
			require.NotNil(t, res)

			assert.Equal(t, int32(0), res.ExitCode)
			assert.FileExists(t, path.Join(job.WorkDir, "blammo.txt"))
			data, err := os.ReadFile(path.Join(job.WorkDir, "blammo.txt"))
			require.NoError(t, err)
			assert.Equal(t, "foobarbaz", strings.TrimSpace(string(data)))

			job.Close()
			assert.NoDirExists(t, job.TmpDir)
		})
	}
}

func Test_StepRunnerService_FollowSteps(t *testing.T) {
	bg := context.Background()
	srs, client, cleanup := startService(t)
	defer cleanup()

	rr := makeRunRequest(t, makeBashStep("sleep 1"), false)

	_, err := client.Run(bg, rr)
	require.NoError(t, err)

	stream, err := client.FollowSteps(bg, &proto.FollowStepsRequest{Id: id})
	require.NoError(t, err)

	got, err := stream.Recv()
	require.NoError(t, err)
	require.NotNil(t, got)

	// since there's currently only one step-result, a subsequent read should return EOF.
	_, err = stream.Recv()
	require.True(t, errors.Is(err, io.EOF))

	job, ok := srs.jobs.Get(id)
	require.True(t, ok)
	defer os.RemoveAll(job.WorkDir)
	want, _ := job.Result()

	assert.Equal(t, want.String(), got.Result.String())
}
