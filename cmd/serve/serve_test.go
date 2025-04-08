package serve_test

import (
	"net"
	"os"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"gitlab.com/gitlab-org/step-runner/cmd/serve"
	"gitlab.com/gitlab-org/step-runner/pkg/api"
	"gitlab.com/gitlab-org/step-runner/pkg/di"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func TestGetSocketAddr(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		expectAddr string
		expectErr  string
	}{
		{
			name:      "invalid socket path",
			args:      []string{"/foo/bar/baz"},
			expectErr: "invalid socket dir",
		},
		{
			name:      "empty socket path",
			args:      []string{""},
			expectErr: "invalid empty socket dir",
		},
		{
			name:       "absent socket path uses default",
			args:       nil,
			expectAddr: "step-runner.sock",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			socketAddr, err := serve.GetSocketAddr(test.args)
			if test.expectErr == "" {
				require.Contains(t, socketAddr.String(), test.expectAddr)
			} else {
				require.ErrorContains(t, err, test.expectErr)
			}
		})
	}
}

func TestRun(t *testing.T) {
	stepRunnerService, err := di.NewContainer().StepRunnerService()
	require.NoError(t, err)

	// a short path length is required otherwise socket path length can be exceeded on osx causing the test to fail
	socketFile := bldr.Files(t).WithShortBaseDir().BuildPath("step-runner.sock")
	socketAddr := api.SocketAddr(socketFile)
	sigChan := make(chan os.Signal, 1)
	serveCmd := serve.NewServeCmd(stepRunnerService, socketAddr, sigChan)

	wg := sync.WaitGroup{}
	wg.Add(1)

	// wait for the command to bind to the port for test stability on all platforms
	listener, err := serveCmd.Listen()
	require.NoError(t, err)

	// run the service
	go func() {
		defer wg.Done()
		assert.NoError(t, serveCmd.Serve(listener))
	}()

	// create a client
	cl := newSRClient(t, socketAddr)

	// execute a simple command to ensure the service is running
	status, err := cl.Status(bldr.DefaultCtx(t), &proto.StatusRequest{})
	assert.NoError(t, err)
	require.Len(t, status.Jobs, 0)

	// shut it all down
	sigChan <- syscall.SIGTERM
	wg.Wait()
}

func newSRClient(t *testing.T, socketAddr *net.UnixAddr) proto.StepRunnerClient {
	cliConn, err := grpc.NewClient("unix:"+socketAddr.Name, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return proto.NewStepRunnerClient(cliConn)
}
