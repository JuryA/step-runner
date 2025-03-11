package service_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test/server"

	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/jobs"
	"gitlab.com/gitlab-org/step-runner/proto"
)

const (
	scriptStep = `spec: {}
---
run:
  - name: %s
    script: %s
`
)

var nonAlphaNumericRe = regexp.MustCompile("[^a-zA-Z0-9_]+")
var whitespaceRe = regexp.MustCompile(`\s+`)

func makeScriptStep(t *testing.T, cmd string) string {
	noWhitespace := whitespaceRe.ReplaceAllString(t.Name(), "_")
	stepName := nonAlphaNumericRe.ReplaceAllString(noWhitespace, "")

	return fmt.Sprintf(scriptStep, stepName, cmd)
}

func jobFinished(j *jobs.Job) func() bool {
	return func() bool {
		stat := j.Status()
		return stat.Status == proto.StepResult_success || stat.Status == proto.StepResult_failure || stat.Status == proto.StepResult_cancelled
	}
}

func jobStatusIs(j *jobs.Job, status proto.StepResult_Status) func() bool {
	return func() bool {
		return j.Status().Status == status
	}
}

func Test_StepRunnerService_Run_Success(t *testing.T) {
	ctx := context.Background()
	rr := test.ProtoRunRequest(t, makeScriptStep(t, "echo foo bar baz"), false)

	srvr := server.New(t).Serve()
	apiClient := proto.NewStepRunnerClient(srvr.NewConnection())

	_, err := apiClient.Run(ctx, rr)
	require.NoError(t, err)

	job, ok := srvr.GetJob(rr.Id)
	require.True(t, ok)
	require.Equal(t, statusName(proto.StepResult_success), statusName(job.Status().Status))
	require.Empty(t, job.Status().Message)
	require.NoError(t, job.Ctx.Err())

	_, err = apiClient.Close(ctx, &proto.CloseRequest{Id: rr.Id})
	require.NoError(t, err)
	require.NoDirExists(t, job.TmpDir)

	// the job was removed from the map of jobs
	_, ok = srvr.GetJob(rr.Id)
	require.False(t, ok)
}

func Test_StepRunnerService_Run(t *testing.T) {
	t.Run("job has unspecified status before running", func(t *testing.T) {
		ctx := context.Background()
		wg := sync.WaitGroup{}
		wg.Add(1)

		executor := func(delegate func()) {
			wg.Done()
		}
		srvr := server.New(t, server.WithExecutor(executor)).Serve()
		apiClient := proto.NewStepRunnerClient(srvr.NewConnection())

		rr := test.ProtoRunRequest(t, makeScriptStep(t, "echo 'yes we can!!!'"), false)
		t.Cleanup(func() { _, _ = apiClient.Close(ctx, &proto.CloseRequest{Id: rr.Id}) })

		go func() {
			_, err := apiClient.Run(ctx, rr)
			require.NoError(t, err)
		}()

		wg.Wait()
		job, ok := srvr.GetJob(rr.Id)
		require.True(t, ok)
		require.Equal(t, statusName(proto.StepResult_unspecified), statusName(job.Status().Status))
	})
}

