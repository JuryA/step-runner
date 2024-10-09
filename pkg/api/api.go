package api

import (
	"net"
	"os"
	"path"
)

var defaultSocketPath = path.Join(os.TempDir(), "step-runner.sock")

func ListenSocketPath() string { return defaultSocketPath }

func ListenSocketAddr() *net.UnixAddr { return &net.UnixAddr{Name: ListenSocketPath(), Net: "unix"} }
