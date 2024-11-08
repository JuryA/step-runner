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

	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"

	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/jobs"
	"gitlab.com/gitlab-org/step-runner/proto"
)

const (
	scriptStep = `spec: {}
---
steps:
  - name: script_step
    script: %s
`
)

func makeScriptStep(cmd string) string {
	return fmt.Sprintf(scriptStep, cmd)
}

const bufSize = 1024 * 1024

func must(e error) {
	if e != nil {
		panic(e)
	}
}

var (
	conn         *grpc.ClientConn
	stepsService *StepRunnerService
	apiClient    proto.StepRunnerClient
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	stepCache, err := cache.New()
	must(err)

	stepsService = New(stepCache, runner.NewEmptyEnvironment())

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

func Test_StepRunnerService_Run_Success(t *testing.T) {
	defer cleanup(t)

	bg := context.Background()
	rr := test.ProtoRunRequest(t, makeScriptStep("echo foo bar baz"), false)

	_, err := apiClient.Run(bg, rr)
	require.NoError(t, err)

	job, ok := stepsService.jobs.Get(rr.Id)
	require.True(t, ok)

	assert.Eventually(t, job.Finished, time.Second*20, time.Millisecond*50)
	assert.NoError(t, job.Ctx.Err())

	stat := job.Status()
	assert.Empty(t, stat.Message)
	assert.Equal(t, proto.StepResult_success, stat.Status)

	apiClient.Close(bg, &proto.CloseRequest{Id: rr.Id})
	assert.NoDirExists(t, job.TmpDir)
}

func Test_StepRunnerService_Run_RequestCancelled(t *testing.T) {
	defer cleanup(t)

	stepCache, err := cache.New()
	require.NoError(t, err)
	srs := New(stepCache, runner.NewEmptyEnvironment())

	rr := test.ProtoRunRequest(t, makeScriptStep("echo foo bar baz"), false)

	ctx, cancel := context.WithCancel(context.TODO())
	cancel()
	_, err = srs.Run(ctx, rr)
	require.Error(t, err)
	require.ErrorContains(t, err, context.Canceled.Error())

	_, ok := srs.jobs.Get(rr.Id)
	require.False(t, ok)

	assert.NoDirExists(t, path.Join(os.TempDir(), "step-runner-output-"+rr.Id))
}

func Test_StepRunnerService_Run_Cancelled(t *testing.T) {
	defer cleanup(t)
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
				stat := j.Status()
				assert.Contains(t, stat.Message, context.Canceled.Error())
				assert.Equal(t, proto.StepResult_cancelled, stat.Status)
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
				stat := j.Status()
				assert.Empty(t, stat.Message)
				assert.Equal(t, proto.StepResult_success, stat.Status)
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
				stat := j.Status()
				assert.Contains(t, stat.Message, "signal: killed")
				assert.Equal(t, proto.StepResult_cancelled, stat.Status)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			wg := sync.WaitGroup{}
			wg.Add(1)

			rr := test.ProtoRunRequest(t, makeScriptStep(tt.script), false)
			_, err := apiClient.Run(bg, rr)
			require.NoError(t, err)

			job, ok := stepsService.jobs.Get(rr.Id)
			require.True(t, ok)

			go tt.finish(job, apiClient, &wg)

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
	defer cleanup(t, "blammo.txt")

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

			rr := test.ProtoRunRequest(t, makeScriptStep(tt.script), tt.jobWorkDir)
			tt.setup(rr)

			_, err := apiClient.Run(bg, rr)
			require.NoError(t, err)

			job, ok := stepsService.jobs.Get(rr.Id)
			require.True(t, ok)

			assert.Eventually(t, job.Finished, time.Millisecond*500, time.Millisecond*50)
			assert.NoError(t, job.Ctx.Err())

			stat := job.Status()
			assert.Empty(t, stat.Message)
			assert.Equal(t, proto.StepResult_success, stat.Status)

			assert.FileExists(t, path.Join(job.WorkDir, "blammo.txt"))
			data, err := os.ReadFile(path.Join(job.WorkDir, "blammo.txt"))
			require.NoError(t, err)
			assert.Equal(t, "foobarbaz", strings.TrimSpace(string(data)))

			apiClient.Close(bg, &proto.CloseRequest{Id: rr.Id})
			assert.NoDirExists(t, job.TmpDir)
		})
	}
}

func Test_StepRunnerService_Run_DuplicateID(t *testing.T) {
	defer cleanup(t)

	stepCache, err := cache.New()
	require.NoError(t, err)
	srs := New(stepCache, runner.NewEmptyEnvironment())

	rr := test.ProtoRunRequest(t, makeScriptStep("echo foo bar baz"), false)

	ctx := context.Background()

	_, err = srs.Run(ctx, rr)
	require.NoError(t, err)

	_, err = srs.Run(ctx, rr)
	require.NoError(t, err)
}

func Test_StepRunnerService_Close(t *testing.T) {
	defer cleanup(t)

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
				stat := j.Status()
				assert.Empty(t, stat.Message)
				assert.Equal(t, proto.StepResult_success, stat.Status)
			},
		},
		"Close called before job finished (should cancel task)": {
			cmd: "sleep 60",
			preClose: func(j *jobs.Job) {
				assert.False(t, j.Finished())
			},
			validate: func(j *jobs.Job) {
				require.Eventually(t, j.Finished, 200*time.Millisecond, 25*time.Millisecond)
				stat := j.Status()
				assert.Contains(t, stat.Message, "signal: killed")
				assert.Equal(t, proto.StepResult_cancelled, stat.Status)
			},
		},
	}

	bg := context.Background()

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rr := test.ProtoRunRequest(t, makeScriptStep(tt.cmd), false)
			_, err := apiClient.Run(bg, rr)
			require.NoError(t, err)

			// wait for job to start...
			var job *jobs.Job
			var ok bool
			require.Eventually(t, func() bool {
				job, ok = stepsService.jobs.Get(rr.Id)
				return ok && job != nil
			}, 200*time.Millisecond, 25*time.Millisecond)

			tt.preClose(job)

			_, err = apiClient.Close(bg, &proto.CloseRequest{Id: rr.Id})
			require.NoError(t, err)

			tt.validate(job)

			// the job was removed from the map of jobs
			_, ok = stepsService.jobs.Get(rr.Id)
			require.False(t, ok)
		})
	}
}

