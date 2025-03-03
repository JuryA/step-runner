package oci

import (
	"fmt"
	"io/fs"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type Artifact struct {
	From     string
	To       string
	Platform *v1.Platform
}

func NewArtifact(platform *v1.Platform, from, to string) *Artifact {
	return &Artifact{
		From:     from,
		To:       to,
		Platform: platform,
	}
}

func (a *Artifact) FS() fs.FS {
	return os.DirFS(a.From)
}

func (a *Artifact) String() string {
	return fmt.Sprintf("%s[%s->%s]", a.Platform, a.From, a.To)
}
