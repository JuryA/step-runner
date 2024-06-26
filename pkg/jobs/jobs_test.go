package jobs

import (
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
