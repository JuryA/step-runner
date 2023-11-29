package cache

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gitlab.com/gitlab-org/step-runner/pkg/step"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Definitions struct {
	mux      sync.Mutex
	cacheDir string
	entries  map[string]*entry
}

type entry struct {
	spec *proto.Spec
	def  *proto.Definition
	dir  string
}

func New() (*Definitions, error) {
	cacheDir := os.TempDir() + string(os.PathSeparator) + "step-runner-cache-" + strconv.Itoa(int(rand.Uint32()))
	err := os.Mkdir(cacheDir, 0750)
	if err != nil {
		return nil, fmt.Errorf("making cache dir %q: %w", cacheDir, err)
	}
	return &Definitions{
		cacheDir: cacheDir,
		entries:  map[string]*entry{},
	}, nil
}

func (d *Definitions) Cleanup() {
	os.RemoveAll(d.cacheDir)
}

func (d *Definitions) Get(step string) (*proto.Spec, *proto.Definition, string, error) {
	d.mux.Lock()
	defer d.mux.Unlock()
	var err error
	e, ok := d.entries[step]
	if !ok {
		e, err = d.cacheMiss(step)
		if err != nil {
			return nil, nil, "", fmt.Errorf("fetching step %q: %w", step, err)
		}
	}
	return e.spec, e.def, e.dir, nil
}

func (d *Definitions) cacheMiss(s string) (*entry, error) {
	switch {
	case strings.HasPrefix(s, "."):
		return d.fetchLocal(s)
	case strings.HasPrefix(s, "https+git"):
		return d.fetchGit(s)
	default:
		return nil, fmt.Errorf("invalid step reference: %v", s)
	}
}

func (d *Definitions) fetchLocal(s string) (*entry, error) {
	path, err := filepath.Abs(s)
	if err != nil {
		return nil, fmt.Errorf("resolving path %q: %w", s, err)
	}
	return load(path)
}

func (d *Definitions) fetchGit(s string) (*entry, error) {
	dir := d.cacheDir + string(os.PathSeparator) + strconv.Itoa(int(rand.Uint32()))
	err := os.Mkdir(dir, 0750)
	if err != nil {
		return nil, fmt.Errorf("making dir for cloning: %w", err)
	}
	s = strings.Replace(s, "https+git", "https", 1)
	err = execIn(dir, "git", "clone", s)
	if err != nil {
		return nil, fmt.Errorf("cloning %q: %w", s, err)
	}
	folder, err := probablyFolder(s)
	if err != nil {
		return nil, fmt.Errorf("couldn't figure out the folder in %q: %w", s, err)
	}
	return load(dir + string(os.PathSeparator) + folder)
}

func execIn(dir string, c string, args ...string) error {
	cmd := exec.Command(c, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v", err, string(out))
	}
	if cmd.ProcessState.ExitCode() != 0 {
		return fmt.Errorf("exit code %v: %v", cmd.ProcessState.ExitCode(), string(out))
	}
	return nil
}

func load(dir string) (*entry, error) {
	filename := dir + string(os.PathSeparator) + "step.yml"
	spec, def, err := step.LoadSpecDef(filename)
	if err != nil {
		return nil, fmt.Errorf("loading file %q: %w", dir, err)
	}
	return &entry{
		spec: spec,
		def:  def,
		dir:  dir,
	}, nil
}

func probablyFolder(s string) (string, error) {
	// TODO implement `go get` protocol to support steps in subfolders
	fields := strings.Split(s, "//")
	if len(fields) != 2 {
		return "", fmt.Errorf("need exactly protocol//host/folder")
	}
	fields = strings.Split(fields[1], "/")
	return fields[len(fields)-1], nil
}
