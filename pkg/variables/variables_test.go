package variables

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func Test_Variable_Regular(t *testing.T) {
	pv := proto.Variable{
		Key:   "k",
		Value: "v",
	}
	v := Variable{
		v:       &pv,
		tmpPath: os.TempDir(),
	}

	assert.Equal(t, pv.Key, v.Key())
	assert.Equal(t, pv.Value, v.Value())
	assert.False(t, v.File())
	assert.ErrorContains(t, v.Write(), "is not a file variable")
}

func Test_Variable_File(t *testing.T) {
	pv := proto.Variable{
		Key:   "k",
		Value: "v",
		File:  true,
	}

	tmp := "blammo"

	v := Variable{
		v:       &pv,
		tmpPath: tmp,
	}

	assert.Equal(t, pv.Key, v.Key())
	assert.Equal(t, path.Join(tmp, pv.Key), v.Value())
	assert.True(t, v.File())
}

func Test_Variable_Write_Regular(t *testing.T) {
	pv := proto.Variable{
		Key:   "k",
		Value: "v",
	}

	v := Variable{
		v:       &pv,
		tmpPath: "blammo",
	}

	assert.ErrorContains(t, v.Write(), "is not a file variable")
}

func Test_Variable_Write_File(t *testing.T) {
	pv := proto.Variable{
		Key:   "k",
		Value: "v",
		File:  true,
	}

	tmp, err := os.MkdirTemp("", t.Name()+"_")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	v := Variable{
		v:       &pv,
		tmpPath: tmp,
	}

	require.NoError(t, v.Write())
	assert.FileExists(t, v.Value())
	data, err := os.ReadFile(v.Value())
	require.NoError(t, err)
	assert.Equal(t, pv.Value, string(data))
}

func Test_New(t *testing.T) {
	pvs := []*proto.Variable{
		{Key: "a", Value: "A"},
		{Key: "b", Value: "B"},
		{Key: "c", Value: "C", File: true},
	}

	tmp := "blammo"
	vs := New(pvs, tmp)
	assert.Len(t, vs, len(pvs))

	for i, v := range vs {
		assert.Equal(t, pvs[i].Key, v.Key())
		if v.File() {
			assert.Equal(t, path.Join(tmp, pvs[i].Key), v.Value())
		} else {
			assert.Equal(t, pvs[i].Value, v.Value())
		}
	}
}

func Test_Variables_Write(t *testing.T) {
	pvs := []*proto.Variable{
		{Key: "a", Value: "A"},
		{Key: "b", Value: "B"},
		{Key: "c", Value: "C", File: true},
	}

	tmp, err := os.MkdirTemp("", t.Name()+"_")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	vs := New(pvs, tmp)
	assert.Len(t, vs, len(pvs))

	require.NoError(t, vs.Write())

	for i, v := range vs {
		if v.File() {
			assert.FileExists(t, v.Value())
			data, err := os.ReadFile(v.Value())
			require.NoError(t, err)
			assert.Equal(t, pvs[i].Value, string(data))
		} else {
			assert.NoFileExists(t, v.Value())
			assert.NoFileExists(t, path.Join(tmp, v.Value()))
			assert.NoFileExists(t, path.Join(tmp, pvs[i].Value))
		}
	}
}

func Test_Prepare(t *testing.T) {
	pvs := []*proto.Variable{
		{Key: "a", Value: "A"},
		{Key: "b", Value: "B"},
		{Key: "c", Value: "C", File: true},
	}

	tmp, err := os.MkdirTemp("", t.Name()+"_")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	vs, err := Prepare(&proto.Job{Variables: pvs}, tmp)
	require.NoError(t, err)

	assert.Len(t, vs, len(pvs))

	for _, pv := range pvs {
		assert.Contains(t, vs, pv.Key)
		v := vs[pv.Key]
		if pv.File {
			assert.Equal(t, path.Join(tmp, pv.Key), v)
			assert.FileExists(t, v)
		} else {
			assert.Equal(t, pv.Value, v)
			assert.NoFileExists(t, v)
		}

	}

	vs, err = Prepare(nil, tmp)
	require.NoError(t, err)
	require.Len(t, vs, 0)
}

func Test_Expand(t *testing.T) {
	env := map[string]string{
		"FOO":    "foo",
		"BAR":    "bar",
		"BAZ":    "$FOO/$BAR",
		"BLAMMO": "${FOO}/${BAR}/baz",
	}

	expanded := Expand(env)

	assert.Equal(t, "foo", expanded["FOO"])
	assert.Equal(t, "bar", expanded["BAR"])
	assert.Equal(t, "foo/bar", expanded["BAZ"])
	assert.Equal(t, "foo/bar/baz", expanded["BLAMMO"])
}