func Test_StepRunnerService_Run_Cancelled(t *testing.T) {
	t.Run("close called before request executed", func(t *testing.T) {
		ctx := context.Background()

		options := []func(*server.TestStepRunnerServer){
			server.WithExecutor(func(delegate func()) {
				// don't call the delegate so the job is created and not executed
			}),
			server.WithJobRunExitWaitTime(time.Millisecond),
		}

		srvr := server.New(t, options...).Serve()
		apiClient := proto.NewStepRunnerClient(srvr.NewConnection())

		rr := test.ProtoRunRequest(t, makeScriptStep(t, "sleep 1"), false)
		_, err := apiClient.Run(ctx, rr)
		require.NoError(t, err)

		job, ok := srvr.GetJob(rr.Id)
		require.True(t, ok)

		_, err = apiClient.Close(ctx, &proto.CloseRequest{Id: job.ID})
		require.NoError(t, err)

		// require eventually used because job.Close has an asynchronous side effect
		require.Eventually(t, jobStatusIs(job, proto.StepResult_cancelled), time.Millisecond*500, time.Millisecond*50, "job status %s", statusName(job.Status().Status))
		require.Error(t, job.Ctx.Err())
		require.Contains(t, job.Status().Message, context.Canceled.Error())
	})

	t.Run("close called before request finishes", func(t *testing.T) {
		ctx := context.Background()
		wg := sync.WaitGroup{}
		wg.Add(1)

		// run job in a goroutine, as usual. WaitGroup is used to halt the test
		// so the assert eventually starts closer to when the job is started
		executor := server.WithExecutor(func(delegate func()) {
			go delegate()
			wg.Done()
		})

		srvr := server.New(t, executor).Serve()
		apiClient := proto.NewStepRunnerClient(srvr.NewConnection())

		// run a script that takes a long time to minimize chances of it finishing before being cancelled
		rr := test.ProtoRunRequest(t, makeScriptStep(t, "sleep 5"), false)
		_, err := apiClient.Run(ctx, rr)
		require.NoError(t, err)

		job, ok := srvr.GetJob(rr.Id)
		require.True(t, ok)

		wg.Wait()
		_, _ = apiClient.Close(ctx, &proto.CloseRequest{Id: job.ID})

		require.Eventually(t, jobStatusIs(job, proto.StepResult_cancelled), time.Second*500, time.Millisecond*50, "job status %s", statusName(job.Status().Status))
		require.Error(t, job.Ctx.Err())

		// the variability here is due to Go's Cmd.Start checking if the context has an error before starting the process
		require.True(t, strings.Contains(job.Status().Message, "signal: killed") ||
			strings.Contains(job.Status().Message, "context canceled"))
	})
}

func Test_StepRunnerService_Run_Vars(t *testing.T) {
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

	server := server.New(t).Serve()
	apiClient := proto.NewStepRunnerClient(server.NewConnection())

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			bg := context.Background()

			rr := test.ProtoRunRequest(t, makeScriptStep(t, tt.script), tt.jobWorkDir)
			tt.setup(rr)

			_, err := apiClient.Run(bg, rr)
			require.NoError(t, err)

			job, ok := server.GetJob(rr.Id)
			require.True(t, ok)

			assert.Eventually(t, jobFinished(job), time.Millisecond*500, time.Millisecond*50, "job status %s", statusName(job.Status().Status))
			assert.NoError(t, job.Ctx.Err())

			stat := job.Status()
			assert.Empty(t, stat.Message)
			assert.Equal(t, statusName(proto.StepResult_success), statusName(stat.Status))

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
	server := server.New(t).Serve()
	apiClient := proto.NewStepRunnerClient(server.NewConnection())

	rr := test.ProtoRunRequest(t, makeScriptStep(t, "echo foo bar baz"), false)

	ctx := context.Background()

	_, err := apiClient.Run(ctx, rr)
	require.NoError(t, err)

	_, err = apiClient.Run(ctx, rr)
	require.NoError(t, err)
}

func Test_StepRunnerService_Close_BadID(t *testing.T) {
	server := server.New(t).Serve()
	apiClient := proto.NewStepRunnerClient(server.NewConnection())

	bg := context.Background()
	_, err := apiClient.Close(bg, &proto.CloseRequest{Id: "4130"})
	require.NoError(t, err)
}

func Test_StepRunnerService_FollowLogs(t *testing.T) {
	rr := test.ProtoRunRequest(t, makeScriptStep(t, "echo foo bar baz"), false)
	server := server.New(t).Serve()
	apiClient := proto.NewStepRunnerClient(server.NewConnection())

	bg := context.Background()
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
	require.Contains(t, logs.String(), `Running step "Test_StepRunnerService_FollowLogs"`)
	require.Contains(t, logs.String(), "foo bar baz\n")
}

