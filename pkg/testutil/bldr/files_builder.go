package bldr

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type FilesBuilder struct {
	files   map[string]FileData
	t       *testing.T
	baseDir string
}

type FileData struct {
	data []byte
	perm os.FileMode
}

func Files(t *testing.T) *FilesBuilder {
	return &FilesBuilder{
		files:   make(map[string]FileData),
		t:       t,
		baseDir: t.TempDir(),
	}
}

func (b *FilesBuilder) TouchFile(path string) *FilesBuilder {
	return b.WriteFile(path, "")
}

func (b *FilesBuilder) WriteFile(path string, data any) *FilesBuilder {
	return b.WriteFileWithPerms(path, data, 0o644)
}

func (b *FilesBuilder) WriteFileWithPerms(path string, data any, perm os.FileMode) *FilesBuilder {
	var contents []byte
	switch v := data.(type) {
	case []byte:
		contents = v
	case string:
		contents = []byte(v)
	default:
		b.t.Fatalf("data must be of type []byte or string, got %T", v)
	}

	b.files[filepath.Join(b.baseDir, path)] = FileData{data: contents, perm: perm}
	return b
}

func (b *FilesBuilder) Build() string {
	for filePath, fileData := range b.files {
		dir, _ := path.Split(filePath)

		err := os.MkdirAll(dir, 0755)
		require.NoError(b.t, err)

		err = os.WriteFile(filePath, fileData.data, fileData.perm)
		require.NoError(b.t, err)
	}

	return b.baseDir
}

func (b *FilesBuilder) BuildFS() fs.FS {
	dir := b.Build()
	return os.DirFS(dir)
}
