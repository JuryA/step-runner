package test

import (
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/exp/rand"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestDirName(t *testing.T) string {
	return path.Join(os.TempDir(), strings.ReplaceAll(t.Name(), "/", "-"))
}

func RandJobID() string { return strconv.Itoa(rand.Intn(999)) }

func ProtoRunRequest(t *testing.T, step string, withJob bool) *proto.RunRequest {
	testDir := TestDirName(t)
	runReq := proto.RunRequest{
		Id:    RandJobID(),
		Steps: step,
		Env:   map[string]string{},
	}

	if withJob {
		runReq.Job = &proto.Job{BuildDir: testDir}
	} else {
		runReq.WorkDir = testDir
	}

	return &runReq
}
