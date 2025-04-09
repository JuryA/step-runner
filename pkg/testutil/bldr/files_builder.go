package bldr

import (
	"io/fs"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type FilesBuilder struct {
	dirs     map[string]os.FileMode
	symlinks map[string]string
	files    map[string]FileData
	t        *testing.T
	baseDir  string
}

type FileData struct {
	data []byte
	perm os.FileMode
}

func Files(t *testing.T) *FilesBuilder {
	return &FilesBuilder{
		dirs:     make(map[string]os.FileMode),
		files:    make(map[string]FileData),
		symlinks: make(map[string]string),
		t:        t,
		baseDir:  t.TempDir(),
	}
}

func (b *FilesBuilder) WithShortBaseDir() *FilesBuilder {
	b.baseDir = filepath.Join(os.TempDir(), strconv.FormatUint(rand.Uint64(), 10))
	err := os.MkdirAll(b.baseDir, 0777)
	require.NoError(b.t, err)

	b.t.Cleanup(func() { _ = os.RemoveAll(b.baseDir) })
	return b
}

func (b *FilesBuilder) WriteDir(dir string) *FilesBuilder {
	b.dirs[filepath.Join(b.baseDir, dir)] = 0755
	return b
}

func (b *FilesBuilder) TouchFile(path string) *FilesBuilder {
	return b.WriteFile(path, "")
}

func (b *FilesBuilder) WriteFile(path string, data any) *FilesBuilder {
	return b.WriteFileWithPerms(path, data, 0644)
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

func (b *FilesBuilder) WriteSymlink(from string, to string) *FilesBuilder {
	b.symlinks[filepath.Join(b.baseDir, from)] = filepath.Join(b.baseDir, to)
	return b
}

func (b *FilesBuilder) Build() string {
	for dir, perm := range b.dirs {
		err := os.MkdirAll(dir, perm)
		require.NoError(b.t, err)
	}

	for filePath, fileData := range b.files {
		dir, _ := path.Split(filePath)

		err := os.MkdirAll(dir, 0755)
		require.NoError(b.t, err)

		err = os.WriteFile(filePath, fileData.data, fileData.perm)
		require.NoError(b.t, err)
	}

	for from, to := range b.symlinks {
		err := os.Symlink(to, from)
		require.NoError(b.t, err)
	}

	return b.baseDir
}

func (b *FilesBuilder) BuildFS() fs.FS {
	dir := b.Build()
	return os.DirFS(dir)
}

func (b *FilesBuilder) BuildPath(path string) string {
	dir := b.Build()
	return filepath.Join(dir, path)
}
