package bldr

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

type tcpPort struct {
	t *testing.T
}

func TCPPort(t *testing.T) *tcpPort {
	return &tcpPort{t: t}
}

func (p *tcpPort) FindFree() string {
	l, port := p.Listen("0")
	require.NoError(p.t, l.Close())
	return port
}

func (p *tcpPort) Listen(port string) (net.Listener, string) {
	tcpListener, err := net.Listen("tcp", ":"+port)
	require.NoError(p.t, err, "failed to allocate TCP port")

	_, portPart, err := net.SplitHostPort(tcpListener.Addr().String())
	require.NoError(p.t, err, "failed to split host and port")

	return tcpListener, portPart
}