func Test_StepRunnerService_Status(t *testing.T) {
	bg := context.Background()

	server := server.New(t).Serve()
	apiClient := proto.NewStepRunnerClient(server.NewConnection())

	type spec struct {
		runRequests func(*testing.T) []*proto.RunRequest
		validate    func(*testing.T, *spec, []*proto.RunRequest)
	}
	tests := map[string]spec{
		"single job eventually finishes": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeScriptStep(t, "echo foo bar baz"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := server.GetJob(rr.Id)
				assert.True(t, ok)

				sr, err := apiClient.Status(bg, &proto.StatusRequest{Id: rr.Id})
				require.NoError(t, err)
				assert.Len(t, sr.Jobs, 1)
				assert.Equal(t, rr.Id, sr.Jobs[0].Id)
				assert.Equal(t, statusName(proto.StepResult_running), statusName(sr.Jobs[0].Status))
				assert.NotNil(t, sr.Jobs[0].StartTime)
				assert.Nil(t, sr.Jobs[0].EndTime)
				assert.Empty(t, sr.Jobs[0].Message)

				assert.Eventually(t, jobFinished(j), time.Second*15, time.Millisecond*50, "job status %s", statusName(j.Status().Status))

				sr, err = apiClient.Status(bg, &proto.StatusRequest{Id: rr.Id})
				require.NoError(t, err)
				assert.Len(t, sr.Jobs, 1)
				assert.Equal(t, rr.Id, sr.Jobs[0].Id)
				assert.Equal(t, statusName(proto.StepResult_success), statusName(sr.Jobs[0].Status))
				assert.NotNil(t, sr.Jobs[0].EndTime)
				assert.Empty(t, sr.Jobs[0].Message)
			},
		},
		"multiple jobs, no ids in request": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{
					test.ProtoRunRequest(t, makeScriptStep(t, "echo foo bar baz"), false),
					test.ProtoRunRequest(t, makeScriptStep(t, "bling blang blong"), false),
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
					test.ProtoRunRequest(t, makeScriptStep(t, "echo foo bar baz"), false),
					test.ProtoRunRequest(t, makeScriptStep(t, "bling blang blong"), false),
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
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeScriptStep(t, "echo foo bar baz"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				sr, err := apiClient.Status(bg, &proto.StatusRequest{Id: "blablabla"})
				assert.Error(t, err)
				assert.Nil(t, sr)
			},
		},
		"single job failed": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeScriptStep(t, "sdjskjdfh"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := server.GetJob(rr.Id)
				require.True(t, ok)

				assert.Eventually(t, jobFinished(j), time.Second*20, time.Millisecond*50, "job status %s", statusName(j.Status().Status))

				sr, err := apiClient.Status(bg, &proto.StatusRequest{Id: rr.Id})
				assert.NoError(t, err)
				assert.Equal(t, statusName(proto.StepResult_failure), statusName(sr.Jobs[0].Status))
				assert.Contains(t, sr.Jobs[0].Message, "exec: exit status 127")
			},
		},
		"single job cancelled after execution start": {
			runRequests: func(t *testing.T) []*proto.RunRequest {
				return []*proto.RunRequest{test.ProtoRunRequest(t, makeScriptStep(t, "sleep 1"), false)}
			},
			validate: func(t *testing.T, s *spec, runRequests []*proto.RunRequest) {
				rr := runRequests[0]
				j, ok := server.GetJob(rr.Id)
				require.True(t, ok)

				// give the job enough time to start execution
				time.Sleep(250 * time.Millisecond)
				// this cancels but does not delete the job
				j.Close()

				assert.Eventually(t, func() bool {
					sr, err := apiClient.Status(bg, &proto.StatusRequest{})
					return err == nil &&
						assert.Equal(t, statusName(proto.StepResult_cancelled), statusName(sr.Jobs[0].Status)) && // :-(
						strings.Contains(sr.Jobs[0].Message, "signal: killed")
				}, time.Second*2, time.Millisecond*250)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

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

func statusName(status proto.StepResult_Status) string {
	return proto.StepResult_Status_name[int32(status)]
}
