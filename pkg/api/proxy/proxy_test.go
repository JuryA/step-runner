package proxy

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/api"
	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
)

// Starts a dead-simple echoing server that listens on a socket
func setupEchoServer(t *testing.T) func() {
	ln, err := net.ListenUnix("unix", api.SocketAddr(api.DefaultSocketPath()))
	require.NoError(t, err)

	var conn net.Conn
	go func() {
		conn, err = ln.Accept()
		require.NoError(t, err)
		_, err = io.Copy(conn, conn)
		assert.NoError(t, err)
	}()

	return func() {
		assert.NoError(t, ln.Close())
	}
}

func Test_Proxy(t *testing.T) {
	cancel := setupEchoServer(t)
	defer cancel()

	// use a pipe for the pr and pw to pass to the proxy
	pr, pw := io.Pipe()
	buf := test.SyncBuff{}

	// start the proxy...
	go func() {
		conn, err := net.DialUnix("unix", nil, api.SocketAddr(api.DefaultSocketPath()))
		require.NoError(t, err)
		assert.NoError(t, Proxy(pr, &buf, conn))
	}()

	// write some stuff into the writer 1/2 of the pipe...
	for _, p := range [][]byte{
		[]byte("aaaa"),
		[]byte("bbbb"),
		[]byte("cccc"),
	} {
		_, err := pw.Write(p)
		assert.NoError(t, err)
	}
	// wait for the above stuff to be written into the receiving buffer...
	assert.Eventually(t, func() bool { return buf.String() == "aaaabbbbcccc" }, time.Second, time.Millisecond*100)
	// Close the PipeWriter; this will exit the reading loop in the proxy
	assert.NoError(t, pw.Close())
}
