package jobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/internal/test"
	"gitlab.com/gitlab-org/step-runner/proto"
)

var (
	stepRes  = proto.StepResult{ExecResult: &proto.StepResult_ExecResult{ExitCode: 123456}}
	errFinal = fmt.Errorf("FOO")
)

func Test_New(t *testing.T) {
	runReq := test.MakeRunRequest(t, "", false)
	j, err := New(runReq)
	require.NoError(t, err)
	defer j.Close()
	defer os.RemoveAll(j.WorkDir)

	assert.Equal(t, runReq.Id, j.ID)
	assert.DirExists(t, j.TmpDir)
	assert.DirExists(t, j.WorkDir)
}

func Test_New_Job_WorkDir(t *testing.T) {
	runReq := test.MakeRunRequest(t, "", true)
	j, err := New(runReq)
	require.NoError(t, err)
	defer j.Close()
	defer os.RemoveAll(j.WorkDir)

	assert.Equal(t, runReq.Id, j.ID)
	assert.DirExists(t, j.TmpDir)
	assert.DirExists(t, j.WorkDir)
}

func Test_Result(t *testing.T) {
	j := Job{}

	r, e := j.Result()
	assert.Nil(t, r)
	assert.Nil(t, e)

	j.stepResult = &stepRes
	j.err = errFinal

	r, e = j.Result()
	assert.Equal(t, &stepRes, r)
	assert.Equal(t, errFinal, e)
}

func Test_Finish(t *testing.T) {
	runReq := test.MakeRunRequest(t, "", true)
	j, err := New(runReq)
	require.NoError(t, err)
	defer j.Close()
	defer os.RemoveAll(j.WorkDir)

	j.Finish(&stepRes, nil)

	assert.True(t, j.finished)
	assert.WithinDuration(t, time.Now(), j.finishTime, time.Millisecond*5)

	assert.Equal(t, &stepRes, j.stepResult)
	assert.Nil(t, j.err)

	// stepResult and err remain unchanged on subsequent calls to Close
	j.Finish(nil, errFinal)
	assert.Equal(t, &stepRes, j.stepResult)
	assert.Nil(t, j.err)
}

func Test_Close_AlreadyFinished(t *testing.T) {
	j, err := New(test.MakeRunRequest(t, "", false))
	require.NoError(t, err)
	defer os.RemoveAll(j.WorkDir)

	j.Finish(&stepRes, nil)

	j.Close()

	assert.Equal(t, &stepRes, j.stepResult)
	assert.Nil(t, j.err)
	assert.WithinDuration(t, time.Now(), j.finishTime, time.Millisecond*5)

	assert.NoDirExists(t, j.TmpDir)
}

func Test_Close(t *testing.T) {
	j, err := New(test.MakeRunRequest(t, "", false))
	require.NoError(t, err)
	defer os.RemoveAll(j.WorkDir)

	j.Close()

	assert.True(t, j.finished)
	assert.Nil(t, j.stepResult)
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
		finishErr   error
		writeErr    error
		wantErr     error
		wantWritten []byte
	}{
		"write error": {
			writeErr:    errors.New("POW!!!"),
			finishErr:   errors.New("BLAMMO!!!"),
			wantErr:     errors.New("POW!!!"),
			wantWritten: data[0][:len(data[0])-1],
		},
		"finish error": {
			finishErr:   context.Canceled,
			wantErr:     context.Canceled,
			wantWritten: bytes.Join(data, nil),
		},

		"no error": {
			wantWritten: bytes.Join(data, nil),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotWritten := bytes.Buffer{}

			rr := test.MakeRunRequest(t, "", false)
			j, err := New(rr)
			require.NoError(t, err)
			defer j.Close()
			defer os.RemoveAll(j.WorkDir)

			go func() {
				for _, d := range data {
					_, err := j.logs.Write(d)
					assert.NoError(t, err)
				}
				j.Finish(nil, tt.finishErr)
			}()

			gotErr := j.FollowLogs(context.Background(), 0, toIOWriter(func(p []byte) (int, error) {
				n, err := gotWritten.Write(p)
				require.NoError(t, err)
				return n, tt.writeErr
			}))

			assert.Equal(t, tt.wantErr, gotErr)
			assert.Equal(t, string(tt.wantWritten), gotWritten.String())
		})
	}
}
