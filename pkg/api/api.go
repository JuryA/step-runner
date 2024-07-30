package api

import (
	"os"
	"path"
)

var defaultSocketPath = path.Join(os.TempDir(), "step-runner.sock")

func DefaultSocketPath() string { return defaultSocketPath }
