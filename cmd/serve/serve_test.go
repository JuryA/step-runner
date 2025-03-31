package serve

import (
	"context"
	"os"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func Test_run(t *testing.T) {
	t.Run("invalid socket path", func(t *testing.T) {
		sigChan := make(chan os.Signal, 1)
		assert.ErrorContains(t, run(nil, []string{"/foo/bar/baz"}, sigChan), "invalid socket dir")
	})

	t.Run("empty socket path", func(t *testing.T) {
		sigChan := make(chan os.Signal, 1)
		assert.ErrorContains(t, run(nil, []string{""}, sigChan), "invalid empty socket dir")
	})

	t.Run("no socket path", func(t *testing.T) {
		sigChan := make(chan os.Signal, 1)
		runService(t, nil, sigChan)
	})

	t.Run("valid socket path", func(t *testing.T) {
		sigChan := make(chan os.Signal, 1)
		runService(t, []string{t.TempDir()}, sigChan)
	})
}

func runService(t *testing.T, args []string, sigChan chan os.Signal) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	// run the service.
	go func() {
		defer wg.Done()
		assert.NoError(t, run(nil, args, sigChan))
	}()

	// create a client.
	cl := newSRClient(t, args)

	// execute a simple command to ensure the service is running.
	_, err := cl.Status(context.Background(), &proto.StatusRequest{})
	assert.NoError(t, err)

	// shut it all down.
	sigChan <- syscall.SIGTERM
	wg.Wait()
}

func newSRClient(t *testing.T, args []string) proto.StepRunnerClient {
	sockAddr, err := getSocketAddr(args)
	require.NoError(t, err)

	cliConn, err := grpc.NewClient("unix:"+sockAddr.Name,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return proto.NewStepRunnerClient(cliConn)
}
