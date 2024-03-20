package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

const (
	outputFilename = "output"
	exportFilename = "export"
)

type Files struct {
	stepCtx     *context.Steps
	specOutputs map[string]*proto.Spec_Content_Output

	dir        string
	outputFile string
	exportFile string
}

func New(stepCtx *context.Steps, specOutputs map[string]*proto.Spec_Content_Output) (*Files, error) {
	dir, err := os.MkdirTemp("", "step-runner-output-*")
	if err != nil {
		return nil, fmt.Errorf("making output directoy: %w", err)
	}
	outputFile := filepath.Join(dir, outputFilename)
	err = os.WriteFile(outputFile, []byte{}, 0660)
	if err != nil {
		return nil, fmt.Errorf("creating output file: %w", err)
	}
	exportFile := filepath.Join(dir, exportFilename)
	err = os.WriteFile(exportFile, []byte{}, 0660)
	if err != nil {
		return nil, fmt.Errorf("creating export file: %w", err)
	}
	stepCtx.OutputFile = outputFile
	stepCtx.ExportFile = exportFile
	return &Files{
		stepCtx:     stepCtx,
		specOutputs: specOutputs,
		dir:         dir,
		outputFile:  outputFile,
		exportFile:  exportFile,
	}, nil
}

func (f *Files) OutputTo(result *proto.StepResult) error {
	outputs, err := readFile(f.outputFile)
	if err != nil {
		return fmt.Errorf("reading outputs: %w", err)
	}
	protoOutputs := map[string]*structpb.Value{}
	for k, v := range outputs {
		outputSpec, ok := f.specOutputs[k]
		if !ok {
			return fmt.Errorf("output %q received from step is not declared in spec", k)
		}
		if outputSpec.Type == proto.ValueType_raw_string {
			protoOutputs[k] = structpb.NewStringValue(v)
			continue
		}
		var outputJSON any
		err := json.Unmarshal([]byte(v), &outputJSON)
		if err != nil {
			return fmt.Errorf("unmarshaling output %q as json: %w", k, err)
		}
		protoV, err := structpb.NewValue(outputJSON)
		if err != nil {
			return err
		}
		err = checkOutputType(outputSpec.Type, protoV)
		if err != nil {
			return err
		}
		protoOutputs[k] = protoV
	}
	for k, o := range f.specOutputs {
		if _, ok := protoOutputs[k]; ok {
			continue
		}
		if o.Default != nil {
			protoOutputs[k] = o.Default
			continue
		}
		return fmt.Errorf("output %q was declared by spec but not received from step", k)
	}
	result.Outputs = protoOutputs
	return nil
}

func checkOutputType(want proto.ValueType, have *structpb.Value) error {
	switch want {
	case proto.ValueType_bool:
		if _, ok := have.Kind.(*structpb.Value_BoolValue); ok {
			return nil
		}
	case proto.ValueType_list:
		if _, ok := have.Kind.(*structpb.Value_ListValue); ok {
			return nil
		}
	case proto.ValueType_number:
		if _, ok := have.Kind.(*structpb.Value_NumberValue); ok {
			return nil
		}
	case proto.ValueType_string:
		if _, ok := have.Kind.(*structpb.Value_StringValue); ok {
			return nil
		}
	case proto.ValueType_struct:
		if _, ok := have.Kind.(*structpb.Value_StructValue); ok {
			return nil
		}
	default:
		return fmt.Errorf("unsupported output type: %v", want)
	}
	return fmt.Errorf("declared output type %v and type %T received from step must match", want, have.Kind)
}

func (f *Files) ExportTo(globalCtx *context.Global, result *proto.StepResult) error {
	exports, err := godotenv.Read(f.exportFile)
	if err != nil {
		return fmt.Errorf("reading exports: %w", err)
	}
	for k, v := range exports {
		globalCtx.Env[k] = v
	}
	result.Exports = exports
	return nil
}

func (f *Files) Cleanup() {
	os.RemoveAll(f.dir)
}

func readFile(filename string) (map[string]string, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file %v: %w", filename, err)
	}
	out := map[string]string{}
	lines := strings.Split(string(bytes), "\n")
	for _, l := range lines {
		if len(l) == 0 {
			continue
		}
		fields := strings.Split(l, "=")
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid line %q", l)
		}
		key := fields[0]
		value := l[len(key)+1:]
		out[key] = value
	}
	return out, nil
}
