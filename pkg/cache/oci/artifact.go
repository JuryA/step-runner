package oci

import (
	"io/fs"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type Artifact struct {
	dir      string
	platform *v1.Platform
}

func NewArtifact(dir string, platform *v1.Platform) *Artifact {
	if platform == nil {
		panic("artifact must have a platform")
	}

	return &Artifact{
		dir:      dir,
		platform: platform,
	}
}

func (a *Artifact) DirFS() fs.FS {
	return os.DirFS(a.dir)
}

func (a *Artifact) String() string {
	return a.dir
}
