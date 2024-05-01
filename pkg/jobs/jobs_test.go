package jobs

import (
	"errors"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/step-runner/proto"
)

var (
	stepRes  = proto.StepResult{ExitCode: 123456}
	errFinal = fmt.Errorf("FOO")
)

func makeRunRequest(t *testing.T, withJob bool) *proto.RunRequest {
	testDir := path.Join(os.TempDir(), t.Name())
	runReq := proto.RunRequest{
		Id:  "853",
		Env: map[string]string{},
	}

	if withJob {
		runReq.Job = &proto.Job{BuildDir: testDir}
	} else {
		runReq.WorkDir = testDir
	}

	return &runReq
}

func Test_New(t *testing.T) {
	runReq := makeRunRequest(t, false)
	j, err := New(runReq)
	require.NoError(t, err)

	defer os.RemoveAll(j.TmpDir)
	defer os.RemoveAll(j.WorkDir)

	assert.Equal(t, runReq.Id, j.ID)
	assert.DirExists(t, j.TmpDir)
	assert.DirExists(t, j.WorkDir)
}

func Test_New_Job_WorkDir(t *testing.T) {
	runReq := makeRunRequest(t, true)
	j, err := New(runReq)
	require.NoError(t, err)

	defer os.RemoveAll(j.WorkDir)
	defer os.RemoveAll(j.TmpDir)

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
	j := Job{}

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

func Test_Finalize_AlreadyFinished(t *testing.T) {
	j, err := New(makeRunRequest(t, false))
	require.NoError(t, err)
	defer os.RemoveAll(j.WorkDir)

	j.Finish(&stepRes, nil)

	j.Close()

	assert.Equal(t, &stepRes, j.stepResult)
	assert.Nil(t, j.err)
	assert.WithinDuration(t, time.Now(), j.finishTime, time.Millisecond*5)

	assert.NoDirExists(t, j.TmpDir)
}

func Test_Finalize(t *testing.T) {
	j, err := New(makeRunRequest(t, false))
	require.NoError(t, err)
	defer os.RemoveAll(j.WorkDir)

	j.Close()

	assert.True(t, j.finished)
	assert.Nil(t, j.stepResult)
	assert.True(t, errors.Is(j.err, j.Ctx.Err()))

	assert.NoDirExists(t, j.TmpDir)
}
