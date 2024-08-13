package proxy

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/api"
	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/test"
)

// Starts a dead-simple echoing server that listens on a socket
func setupEchoServer(t *testing.T) func() {
	ln, err := net.Listen("unix", api.DefaultSocketPath())
	require.NoError(t, err)

	var conn net.Conn
	go func() {
		conn, err = ln.Accept()
		require.NoError(t, err)
		_, _ = io.Copy(conn, conn)
	}()

	return func() {
		conn.Close()
		ln.Close()
	}
}

func Test_Proxy(t *testing.T) {
	cancel := setupEchoServer(t)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	// use a pipe for the pr and pw to pass to the proxy
	pr, pw := io.Pipe()
	buf := test.SyncBuff{}

	// start the proxy...
	go func() {
		defer wg.Done()
		conn, err := net.Dial("unix", api.DefaultSocketPath())
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
	// Close the server connection; this will exit the writing loop in the proxy
	cancel()
	time.Sleep(time.Millisecond)
	// Close the PipeWriter; this will exit the reading loop in the proxy
	pw.Close()

	wg.Wait()
	assert.Equal(t, "aaaabbbbcccc", buf.String())
}
