package jobs

import (
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

var errFinal = fmt.Errorf("FOO")

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

func Test_Err(t *testing.T) {
	j := Job{}

	assert.Nil(t, j.Err())

	j.err = errFinal
	assert.Equal(t, errFinal, j.Err())
}

func Test_Finish(t *testing.T) {
	runReq := test.MakeRunRequest(t, "", true)
	j, err := New(runReq)
	require.NoError(t, err)
	defer j.Close()
	defer os.RemoveAll(j.WorkDir)

	j.Finish(nil)

	assert.True(t, j.Finished())
	assert.WithinDuration(t, time.Now(), j.finishTime, time.Millisecond*5)

	assert.Nil(t, j.Err())

	// stepResult and err remain unchanged on subsequent calls to Close
	j.Finish(errFinal)
	assert.Nil(t, j.Err())
}

func Test_Close_AlreadyFinished(t *testing.T) {
	j, err := New(test.MakeRunRequest(t, "", false))
	require.NoError(t, err)
	defer os.RemoveAll(j.WorkDir)

	j.Finish(nil)

	j.Close()

	assert.Nil(t, j.Err())
	assert.WithinDuration(t, time.Now(), j.finishTime, time.Millisecond*5)

	assert.NoDirExists(t, j.TmpDir)
}

func Test_Close(t *testing.T) {
	j, err := New(test.MakeRunRequest(t, "", false))
	require.NoError(t, err)
	defer os.RemoveAll(j.WorkDir)

	j.Close()

	assert.True(t, j.Finished())
	assert.True(t, errors.Is(j.err, j.Ctx.Err()))

	assert.NoDirExists(t, j.TmpDir)
}

func Test_FollowStepResults(t *testing.T) {
	tests := map[string]struct {
		finishErr error
		writeErr  error
		wantErr   error
	}{
		"write error": {
			writeErr:  errors.New("POW!!!"),
			finishErr: errors.New("BLAMMO!!!"),
			wantErr:   errors.New("POW!!!"),
		},
		"finish error": {
			finishErr: errors.New("BLAMMO!!!"),
			wantErr:   errors.New("BLAMMO!!!"),
		},
		"no error": {},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			j, err := New(test.MakeRunRequest(t, "", true))
			require.NoError(t, err)
			defer j.Close()
			defer os.RemoveAll(j.WorkDir)

			go func() {
				j.StepResultWriter()(&proto.StepResult{})
				j.Finish(tt.finishErr)
			}()

			gotErr := j.FollowStepResults(context.Background(), 0, func(w *proto.StepResult) error {
				return tt.writeErr
			})

			assert.Equal(t, tt.wantErr, gotErr)
		})
	}
}
