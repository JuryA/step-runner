package test

import (
	"bytes"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gitlab.com/gitlab-org/step-runner/pkg/api/client"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestDirName(t *testing.T) string {
	return path.Join(os.TempDir(), strings.ReplaceAll(t.Name(), "/", "-"))
}

func RandJobID() string {
	return strconv.Itoa(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(9999))
}

func ProtoRunRequest(t *testing.T, step string, withJob bool) *proto.RunRequest {
	runReq := proto.RunRequest{
		Id:    RandJobID(),
		Steps: step,
		Env:   map[string]string{},
	}

	runReq.WorkDir = t.TempDir()
	if withJob {
		runReq.Job = &proto.Job{BuildDir: runReq.WorkDir}
	}

	return &runReq
}

type SyncBuff struct {
	b bytes.Buffer
	sync.RWMutex
}

func (b *SyncBuff) Write(p []byte) (n int, err error) {
	b.Lock()
	defer b.Unlock()
	return b.b.Write(p)
}

func (b *SyncBuff) Len() int {
	b.RLock()
	defer b.RUnlock()
	return b.b.Len()
}

func (b *SyncBuff) String() string {
	b.RLock()
	defer b.RUnlock()
	return b.b.String()
}

type ClosableBuf struct{ SyncBuff }

func (*ClosableBuf) Close() error { return nil }

func RunRequest(t *testing.T, step string, env map[string]string, vars []client.Variable) *client.RunRequest {
	return &client.RunRequest{
		Id: RandJobID(),
		Steps: `spec: {}
---
` + step,
		WorkDir:   "",
		Env:       env,
		Variables: vars,
	}
}
