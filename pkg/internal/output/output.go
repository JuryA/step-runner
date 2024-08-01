package output

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

const (
	outputFilename = "output"
)

type Files struct {
	stepCtx      *runner.StepsContext
	outputMethod proto.OutputMethod
	specOutputs  map[string]*proto.Spec_Content_Output

	dir        string
	outputFile string
}

func New(
	stepCtx *runner.StepsContext,
	outputMethod proto.OutputMethod,
	specOutputs map[string]*proto.Spec_Content_Output,
) (*Files, error) {
	dir, err := os.MkdirTemp("", "step-runner-output-*")
	if err != nil {
		return nil, fmt.Errorf("making output directory: %w", err)
	}
	outputFile := filepath.Join(dir, outputFilename)
	err = os.WriteFile(outputFile, []byte{}, 0660)
	if err != nil {
		return nil, fmt.Errorf("creating output file: %w", err)
	}
	stepCtx.OutputFile = outputFile
	return &Files{
		stepCtx:      stepCtx,
		outputMethod: outputMethod,
		specOutputs:  specOutputs,
		dir:          dir,
		outputFile:   outputFile,
	}, nil
}

func (f *Files) OutputTo(result *proto.StepResult) error {

	// Delegates take over step execution including reading and
	// validating outputs.
	if f.outputMethod == proto.OutputMethod_delegate {
		return f.mergeDelegateOutput(result)
	}

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

// mergeDelegateOutput reifies the delegate output as a step result
// and merges with the given step result.
func (f *Files) mergeDelegateOutput(result *proto.StepResult) error {

	// Delegate return a full step result in the output file.
	delegateResult := &proto.StepResult{}
	data, err := os.ReadFile(f.outputFile)
	if err != nil {
		return fmt.Errorf("reading file %v: %w", f.outputFile, err)
	}
	if err := json.Unmarshal(data, delegateResult); err != nil {
		return fmt.Errorf("reading output_file as a step result: %w", err)
	}

	// Merge outputs only. Environment variables should be written
	// to the environment file and will be exported by ExportTo in
	// the usual way.
	if result.Outputs == nil {
		result.Outputs = map[string]*structpb.Value{}
	}

	// Outputs are taken as-is. They will not match the
	// outputs declared by the calling step. The delegate
	// step has already verified the outputs when
	// producing the step result.
	maps.Copy(result.Outputs, delegateResult.Outputs)

	// Merge the delegate step result as a sub-step to give an
	// accurate representation of the execution trace.
	result.SubStepResults = append(result.SubStepResults, delegateResult)

	return nil
}

func checkOutputType(want proto.ValueType, have *structpb.Value) error {
	switch want {
	case proto.ValueType_boolean:
		if _, ok := have.Kind.(*structpb.Value_BoolValue); ok {
			return nil
		}
	case proto.ValueType_array:
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
