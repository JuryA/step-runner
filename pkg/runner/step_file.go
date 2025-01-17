package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

var filesDirMutex = sync.Mutex{}
var filesDir string

const lineErrMsg = "failed to unmarshal JSON at line"

type StepFileLine struct {
	Name  *string         `json:"name"`
	Value *structpb.Value `json:"value"`
}

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
	return &StepFile{path: path}
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

func (s *StepFile) ReadKeyValueLines() (map[string]*structpb.Value, error) {
	file, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("opening file %v: %w", s.path, err)
	}
	defer file.Close()

	out := map[string]*structpb.Value{}
	reader := bufio.NewReader(file)
	for i := 1; true; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		errCtx := NewErrorCtx("line", []byte(line))

		if len(line) == 0 {
			continue
		}

		v := &StepFileLine{}
		if err = json.Unmarshal([]byte(line), v); err != nil {
			return nil, errCtx.Errorf("%s %d: %w", lineErrMsg, i, err)
		}

		if v.Name == nil && !strings.Contains(string(line), `"name"`) {
			return nil, errCtx.Errorf(`%s %d: "name" field is missing`, lineErrMsg, i)
		}

		if v.Name == nil {
			return nil, errCtx.Errorf(`%s %d: "name" field value is null`, lineErrMsg, i)
		}

		if *v.Name == "" {
			return nil, errCtx.Errorf(`%s %d: "name" field value is empty`, lineErrMsg, i)
		}

		if v.Value == nil && !strings.Contains(string(line), `"value"`) {
			return nil, errCtx.Errorf(`%s %d: "value" field is missing`, lineErrMsg, i)
		}

		if v.Value == nil {
			return nil, errCtx.Errorf(`%s %d: "value" field value is null`, lineErrMsg, i)
		}

		out[*v.Name] = v.Value
	}

	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	return out, nil
}

func (s *StepFile) ReadStepResult() (*proto.StepResult, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("reading file %v: %w", s.path, err)
	}

	stepResult := &proto.StepResult{}
	if err := protojson.Unmarshal(data, stepResult); err != nil {
		return nil, NewErrorCtx("output file data", data).Errorf("reading output_file as a step result: %w", []any{err}...)
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

func (s *StepFile) ReadEnvironment() (*Environment, error) {
	outputs, err := s.ReadKeyValueLines()

	if err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}

	env := make(map[string]string)

	for key, value := range outputs {
		switch value.GetKind().(type) {
		case *structpb.Value_BoolValue:
			env[key] = strconv.FormatBool(value.GetBoolValue())
		case *structpb.Value_NumberValue:
			env[key] = strconv.FormatFloat(value.GetNumberValue(), 'f', -1, 64)
		case *structpb.Value_StringValue:
			env[key] = value.GetStringValue()
		default:
			return nil, fmt.Errorf("read env file: key %q: cannot convert value type %q to string", key, structpbValueToTypeName(value))
		}
	}

	return NewEnvironment(env), nil
}

func (s *StepFile) ReadValues(specOutputs map[string]*proto.Spec_Content_Output) (map[string]*structpb.Value, error) {
	keyValues, err := s.ReadKeyValueLines()
	if err != nil {
		return nil, fmt.Errorf("read output file: %w", err)
	}
	for key, value := range keyValues {
		outputSpec, ok := specOutputs[key]
		if !ok {
			return nil, fmt.Errorf("read output file: key %q: unexpected output, remove from step outputs or define in step specification", key)
		}

		if err := s.checkOutputType(outputSpec.Type, value); err != nil {
			return nil, fmt.Errorf("read output file: key %q: %w", key, err)
		}
	}

	for name, o := range specOutputs {
		if _, ok := keyValues[name]; ok {
			continue
		}
		if o.Default != nil {
			keyValues[name] = o.Default
			continue
		}
		return nil, fmt.Errorf("read output file: key %q: missing output, add to step outputs or remove from step specification", name)
	}

	return keyValues, nil
}

func (s *StepFile) checkOutputType(want proto.ValueType, have *structpb.Value) error {
	wantType := want.String()
	haveType := structpbValueToTypeName(have)

	if wantType != haveType {
		return fmt.Errorf("mismatched types, declared as %q in step specification and received from step as type %q", wantType, haveType)
	}

	return nil
}

func structpbValueToTypeName(value *structpb.Value) string {
	switch value.GetKind().(type) {
	case *structpb.Value_BoolValue:
		return "boolean"
	case *structpb.Value_ListValue:
		return "array"
	case *structpb.Value_NumberValue:
		return "number"
	case *structpb.Value_StringValue:
		return "string"
	case *structpb.Value_StructValue:
		return "struct"
	case *structpb.Value_NullValue:
		return "null"
	default:
		return "unknown"
	}
}
