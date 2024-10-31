package runner

import (
	"bufio"
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"google.golang.org/protobuf/encoding/protojson"

	"gitlab.com/gitlab-org/step-runner/proto"
)

var filesDirMutex = sync.Mutex{}
var filesDir string

type StepFile struct {
	path string
}

func NewStepFileInTmp() (*StepFile, error) {
	filesDirMutex.Lock()
	defer filesDirMutex.Unlock()

	if filesDir == "" {
		var err error
		filesDir, err = os.MkdirTemp(os.TempDir(), "step-runner-*")

		if err != nil {
			return nil, fmt.Errorf("failed to create step file: failed to create temporary dir: %w", err)
		}
	}

	return NewStepFileInDir(filesDir)
}

func NewStepFileInDir(dir string) (*StepFile, error) {
	path := filepath.Join(dir, fmt.Sprintf("step-file-%d", rand.Uint32()))

	if err := os.WriteFile(path, []byte{}, 0660); err != nil {
		return nil, fmt.Errorf("failed to create step file: %w", err)
	}

	return NewStepFile(path), nil
}

func NewStepFile(path string) *StepFile {
	return &StepFile{
		path: path,
	}
}

func (s *StepFile) Path() string {
	return s.path
}

func (s *StepFile) ReadDotEnv() (map[string]string, error) {
	dotenv, err := godotenv.Read(s.path)

	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	return dotenv, nil
}

func (s *StepFile) ReadKeyValueLines() (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("reading file %v: %w", s.path, err)
	}

	out := map[string]string{}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 {
			continue
		}

		fields := strings.SplitN(line, "=", 2)

		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid line %q", line)
		}

		out[fields[0]] = fields[1]
	}

	return out, scanner.Err()
}

func (s *StepFile) ReadStepResult() (*proto.StepResult, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("reading file %v: %w", s.path, err)
	}

	stepResult := &proto.StepResult{}
	if err := protojson.Unmarshal(data, stepResult); err != nil {
		return nil, fmt.Errorf("reading output_file as a step result: %w", err)
	}

	return stepResult, nil
}

func (s *StepFile) Remove() error {
	err := os.Remove(s.path)

	if err != nil {
		return fmt.Errorf("failed to remove step file %s: %w", s.path, err)
	}

	return nil
}
