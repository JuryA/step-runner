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

	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
	"gitlab.com/gitlab-org/step-runner/proto"
)

var errFinal = fmt.Errorf("FOO")

func stepStepResult(status proto.StepResult_Status, subStepResults ...*proto.StepResult) *proto.StepResult {
	return &proto.StepResult{
		SpecDefinition: &proto.SpecDefinition{Definition: &proto.Definition{Type: proto.DefinitionType_steps}},
		Status:         status,
		SubStepResults: subStepResults,
	}
}

func execStepResult(status proto.StepResult_Status, exitCode int) *proto.StepResult {
	return &proto.StepResult{
		SpecDefinition: &proto.SpecDefinition{Definition: &proto.Definition{Type: proto.DefinitionType_exec}},
		ExecResult:     &proto.StepResult_ExecResult{ExitCode: int32(exitCode)},
		Status:         status,
	}
}

func Test_New(t *testing.T) {
	runReq := test.ProtoRunRequest(t, "", false)
	j, err := New(runReq)
	require.NoError(t, err)
	defer j.Close()

	assert.Equal(t, runReq.Id, j.ID)
	assert.DirExists(t, j.TmpDir)
}

func Test_New_Job_WorkDir(t *testing.T) {
	runReq := test.ProtoRunRequest(t, "", true)
	j, err := New(runReq)
	require.NoError(t, err)
	defer j.Close()

	assert.Equal(t, runReq.Id, j.ID)
	assert.DirExists(t, j.TmpDir)
}

func Test_Finish(t *testing.T) {
	runReq := test.ProtoRunRequest(t, "", true)
	j, err := New(runReq)
	require.NoError(t, err)
	defer j.Close()

	sr := execStepResult(proto.StepResult_failure, 123456)
	j.Finish(sr, nil)

	assert.True(t, j.finished)
	assert.WithinDuration(t, time.Now(), j.finishTime, time.Millisecond*5)

	assert.Equal(t, sr.Status, j.Status().Status)
	assert.Nil(t, j.err)

	// stepResult and err remain unchanged on subsequent calls to Close
	j.Finish(nil, errFinal)
	assert.Equal(t, sr.Status, j.Status().Status)
	assert.NoError(t, j.err)
}

func Test_Close_AlreadyFinished(t *testing.T) {
	j, err := New(test.ProtoRunRequest(t, "", false))
	require.NoError(t, err)

	sr := execStepResult(proto.StepResult_failure, 123456)
	j.Finish(sr, nil)

	j.Close()

	assert.Equal(t, sr.Status, j.Status().Status)
	assert.Nil(t, j.err)
	assert.WithinDuration(t, time.Now(), j.finishTime, time.Millisecond*5)

	assert.NoDirExists(t, j.TmpDir)
}

func Test_Close(t *testing.T) {
	j, err := New(test.ProtoRunRequest(t, "", false))
	require.NoError(t, err)

	j.Close()

	assert.True(t, j.finished)
	assert.Equal(t, proto.StepResult_cancelled, j.Status().Status)
	assert.True(t, errors.Is(j.err, j.Ctx.Err()))

	assert.NoDirExists(t, j.TmpDir)
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
		runStepErr  error
		writeErr    error
		wantErr     string
		wantWritten []byte
	}{
		"write error, incomplete logs written, error returned": {
			writeErr:    errors.New("POW!!!"),
			runStepErr:  errors.New("BLAMMO!!!"),
			wantErr:     `following logs for job "\d*": streaming logs: POW!!!`,
			wantWritten: data[0][:len(data[0])-1],
		},
		"step execution error, logs written successfully, no error returned": {
			writeErr:    nil,
			runStepErr:  context.Canceled,
			wantErr:     "",
			wantWritten: bytes.Join(data, nil),
		},

		"no error, logs written successfully, no error returned": {
			writeErr:    nil,
			runStepErr:  nil,
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
				j.Finish(nil, tt.runStepErr)
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
		validate    func(*testing.T, *proto.Status)
	}{
		"exec-step job running": {
			stepResults: execStepResult(proto.StepResult_running, -1),
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_running, got.Status)
				assert.Nil(t, got.EndTime)
				assert.Empty(t, got.Message)
			},
		},
		"exec-step job succeeded": {
			finish:      true,
			stepResults: stepStepResult(proto.StepResult_success, execStepResult(proto.StepResult_success, 0)),
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_success, got.Status)
				assert.NotNil(t, got.EndTime)
				assert.Empty(t, got.Message)
			},
		},
		"exec-step job failed": {
			finish:      true,
			stepResults: stepStepResult(proto.StepResult_failure, execStepResult(proto.StepResult_failure, 1)),
			finishErr:   errors.New("BLAMMO!!!"),
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_failure, got.Status)
				assert.NotNil(t, got.EndTime)
				assert.Contains(t, got.Message, "BLAMMO!!!")
			},
		},
		"exec-step job finished, missing exec_result": { // is this even possible?
			finish: true,
			stepResults: stepStepResult(proto.StepResult_success, &proto.StepResult{
				SpecDefinition: &proto.SpecDefinition{Definition: &proto.Definition{Type: proto.DefinitionType_exec}},
				Status:         proto.StepResult_success,
			}),
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_success, got.Status)
				assert.NotNil(t, got.EndTime)
				assert.Empty(t, got.Message)
			},
		},
		"exec-step job fails, then job cancelled": {
			finish:      true,
			stepResults: stepStepResult(proto.StepResult_failure, execStepResult(proto.StepResult_failure, 1)),
			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_failure, got.Status)
				assert.NotNil(t, got.EndTime)
				assert.Empty(t, got.Message)
			},
		},
		"job cancelled before execution": {
			finish:      true,
			stepResults: nil,
			finishErr:   context.Canceled,

			validate: func(t *testing.T, got *proto.Status) {
				assert.Equal(t, proto.StepResult_cancelled, got.Status)
				assert.NotNil(t, got.EndTime)
				assert.Contains(t, got.Message, context.Canceled.Error())
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			j, err := New(test.ProtoRunRequest(t, "", true))
			require.NoError(t, err)
			defer j.Close()

			if tt.finish {
				j.Finish(tt.stepResults, tt.finishErr)
			}

			gotStat := j.Status()

			assert.NotNil(t, gotStat.StartTime)
			assert.Equal(t, j.ID, gotStat.Id)
			tt.validate(t, gotStat)
		})
	}
}
