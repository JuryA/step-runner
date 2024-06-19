package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/pkg/jobs"
	"gitlab.com/gitlab-org/step-runner/proto"
	"golang.org/x/exp/rand"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
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

func randID() string { return strconv.Itoa(rand.Intn(999)) }

func makeRunRequest(t *testing.T, step string, withJob bool) *proto.RunRequest {
	testDir := testDirName(t)
	runReq := proto.RunRequest{
		Id:    randID(),
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
	defer os.RemoveAll(testDirName(t))

	bg := context.Background()
	srs, client, cleanup := startService(t)
	defer cleanup()

	rr := makeRunRequest(t, helloStep, false)

	_, err := client.Run(bg, rr)
	require.NoError(t, err)

	job, ok := srs.jobs.Get(rr.Id)
	require.True(t, ok)

	assert.Eventually(t, job.Finished, time.Second*20, time.Millisecond*50)
	assert.NoError(t, job.Ctx.Err())
	assert.NoError(t, job.Err())

	client.Close(bg, &proto.CloseRequest{Id: rr.Id})
	assert.NoDirExists(t, job.TmpDir)
}

func Test_StepRunnerService_Run_Cancelled(t *testing.T) {
	defer os.RemoveAll(testDirName(t))
	bg := context.Background()

	tests := map[string]struct {
		id       string
		script   string
		finish   func(*jobs.Job, proto.StepRunnerClient, *sync.WaitGroup)
		validate func(*jobs.Job)
	}{
		"Close called before request executed": {
			script: "sleep 1",
			finish: func(j *jobs.Job, client proto.StepRunnerClient, wg *sync.WaitGroup) {
				defer wg.Done()
				client.Close(bg, &proto.CloseRequest{Id: j.ID})
			},
			validate: func(j *jobs.Job) {
				assert.True(t, errors.Is(j.Err(), context.Canceled))
			},
		},
		"Close called after request finished": {
			script: "sleep 1",
			finish: func(j *jobs.Job, client proto.StepRunnerClient, wg *sync.WaitGroup) {
				defer wg.Done()
				// Make sure the step execution finished
				assert.Eventually(t, j.Finished, time.Millisecond*1900, time.Millisecond*100)
				client.Close(bg, &proto.CloseRequest{Id: j.ID})
			},
			validate: func(j *jobs.Job) {
				assert.NoError(t, j.Err())
			},
		},
		"Close called before request finishes": {
			script: "sleep 1",
			finish: func(j *jobs.Job, client proto.StepRunnerClient, wg *sync.WaitGroup) {
				defer wg.Done()
				// Make sure the step sub-process was executed before calling Finish()
				time.Sleep(time.Millisecond * 50)
				client.Close(bg, &proto.CloseRequest{Id: j.ID})
			},
			validate: func(j *jobs.Job) {
				assert.True(t, errors.Is(j.Err(), context.Canceled))
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			srs, client, cleanup := startService(t)
			defer cleanup()

			wg := sync.WaitGroup{}
			wg.Add(1)

			rr := makeRunRequest(t, makeBashStep(tt.script), false)
			_, err := client.Run(bg, rr)
			require.NoError(t, err)

			job, ok := srs.jobs.Get(rr.Id)
			require.True(t, ok)
			defer os.RemoveAll(job.WorkDir)

			go tt.finish(job, client, &wg)

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
		script     string
		setup      func(*proto.RunRequest)
	}{
		"env vars": {
			script: "echo ${{ env.BAR}} > ${{ env.FOO }}",
			setup: func(rr *proto.RunRequest) {
				rr.Env = map[string]string{
					"FOO": "blammo.txt",
					"BAR": "foobarbaz",
				}
			},
		},
		"job vars": {
			jobWorkDir: true,
			script:     "echo ${{ job.BAR}} > ${{ job.FOO }}",
			setup: func(rr *proto.RunRequest) {
				rr.Job.Variables = []*proto.Variable{
					{Key: "FOO", Value: "blammo.txt"},
					{Key: "BAR", Value: "foobarbaz"},
				}
			},
		},
		"job file vars": {
			jobWorkDir: true,
			script:     "cat ${{ job.BAR}} > ${{ job.FOO }}",
			setup: func(rr *proto.RunRequest) {
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

			job, ok := srs.jobs.Get(rr.Id)
			require.True(t, ok)
			defer os.RemoveAll(job.WorkDir)

			assert.Eventually(t, job.Finished, time.Millisecond*500, time.Millisecond*50)
			assert.NoError(t, job.Ctx.Err())

			assert.FileExists(t, path.Join(job.WorkDir, "blammo.txt"))
			data, err := os.ReadFile(path.Join(job.WorkDir, "blammo.txt"))
			require.NoError(t, err)
			assert.Equal(t, "foobarbaz", strings.TrimSpace(string(data)))

			client.Close(bg, &proto.CloseRequest{Id: rr.Id})
			assert.NoDirExists(t, job.TmpDir)
		})
	}
}

func Test_StepRunnerService_FollowSteps(t *testing.T) {
	defer os.RemoveAll(testDirName(t))

	bg := context.Background()
	srs, client, cleanup := startService(t)
	defer cleanup()

	rr := makeRunRequest(t, makeBashStep("sleep 1"), false)
	_, err := client.Run(bg, rr)
	require.NoError(t, err)

	stream, err := client.FollowSteps(bg, &proto.FollowStepsRequest{Id: rr.Id})
	require.NoError(t, err)

	got, err := stream.Recv()
	require.NoError(t, err)
	require.NotNil(t, got)
	client.Close(bg, &proto.CloseRequest{Id: rr.Id})

	// since there's currently only one step-result, a subsequent read should return EOF.
	_, err = stream.Recv()
	require.True(t, errors.Is(err, io.EOF))

	job, ok := srs.jobs.Get(rr.Id)
	require.True(t, ok)
	defer os.RemoveAll(job.WorkDir)
	want, _ := job.Result()

	assert.Equal(t, want.String(), got.Result.String())
}

func Test_StepRunnerService_FollowSteps_BadID(t *testing.T) {
	defer os.RemoveAll(testDirName(t))

	bg := context.Background()
	_, client, cleanup := startService(t)
	defer cleanup()

	stream, err := client.FollowSteps(bg, &proto.FollowStepsRequest{Id: "4130"})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no job with id")
}

func Test_StepRunnerService_Close(t *testing.T) {
	defer os.RemoveAll(testDirName(t))

	tests := map[string]struct {
		cmd      string
		preClose func(*jobs.Job)
		validate func(*jobs.Job)
	}{
		"Close called after job finished": {
			cmd: "echo 'yes we can!!!'",
			preClose: func(j *jobs.Job) {
				require.Eventually(t, j.Finished, 200*time.Millisecond, 25*time.Millisecond)
			},
			validate: func(j *jobs.Job) {
				assert.True(t, j.Finished())
				err := j.Err()
				assert.Nil(t, err)
			},
		},
		"Close called before job finished (should cancel task)": {
			cmd: "sleep 60",
			preClose: func(j *jobs.Job) {
				assert.False(t, j.Finished())
			},
			validate: func(j *jobs.Job) {
				require.Eventually(t, j.Finished, 200*time.Millisecond, 25*time.Millisecond)
				err := j.Err()
				assert.True(t, errors.Is(err, context.Canceled))
			},
		},
	}

	bg := context.Background()
	srs, client, cleanup := startService(t)
	defer cleanup()

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rr := makeRunRequest(t, makeBashStep(tt.cmd), false)
			_, err := client.Run(bg, rr)
			require.NoError(t, err)

			// wait for job to start...
			var job *jobs.Job
			var ok bool
			require.Eventually(t, func() bool {
				job, ok = srs.jobs.Get(rr.Id)
				return ok && job != nil
			}, 200*time.Millisecond, 25*time.Millisecond)

			defer os.RemoveAll(job.WorkDir)

			tt.preClose(job)

			_, err = client.Close(bg, &proto.CloseRequest{Id: rr.Id})
			require.NoError(t, err)

			tt.validate(job)

			// the job was removed from the map of jobs
			_, ok = srs.jobs.Get(rr.Id)
			require.False(t, ok)
		})
	}
}

func Test_StepRunnerService_Close_BadID(t *testing.T) {
	bg := context.Background()
	_, client, cleanup := startService(t)
	defer cleanup()

	_, err := client.Close(bg, &proto.CloseRequest{Id: "4130"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no job with id")
}
