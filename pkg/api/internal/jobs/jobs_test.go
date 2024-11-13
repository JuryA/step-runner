package jobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type mockStep struct {
	stepResult *proto.StepResult
	err        error
	sleepTime  time.Duration
}

func (m *mockStep) Describe() string {
	return "mock step"
}

func (m *mockStep) Run(_ context.Context, _ *runner.StepsContext) (*proto.StepResult, error) {
	time.Sleep(m.sleepTime)
	return m.stepResult, m.err
}

// TODO: Replace this with a mockStepBuilder
func makeMockStep(status proto.StepResult_Status, exitCode int32, err error, sleepTime time.Duration) *mockStep {
	return &mockStep{
		err:        err,
		sleepTime:  sleepTime,
		stepResult: &proto.StepResult{Status: status},
	}
}

func Test_New(t *testing.T) {
	runReq := test.ProtoRunRequest(t, "", false)
	j, err := New(runReq)
	require.NoError(t, err)
	defer j.Close()
	j.finishC <- struct{}{}

	assert.Equal(t, runReq.Id, j.ID)
	assert.DirExists(t, j.TmpDir)
}

func jobFinished(j *Job) func() bool {
	return func() bool {
		stat := j.Status()
		return stat.Status == proto.StepResult_success || stat.Status == proto.StepResult_failure || stat.Status == proto.StepResult_cancelled
	}
}

func Test_CloseNoRun(t *testing.T) {
	j, err := New(test.ProtoRunRequest(t, "", false))
	require.NoError(t, err)

	go j.Close()

	assert.Eventually(t, jobFinished(j), time.Second*3, time.Millisecond*500)

	stat := j.Status()
	assert.Equal(t, proto.StepResult_cancelled, stat.Status)
	assert.Nil(t, stat.StartTime)
	assert.Nil(t, stat.StartTime)
	assert.Equal(t, context.Canceled.Error(), stat.Message)
}

// In many cases it's impossible to test one without testing the other, so may as well do them both.
func Test_Run_Close(t *testing.T) {
	tests := map[string]struct {
		step       runner.Step
		wantErr    func(*Job) error
		wantStatus proto.StepResult_Status
		pre        func(*Job)
	}{
		"job runs to completion, success": {
			step:       makeMockStep(proto.StepResult_success, 0, nil, 0),
			wantStatus: proto.StepResult_success,
			wantErr:    func(_ *Job) error { return nil },
		},
		"job runs to completion, failure": {
			step:       makeMockStep(proto.StepResult_failure, -1, errors.New("FOO"), 0),
			wantStatus: proto.StepResult_failure,
			wantErr:    func(_ *Job) error { return errors.New("FOO") },
		},
		"job cancelled while running, final status is cancelled": {
			step:       makeMockStep(proto.StepResult_failure, -1, errors.New("signal: killed"), time.Millisecond*100),
			wantStatus: proto.StepResult_cancelled,
			wantErr:    func(_ *Job) error { return errors.New("signal: killed") },
		},
		"job cancelled before execution started": {
			step:       makeMockStep(proto.StepResult_failure, -1, errors.New("FOO"), 0),
			wantStatus: proto.StepResult_cancelled,
			wantErr:    func(j *Job) error { return fmt.Errorf("job %q cancelled before execution started", j.ID) },
			pre: func(j *Job) {
				j.cancel()
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			j, err := New(test.ProtoRunRequest(t, "", false))
			require.NoError(t, err)

			if tt.pre != nil {
				tt.pre(j)
			}

			stepsCtx, err := runner.NewStepsContext(&runner.GlobalContext{}, "foo", map[string]*structpb.Value{}, &runner.Environment{})
			require.NoError(t, err)

			go j.Run(stepsCtx, tt.step)

			time.Sleep(time.Millisecond * 10) // make sure the job at least started running before closing it

			j.Close()

			assert.True(t, jobFinished(j)())
			assert.Equal(t, tt.wantStatus, j.Status().Status)
			assert.Equal(t, tt.wantErr(j), j.err)

			assert.NoDirExists(t, j.TmpDir)

			// actually running with a nil step should cause a nil pointer exception
			j.Run(nil, nil)
			// actually running Close again will block
			assert.Eventually(t, func() bool {
				j.Close()
				return true
			}, time.Millisecond*100, time.Millisecond*50)
		})
	}
}