func Test_StepRunnerService_Close_BadID(t *testing.T) {
	bg := context.Background()

	_, err := apiClient.Close(bg, &proto.CloseRequest{Id: "4130"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no job with id")
}

func Test_StepRunnerService_FollowLogs(t *testing.T) {
	defer cleanup(t)

	bg := context.Background()

	rr := test.ProtoRunRequest(t, makeScriptStep("echo foo bar baz"), false)
	_, err := apiClient.Run(bg, rr)
	require.NoError(t, err)

	stream, err := apiClient.FollowLogs(bg, &proto.FollowLogsRequest{Id: rr.Id})
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

	apiClient.Close(bg, &proto.CloseRequest{Id: rr.Id})

	require.True(t, errors.Is(err, io.EOF))
	require.Equal(t, "foo bar baz\n", logs.String())
}

func Test_StepRunnerService_Status(t *testing.T) {
	bg := context.Background()

	type spec struct {
		runRequests func(*testing.T) []*proto.RunRequest
		validate    func(*testing.T, *spec, []*proto.RunRequest)
	}
	tests := map[string]spec{
		"single job eventually finishes": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeScriptStep("echo foo bar baz"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := stepsService.jobs.Get(rr.Id)
				assert.True(t, ok)

				sr, err := apiClient.Status(bg, &proto.StatusRequest{Id: rr.Id})
				require.NoError(t, err)
				assert.Len(t, sr.Jobs, 1)
				assert.Equal(t, rr.Id, sr.Jobs[0].Id)
				assert.Equal(t, proto.StepResult_running, sr.Jobs[0].Status)
				assert.NotNil(t, sr.Jobs[0].StartTime)
				assert.Nil(t, sr.Jobs[0].EndTime)
				assert.Empty(t, sr.Jobs[0].Message)

				assert.Eventually(t, j.Finished, time.Second*15, time.Millisecond*50)

				sr, err = apiClient.Status(bg, &proto.StatusRequest{Id: rr.Id})
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
					test.ProtoRunRequest(t, makeScriptStep("echo foo bar baz"), false),
					test.ProtoRunRequest(t, makeScriptStep("bling blang blong"), false),
				}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				sr, err := apiClient.Status(bg, &proto.StatusRequest{})
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
					test.ProtoRunRequest(t, makeScriptStep("echo foo bar baz"), false),
					test.ProtoRunRequest(t, makeScriptStep("bling blang blong"), false),
				}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[1]
				sr, err := apiClient.Status(bg, &proto.StatusRequest{Id: rr.Id})
				require.NoError(t, err)
				assert.Len(t, sr.Jobs, 1)
				assert.Equal(t, rr.Id, sr.Jobs[0].Id)
			},
		},
		"bad id in request": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeScriptStep("echo foo bar baz"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				sr, err := apiClient.Status(bg, &proto.StatusRequest{Id: "blablabla"})
				assert.Error(t, err)
				assert.Nil(t, sr)
			},
		},
		"single job failed": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeScriptStep("sdjskjdfh"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := stepsService.jobs.Get(rr.Id)
				require.True(t, ok)

				assert.Eventually(t, j.Finished, time.Second*20, time.Millisecond*50)

				sr, err := apiClient.Status(bg, &proto.StatusRequest{Id: rr.Id})
				assert.NoError(t, err)
				assert.Equal(t, proto.StepResult_failure, sr.Jobs[0].Status)
				assert.Contains(t, sr.Jobs[0].Message, "exec: exit status 127")
			},
		},
		"single job cancelled before execution start": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeScriptStep("sleep 1"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := stepsService.jobs.Get(rr.Id)
				require.True(t, ok)

				// cancel the job before it starts executing. This cancels but does not delete the job
				j.Close()

				sr, err := apiClient.Status(bg, &proto.StatusRequest{})
				assert.NoError(t, err)
				// exit-code was never set
				assert.Equal(t, proto.StepResult_cancelled, sr.Jobs[0].Status)
				assert.Contains(t, sr.Jobs[0].Message, context.Canceled.Error())
			},
		},
		"single job cancelled after execution start": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeScriptStep("sleep 1"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := stepsService.jobs.Get(rr.Id)
				require.True(t, ok)

				// give the job enough time to start execution
				time.Sleep(250 * time.Millisecond)
				// this cancels but does not delete the job
				j.Close()

				assert.Eventually(t, func() bool {
					sr, err := apiClient.Status(bg, &proto.StatusRequest{})
					return err == nil &&
						assert.Equal(t, proto.StepResult_cancelled, sr.Jobs[0].Status) && // :-(
						strings.Contains(sr.Jobs[0].Message, "signal: killed")
				}, time.Second*2, time.Millisecond*250)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defer cleanup(t)

			rrs := tt.runRequests(t)
			for _, rr := range rrs {
				_, err := apiClient.Run(bg, rr)
				require.NoError(t, err)
				defer apiClient.Close(bg, &proto.CloseRequest{Id: rr.Id})
			}

			tt.validate(t, &tt, rrs)
		})
	}
}
