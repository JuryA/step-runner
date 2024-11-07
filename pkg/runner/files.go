package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

const (
	outputFilename  = "output"
	contextFilename = "context"
)

type Files struct {
	stepCtx      *StepsContext
	outputMethod proto.OutputMethod
	specOutputs  map[string]*proto.Spec_Content_Output

	dir         string
	outputFile  string
	contextFile string
}

func NewFiles(
	stepCtx *StepsContext,
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
	// We want to provide a context file because serializing whole
	// chunks of context (e.g. the environment) and passing as a
	// string becomes an escaping nightmare. We do something
	// similar for the custom executor by providing the job
	// response (though for different reasons):
	// https://docs.gitlab.com/runner/executors/custom.html#job-response
	contextFile := filepath.Join(dir, contextFilename)
	bytes, err := protojson.Marshal(stepCtx.Proto())
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(contextFile, bytes, 0660)
	if err != nil {
		return nil, fmt.Errorf("creating context file: %w", err)
	}
	stepCtx.ContextFile = contextFile
	return &Files{
		stepCtx:      stepCtx,
		outputMethod: outputMethod,
		specOutputs:  specOutputs,
		dir:          dir,
		outputFile:   outputFile,
		contextFile:  contextFile,
	}, nil
}

func (f *Files) Outputs() (map[string]*structpb.Value, *proto.StepResult, error) {
	// Delegates take over step execution including reading and validating outputs.
	if f.outputMethod == proto.OutputMethod_delegate {
		delegateResult, err := f.loadStepResultFromOutputFile()

		if err != nil {
			return nil, nil, fmt.Errorf("reading outputs: %w", err)
		}

		return delegateResult.Outputs, delegateResult, nil
	}

	outputs, err := readFile(f.outputFile)
	if err != nil {
		return nil, nil, fmt.Errorf("reading outputs: %w", err)
	}

	protoOutputs := map[string]*structpb.Value{}
	for k, v := range outputs {
		outputSpec, ok := f.specOutputs[k]
		if !ok {
			return nil, nil, fmt.Errorf("output %q received from step is not declared in spec", k)
		}
		if outputSpec.Type == proto.ValueType_raw_string {
			protoOutputs[k] = structpb.NewStringValue(v)
			continue
		}
		var outputJSON any
		err := json.Unmarshal([]byte(v), &outputJSON)
		if err != nil {
			return nil, nil, fmt.Errorf("unmarshaling output %q as json: %w", k, err)
		}
		protoV, err := structpb.NewValue(outputJSON)
		if err != nil {
			return nil, nil, err
		}
		err = checkOutputType(outputSpec.Type, protoV)
		if err != nil {
			return nil, nil, err
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
		return nil, nil, fmt.Errorf("output %q was declared by spec but not received from step", k)
	}

	return protoOutputs, nil, nil
}

func (f *Files) loadStepResultFromOutputFile() (*proto.StepResult, error) {
	data, err := os.ReadFile(f.outputFile)

	if err != nil {
		return nil, fmt.Errorf("reading file %v: %w", f.outputFile, err)
	}

	stepResult := &proto.StepResult{}
	if err := protojson.Unmarshal(data, stepResult); err != nil {
		return nil, fmt.Errorf("reading output_file as a step result: %w", err)
	}

	return stepResult, nil
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
