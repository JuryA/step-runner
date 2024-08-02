package service

import (
	"bytes"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/test"

	"gitlab.com/gitlab-org/step-runner/pkg/jobs"
	"gitlab.com/gitlab-org/step-runner/proto"
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
    step: ../testdata/bash
    inputs:
        script: %s
`
)

func makeBashStep(cmd string) string {
	return fmt.Sprintf(bashStep, cmd)
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
	defer os.RemoveAll(test.TestDirName(t))

	bg := context.Background()
	srs, client, cleanup := startService(t)
	defer cleanup()

	rr := test.ProtoRunRequest(t, helloStep, false)

	_, err := client.Run(bg, rr)
	require.NoError(t, err)

	job, ok := srs.jobs.Get(rr.Id)
	require.True(t, ok)

	assert.Eventually(t, job.Finished, time.Second*20, time.Millisecond*50)
	assert.NoError(t, job.Ctx.Err())

	res, err := job.Result()
	assert.Nil(t, err)
	require.NotNil(t, res)

	assert.Equal(t, proto.StepResult_success, res.Status)

	client.Close(bg, &proto.CloseRequest{Id: rr.Id})
	assert.NoDirExists(t, job.TmpDir)
}

func Test_StepRunnerService_Run_Cancelled(t *testing.T) {
	defer os.RemoveAll(test.TestDirName(t))
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
				res, err := j.Result()
				assert.True(t, errors.Is(err, context.Canceled))
				assert.Nil(t, res)
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
				res, err := j.Result()
				assert.NoError(t, err)
				assert.NotNil(t, res)
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
				res, err := j.Result()
				assert.True(t, errors.Is(err, context.Canceled))
				assert.Nil(t, res)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			srs, client, cleanup := startService(t)
			defer cleanup()

			wg := sync.WaitGroup{}
			wg.Add(1)

			rr := test.ProtoRunRequest(t, makeBashStep(tt.script), false)
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

			assert.Eventually(t, func() bool {
				return assert.NoDirExists(t, job.TmpDir)
			}, time.Millisecond*5500, time.Millisecond*100)
		})
	}
}

func Test_StepRunnerService_Run_Vars(t *testing.T) {
	defer os.RemoveAll(test.TestDirName(t))

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

			rr := test.ProtoRunRequest(t, makeBashStep(tt.script), tt.jobWorkDir)
			tt.setup(rr)

			_, err := client.Run(bg, rr)
			require.NoError(t, err)

			job, ok := srs.jobs.Get(rr.Id)
			require.True(t, ok)
			defer os.RemoveAll(job.WorkDir)

			assert.Eventually(t, job.Finished, time.Millisecond*500, time.Millisecond*50)
			assert.NoError(t, job.Ctx.Err())

			res, err := job.Result()
			assert.Nil(t, err)
			require.NotNil(t, res)

			assert.Equal(t, proto.StepResult_success, res.Status)
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
	defer os.RemoveAll(test.TestDirName(t))

	bg := context.Background()
	srs, client, cleanup := startService(t)
	defer cleanup()

	rr := test.ProtoRunRequest(t, makeBashStep("sleep 1"), false)
	_, err := client.Run(bg, rr)
	require.NoError(t, err)
	defer client.Close(bg, &proto.CloseRequest{Id: rr.Id})

	stream, err := client.FollowSteps(bg, &proto.FollowStepsRequest{Id: rr.Id})
	require.NoError(t, err)

	got, err := stream.Recv()
	require.NoError(t, err)
	require.NotNil(t, got)

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
	defer os.RemoveAll(test.TestDirName(t))

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
	defer os.RemoveAll(test.TestDirName(t))

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
				r, err := j.Result()
				assert.Nil(t, err)
				assert.NotNil(t, r)
			},
		},
		"Close called before job finished (should cancel task)": {
			cmd: "sleep 60",
			preClose: func(j *jobs.Job) {
				assert.False(t, j.Finished())
			},
			validate: func(j *jobs.Job) {
				require.Eventually(t, j.Finished, 200*time.Millisecond, 25*time.Millisecond)
				r, err := j.Result()
				assert.True(t, errors.Is(err, context.Canceled))
				assert.Nil(t, r)
			},
		},
	}

	bg := context.Background()
	srs, client, cleanup := startService(t)
	defer cleanup()

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rr := test.ProtoRunRequest(t, makeBashStep(tt.cmd), false)
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

func Test_StepRunnerService_FollowLogs(t *testing.T) {
	defer os.RemoveAll(test.TestDirName(t))

	bg := context.Background()
	_, client, cleanup := startService(t)
	defer cleanup()

	rr := test.ProtoRunRequest(t, helloStep, false)
	_, err := client.Run(bg, rr)
	require.NoError(t, err)

	stream, err := client.FollowLogs(bg, &proto.FollowLogsRequest{Id: rr.Id})
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

	client.Close(bg, &proto.CloseRequest{Id: rr.Id})

	require.True(t, errors.Is(err, io.EOF))
	require.Equal(t, "meet steppy who is 1 likes {\"color\":\"red\"} and is hungry false\n", logs.String())
}

func Test_StepRunnerService_Status(t *testing.T) {
	bg := context.Background()
	srv, client, cleanup := startService(t)
	defer cleanup()

	type spec struct {
		runRequests func(*testing.T) []*proto.RunRequest
		validate    func(*testing.T, *spec, []*proto.RunRequest)
	}
	tests := map[string]spec{
		"single job eventually finishes": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, helloStep, false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := srv.jobs.Get(rr.Id)
				assert.True(t, ok)

				sr, err := client.Status(bg, &proto.StatusRequest{Id: rr.Id})
				require.NoError(t, err)
				assert.Len(t, sr.Jobs, 1)
				assert.Equal(t, rr.Id, sr.Jobs[0].Id)
				assert.Equal(t, proto.StepResult_running, sr.Jobs[0].Status)
				assert.NotNil(t, sr.Jobs[0].StartTime)
				assert.Nil(t, sr.Jobs[0].EndTime)
				assert.Empty(t, sr.Jobs[0].Message)

				assert.Eventually(t, j.Finished, time.Second*15, time.Millisecond*50)

				sr, err = client.Status(bg, &proto.StatusRequest{Id: rr.Id})
				require.NoError(t, err)
				assert.Len(t, sr.Jobs, 1)
				assert.Equal(t, rr.Id, sr.Jobs[0].Id)
				assert.Equal(t, proto.StepResult_success, sr.Jobs[0].Status)
				assert.NotNil(t, sr.Jobs[0].EndTime)
				assert.Empty(t, sr.Jobs[0].Message)
			},
		},
		"multiple jobs, no ids in request": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{
					test.ProtoRunRequest(t, helloStep, false),
					test.ProtoRunRequest(t, helloStep, false),
				}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				sr, err := client.Status(bg, &proto.StatusRequest{})
				require.NoError(t, err)
				assert.Len(t, sr.Jobs, 2)
				ids := []string{runRequests[0].Id, runRequests[1].Id}
				assert.Contains(t, ids, sr.Jobs[0].Id)
				assert.Contains(t, ids, sr.Jobs[1].Id)
			},
		},
		"multiple jobs, single id in request": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{
					test.ProtoRunRequest(t, helloStep, false),
					test.ProtoRunRequest(t, helloStep, false),
				}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[1]
				sr, err := client.Status(bg, &proto.StatusRequest{Id: rr.Id})
				require.NoError(t, err)
				assert.Len(t, sr.Jobs, 1)
				assert.Equal(t, rr.Id, sr.Jobs[0].Id)
			},
		},
		"bad id in request": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, helloStep, false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				sr, err := client.Status(bg, &proto.StatusRequest{Id: "blablabla"})
				assert.Error(t, err)
				assert.Nil(t, sr)
			},
		},
		"single job failed": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeBashStep("sdjskjdfh"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := srv.jobs.Get(rr.Id)
				require.True(t, ok)

				assert.Eventually(t, j.Finished, time.Second*20, time.Millisecond*50)

				sr, err := client.Status(bg, &proto.StatusRequest{Id: rr.Id})
				assert.NoError(t, err)
				assert.Equal(t, proto.StepResult_failure, sr.Jobs[0].Status)
				assert.Contains(t, sr.Jobs[0].Message, "exec: exit status 127")
			},
		},
		"single job cancelled before execution start": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeBashStep("sleep 1"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := srv.jobs.Get(rr.Id)
				require.True(t, ok)

				// cancel the job before it starts executing. This cancels but does not delete the job
				j.Close()

				sr, err := client.Status(bg, &proto.StatusRequest{})
				assert.NoError(t, err)
				// exit-code was never set
				assert.Equal(t, proto.StepResult_cancelled, sr.Jobs[0].Status)
				assert.Contains(t, sr.Jobs[0].Message, context.Canceled.Error())
			},
		},
		"single job cancelled after execution start": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeBashStep("sleep 1"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := srv.jobs.Get(rr.Id)
				require.True(t, ok)

				// give the job enough time to start execution
				time.Sleep(250 * time.Millisecond)
				// this cancels but does not delete the job
				j.Close()

				assert.Eventually(t, func() bool {
					sr, err := client.Status(bg, &proto.StatusRequest{})
					return err == nil &&
						assert.Equal(t, proto.StepResult_cancelled, sr.Jobs[0].Status) && // :-(
						strings.Contains(sr.Jobs[0].Message, context.Canceled.Error())
				}, time.Second*2, time.Millisecond*250)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defer os.RemoveAll(test.TestDirName(t))

			rrs := tt.runRequests(t)
			for _, rr := range rrs {
				_, err := client.Run(bg, rr)
				require.NoError(t, err)
				defer client.Close(bg, &proto.CloseRequest{Id: rr.Id})
			}

			tt.validate(t, &tt, rrs)
		})
	}
}