var data = [][]byte{
	[]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"),
	[]byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n"),
	[]byte("cccccccccccccccccccccccccccccccccc\n"),
}

// toIOWriter can be used to "cast" a func([]byte)(int, error) to an io.Writer.
type toIOWriter func([]byte) (int, error)

func (w toIOWriter) Write(p []byte) (int, error) { return w(p) }

func Test_FollowLogs(t *testing.T) {
	tests := map[string]struct {
		step        runner.Step
		writeErr    error
		wantErr     string
		wantWritten []byte
	}{
		"write error, incomplete logs written, error returned": {
			writeErr:    errors.New("POW!!!"),
			step:        makeMockStep(proto.StepResult_failure, -1, errors.New("BLAMMO!!!"), 0),
			wantErr:     `following logs for job "\d*": streaming logs: POW!!!`,
			wantWritten: data[0][:len(data[0])-1],
		},
		"step execution error, logs written successfully, no error returned": {
			writeErr:    nil,
			step:        makeMockStep(proto.StepResult_cancelled, -1, context.Canceled, 0),
			wantErr:     "",
			wantWritten: bytes.Join(data, nil),
		},
		"step execution succeeds, logs written successfully, no error returned": {
			writeErr:    nil,
			step:        makeMockStep(proto.StepResult_success, 0, nil, 0),
			wantErr:     "",
			wantWritten: bytes.Join(data, nil),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotWritten := bytes.Buffer{}

			rr := test.ProtoRunRequest(t, "", false)
			j, err := New(rr)
			require.NoError(t, err)

			defer j.Close()

			go func() {
				for _, d := range data {
					_, err := j.logs.Write(d)
					assert.NoError(t, err)
				}
				stepsCtx, err := runner.NewStepsContext(&runner.GlobalContext{}, "foo", map[string]*structpb.Value{}, &runner.Environment{})
				require.NoError(t, err)
				j.Run(stepsCtx, tt.step)
			}()

			gotErr := j.FollowLogs(context.Background(), 0, toIOWriter(func(p []byte) (int, error) {
				n, err := gotWritten.Write(p)
				require.NoError(t, err)
				return n, tt.writeErr
			}))

			if tt.wantErr == "" {
				assert.NoError(t, gotErr)
			} else {
				assert.Error(t, gotErr)
				assert.Regexp(t, regexp.MustCompile(tt.wantErr), gotErr)
			}
			assert.Equal(t, string(tt.wantWritten), gotWritten.String())
		})
	}
}

