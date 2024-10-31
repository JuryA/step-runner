package runner

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Files struct {
	stepCtx      *StepsContext
	outputMethod proto.OutputMethod
	specOutputs  map[string]*proto.Spec_Content_Output
	outputFile   *StepFile
}

func NewFiles(
	stepCtx *StepsContext,
	outputMethod proto.OutputMethod,
	specOutputs map[string]*proto.Spec_Content_Output,
) (*Files, error) {
	outputFile, err := NewStepFileInTmp()

	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}

	stepCtx.OutputFile = outputFile.Path()

	return &Files{
		stepCtx:      stepCtx,
		outputMethod: outputMethod,
		specOutputs:  specOutputs,
		outputFile:   outputFile,
	}, nil
}

func (f *Files) Outputs() (map[string]*structpb.Value, *proto.StepResult, error) {
	// Delegates take over step execution including reading and validating outputs.
	if f.outputMethod == proto.OutputMethod_delegate {
		delegateResult, err := f.outputFile.ReadStepResult()

		if err != nil {
			return nil, nil, fmt.Errorf("reading outputs: %w", err)
		}

		return delegateResult.Outputs, delegateResult, nil
	}

	outputs, err := f.outputFile.ReadKeyValueLines()
	if err != nil {
		return nil, nil, fmt.Errorf("reading outputs: %w", err)
	}

	protoOutputs := map[string]*structpb.Value{}
	for k, v := range outputs {
		outputSpec, ok := f.specOutputs[k]
		if !ok {
			return nil, nil, fmt.Errorf("output %q received from step is not declared in spec", k)
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
	_ = f.outputFile.Remove()
}
