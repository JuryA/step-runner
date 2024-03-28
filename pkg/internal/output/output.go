package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/encoding/protojson"
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

	// Delegates take over step execution including reading and
	// validating outputs.
	if f.isDelegate() {
		return f.mergeDelegateOutput(result, outputs)
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

// isDelegate detects a step which returns a single output of type
// step_result. This is the signature of a delegate step which has
// taken over step execution. A delegate step's results should be
// incorporated directly into the step result tree.
func (f *Files) isDelegate() bool {
	return len(f.specOutputs) == 1 && maps.Values(f.specOutputs)[0].Type == proto.ValueType_step_result
}

// mergeDelegateOutput reifies the delegate output as a step result
// and merges with the given step result.
func (f *Files) mergeDelegateOutput(result *proto.StepResult, delegateOutputs map[string]string) error {

	// Find and unmarshal the delegate result
	delegateResult := &proto.StepResult{}
	delegateResultFound := false
	for k, v := range delegateOutputs {
		outputSpec, ok := f.specOutputs[k]
		if !ok {
			return fmt.Errorf("output %q received from step is not declared in spec", k)
		}
		if delegateResultFound {
			return fmt.Errorf("output %q emitted more than once. unsupported for type step_result", k)
		}
		if outputSpec.Type != proto.ValueType_step_result {
			// Checked already in isDelegate but this makes it clear
			return fmt.Errorf("output %q must be type step_result", k)
		}
		err := protojson.Unmarshal([]byte(v), delegateResult)
		if err != nil {
			return fmt.Errorf("unmarshaling output %q as step result: %w", k, err)
		}
		delegateResultFound = true
	}
	// isDelegate already verified there is only one which is of
	// type step_result
	if !delegateResultFound && maps.Values(f.specOutputs)[0].Type == proto.ValueType_step_result {
		return fmt.Errorf("output %q (type step_result) was declared by spec but not received from step", maps.Keys(f.specOutputs)[0])
	}

	// Merge outputs only. Environment variables should be written
	// to the environment file and will be exported by ExportTo in
	// the usual way.
	if result.Outputs == nil {
		result.Outputs = map[string]*structpb.Value{}
	}
	for k, v := range delegateResult.Outputs {
		// Outputs are take as-is. They will not match the
		// outputs declared by the calling step. The delegate
		// step has already verified the outputs when
		// producing the step result.
		result.Outputs[k] = v
	}

	// Merge the delegate step result as a child to give an
	// accurate representation of the execution trace.
	result.ChildrenStepResults = append(result.ChildrenStepResults, delegateResult)

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