func Test_Status(t *testing.T) {
	tests := map[string]struct {
		finishErr error

		finish      bool
		stepResults *proto.StepResult
		finishError error
		set         func(*Job)
		validate    func(*testing.T, *proto.Status)
		step        *mockStep
	}{
		"job not yet run": {
			set: func(j *Job) {
			},
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_unspecified, got.Status)
				assert.Nil(t, got.StartTime)
				assert.Nil(t, got.EndTime)
				assert.Empty(t, got.Message)
			},
		},
		"job running": {
			set: func(j *Job) {
				j.status = proto.StepResult_running
				j.startTime = time.Now()
			},
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_running, got.Status)
				assert.NotNil(t, got.StartTime)
				assert.Nil(t, got.EndTime)
				assert.Empty(t, got.Message)
			},
		},
		"job succeeded": {
			set: func(j *Job) {
				j.status = proto.StepResult_success
				j.startTime = time.Now()
				j.finishTime = j.startTime.Add(time.Second)
			},
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_success, got.Status)
				assert.NotNil(t, got.StartTime)
				assert.NotNil(t, got.EndTime)
				assert.Empty(t, got.Message)
			},
		},
		"job failed": {
			set: func(j *Job) {
				j.status = proto.StepResult_failure
				j.startTime = time.Now()
				j.finishTime = j.startTime.Add(time.Second)
				j.err = errors.New("BLAMMO!!!")
			},
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_failure, got.Status)
				assert.NotNil(t, got.StartTime)
				assert.NotNil(t, got.EndTime)
				assert.Contains(t, got.Message, "BLAMMO!!!")
			},
		},
		"job cancelled after execution start": {
			set: func(j *Job) {
				j.status = proto.StepResult_cancelled
				j.startTime = time.Now()
				j.finishTime = j.startTime.Add(time.Second)
				j.err = context.Canceled
			},
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_cancelled, got.Status)
				assert.NotNil(t, got.StartTime)
				assert.NotNil(t, got.EndTime)
				assert.Contains(t, got.Message, context.Canceled.Error())
			},
		},
		"job cancelled before execution start": {
			set: func(j *Job) {
				j.status = proto.StepResult_cancelled
				j.err = context.Canceled
			},
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_cancelled, got.Status)
				assert.Nil(t, got.StartTime)
				assert.Nil(t, got.EndTime)
				assert.Contains(t, got.Message, context.Canceled.Error())
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			j, err := New(test.ProtoRunRequest(t, "", true))
			require.NoError(t, err)
			j.finishC <- struct{}{}
			defer j.Close()

			tt.set(j)

			gotStat := j.Status()

			assert.Equal(t, j.ID, gotStat.Id)
			tt.validate(t, gotStat)
		})
	}
}

func Test_computeFinalStatus(t *testing.T) {
	tests := map[string]struct {
		incomingStatus proto.StepResult_Status
		incomingErr    error
		cancelled      bool
		wantStatus     proto.StepResult_Status
	}{
		"unspecified incoming status": {
			incomingStatus: proto.StepResult_unspecified,
			incomingErr:    nil,
			cancelled:      false,
			wantStatus:     proto.StepResult_failure,
		},
		"running incoming status": {
			incomingStatus: proto.StepResult_running,
			incomingErr:    nil,
			cancelled:      false,
			wantStatus:     proto.StepResult_failure,
		},
		"success incoming status": {
			incomingStatus: proto.StepResult_success,
			incomingErr:    nil,
			cancelled:      false,
			wantStatus:     proto.StepResult_success,
		},
		"cancelled incoming status": {
			incomingStatus: proto.StepResult_cancelled,
			incomingErr:    nil,
			cancelled:      false,
			wantStatus:     proto.StepResult_cancelled,
		},
		"failed incoming status, context cancelled error": {
			incomingStatus: proto.StepResult_failure,
			incomingErr:    context.Canceled,
			cancelled:      false,
			wantStatus:     proto.StepResult_cancelled,
		},
		"failed incoming status, context expired error": {
			incomingStatus: proto.StepResult_failure,
			incomingErr:    context.DeadlineExceeded,
			cancelled:      false,
			wantStatus:     proto.StepResult_cancelled,
		},
		"failed incoming status, signal killed error, context canceled": {
			incomingStatus: proto.StepResult_failure,
			incomingErr:    errors.New("foo bar baz signal: killed"),
			cancelled:      true,
			wantStatus:     proto.StepResult_cancelled,
		},
		"failed incoming status, signal killed error, context not canceled": {
			incomingStatus: proto.StepResult_failure,
			incomingErr:    errors.New("foo bar baz signal: killed"),
			cancelled:      false,
			wantStatus:     proto.StepResult_failure,
		},
		"failed incoming status, other error, context canceled": {
			incomingStatus: proto.StepResult_failure,
			incomingErr:    errors.New("foo bar baz"),
			cancelled:      true,
			wantStatus:     proto.StepResult_failure,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			j := Job{Ctx: ctx}
			if tt.cancelled {
				cancel()
			}

			sr := &proto.StepResult{Status: tt.incomingStatus}
			gotStatus := j.computeFinalStatus(sr, tt.incomingErr)

			assert.Equal(t, tt.wantStatus, gotStatus)
		})
	}
}
