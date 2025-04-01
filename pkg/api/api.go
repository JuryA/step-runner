package api

import (
	"net"
	"os"
	"path"
)

var defaultSocketPath = path.Join(os.TempDir(), "step-runner.sock")

func DefaultSocketPath() string { return defaultSocketPath }

func SocketAddr(socketPath string) *net.UnixAddr {
	return &net.UnixAddr{Name: socketPath, Net: "unix"}
}
