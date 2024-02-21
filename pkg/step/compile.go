package step

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/step-runner/proto"
	schema "gitlab.com/gitlab-org/step-runner/schema/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func CompileSteps(steps *schema.StepDefinition) (*proto.StepDefinition, error) {
	protoStepDef := &proto.StepDefinition{
		Dir: steps.Dir,
	}
	if steps.Spec != nil {
		protoSpec, err := (*specCompiler)(steps.Spec).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling spec: %w", err)
		}
		protoStepDef.Spec = protoSpec
	}
	if steps.Definition != nil {
		protoDef, err := (*definitionCompiler)(steps.Definition).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling definition: %w", err)
		}
		protoStepDef.Definition = protoDef
	}

	if err := ValidateStepDefinition(protoStepDef); err != nil {
		return nil, err
	}
	return protoStepDef, nil
}

type specCompiler schema.Spec

func (spec *specCompiler) compile() (*proto.Spec, error) {
	protoSpec := &proto.Spec{Spec: &proto.Spec_Content{}}
	inputs := map[string]*proto.Spec_Content_Input{}
	for k, v := range spec.Spec.Inputs {
		protoV, err := (*inputCompiler)(&v).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
		}
		inputs[k] = protoV
	}
	outputs := map[string]*proto.Spec_Content_Output{}
	for k, v := range spec.Spec.Outputs {
		protoV, err := (*outputCompiler)(&v).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
		}
		outputs[k] = protoV
	}
	protoSpec.Spec.Inputs = inputs
	protoSpec.Spec.Outputs = outputs
	return protoSpec, nil
}

type inputCompiler schema.Input

func (input *inputCompiler) compile() (*proto.Spec_Content_Input, error) {
	input.defaultTypeToString()
	protoInput, err := input.compileToProto()
	if err != nil {
		return nil, err
	}
	err = input.verifyDefaultValueMatchesType(protoInput)
	if err != nil {
		return nil, err
	}
	return protoInput, nil
}

func (input *inputCompiler) defaultTypeToString() {
	if input.Type != "" {
		return
	}
	input.Type = schema.ValueTypeString
}

func (input *inputCompiler) compileToProto() (*proto.Spec_Content_Input, error) {
	protoInput := &proto.Spec_Content_Input{}
	switch input.Type {
	case schema.ValueTypeBool:
		protoInput.Type = proto.InputType_bool
	case schema.ValueTypeList:
		protoInput.Type = proto.InputType_list
	case schema.ValueTypeNumber:
		protoInput.Type = proto.InputType_number
	case schema.ValueTypeString:
		protoInput.Type = proto.InputType_string
	case schema.ValueTypeStruct:
		protoInput.Type = proto.InputType_struct
	default:
		return nil, fmt.Errorf("unsupported input type: %v", input.Type)
	}
	if input.Default != nil {
		protoV, err := (&valueCompiler{input.Default}).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling default %v: %w", input.Default, err)
		}
		protoInput.Default = protoV
	}
	return protoInput, nil
}

func (input inputCompiler) verifyDefaultValueMatchesType(protoInput *proto.Spec_Content_Input) error {
	if input.Default == nil || protoInput.Default == nil {
		return nil
	}
	var defaultType schema.ValueType
	switch input.Type {
	case schema.ValueTypeBool:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_BoolValue); ok {
			defaultType = schema.ValueTypeBool
		}
	case schema.ValueTypeList:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_ListValue); ok {
			defaultType = schema.ValueTypeList
		}
	case schema.ValueTypeNull:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_NullValue); ok {
			defaultType = schema.ValueTypeNull
		}
	case schema.ValueTypeNumber:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_NumberValue); ok {
			defaultType = schema.ValueTypeNumber
		}
	case schema.ValueTypeString:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_StringValue); ok {
			defaultType = schema.ValueTypeString
		}
	case schema.ValueTypeStruct:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_StructValue); ok {
			defaultType = schema.ValueTypeStruct
		}
	default:
		return fmt.Errorf("unsupported type: %v", input.Type)
	}
	if defaultType != input.Type {
		return fmt.Errorf("input type %v and default value type %v must match", input.Type, defaultType)
	}
	return nil
}

type outputCompiler schema.Output

func (output *outputCompiler) compile() (*proto.Spec_Content_Output, error) {
	protoOutput := &proto.Spec_Content_Output{}
	protoOutput.Default = output.Default
	return protoOutput, nil
}

type definitionCompiler schema.Definition

func (def *definitionCompiler) compile() (*proto.Definition, error) {
	err := def.verifyOneTypeProvided()
	if err != nil {
		return nil, err
	}
	return def.compileToProto()
}

func (def *definitionCompiler) verifyOneTypeProvided() error {
	have := 0
	if len(def.Exec.Command) > 0 || def.Exec.WorkDir != "" {
		// Exec type step
		have++
	}
	if def.Steps != nil {
		// Steps type step
		have++
	}
	if have == 0 {
		return fmt.Errorf("at least one of `script`, `exec` or `steps` must be provided")
	}
	if have > 1 {
		return fmt.Errorf("only one of `script`, `exec` or `steps` may be provided. have %v", have)
	}
	return nil
}

func (def *definitionCompiler) compileToProto() (*proto.Definition, error) {
	protoDef := &proto.Definition{}
	switch {
	case len(def.Exec.Command) > 0:
		// Exec type step
		protoDef.Type = proto.DefinitionType_exec
		protoDef.Exec = &proto.Definition_Exec{
			Command: def.Exec.Command,
			WorkDir: def.Exec.WorkDir,
		}
	case def.Steps != nil:
		// Steps type step
		protoDef.Type = proto.DefinitionType_steps
		protoDef.Steps = make([]*proto.Step, len(def.Steps))
		for i, s := range def.Steps {
			protoStep, err := (*stepCompiler)(s).compile()
			if err != nil {
				return nil, fmt.Errorf("compiling steps[%v]: %q: %w", i, s.Name, err)
			}
			protoDef.Steps[i] = protoStep
		}
		protoDef.Outputs = def.Outputs
	default:
		return nil, fmt.Errorf("could not determine step type")
	}
	return protoDef, nil
}

type stepCompiler schema.Step

func (step *stepCompiler) compile() (*proto.Step, error) {
	err := step.compileScriptKeywordToStep()
	if err != nil {
		return nil, err
	}
	return step.compileToProto()
}

func (step *stepCompiler) compileScriptKeywordToStep() error {
	if step.Script == "" {
		return nil
	}
	if step.Step != "" {
		return fmt.Errorf("the `script` keyword cannot be used with the `step` keyword")
	}
	if len(step.Inputs) != 0 {
		return fmt.Errorf("the `script` keyword cannot be used with `inputs`")
	}
	step.Step = scriptStep
	step.Inputs = map[string]any{
		"script": step.Script,
	}
	step.Script = ""
	return nil
}

func (step *stepCompiler) compileToProto() (*proto.Step, error) {
	protoStep := &proto.Step{}
	protoInputs := map[string]*structpb.Value{}
	for k, v := range step.Inputs {
		protoValue, err := (&valueCompiler{v}).compile()
		if err != nil {
			return nil, err
		}
		protoInputs[k] = protoValue
	}
	ref, err := (*referenceCompiler)(&step.Step).compile()
	if err != nil {
		return nil, fmt.Errorf("compiling reference: %w", err)
	}
	protoStep.Name = step.Name
	protoStep.Env = step.Env
	protoStep.Step = ref
	protoStep.Inputs = protoInputs
	return protoStep, nil
}

type referenceCompiler string

func (reference *referenceCompiler) compile() (*proto.Step_Reference, error) {
	refStr := string(*reference)
	url := strings.Replace(refStr, "https+git://", "https://", 1)
	switch {
	case strings.HasPrefix(refStr, "."):
		return &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_local,
			// Step references always use '/' as a path
			// separator, regardless of operating system.
			Path:     strings.Split(refStr, "/"),
			Filename: "step.yml",
		}, nil
	case strings.HasPrefix(url, "https://"):
		url, versionPlusPath, _ := strings.Cut(url, "@")
		version, path, _ := strings.Cut(versionPlusPath, ":")
		if path != "" {
			return nil, fmt.Errorf("nested steps are not yet supported")
		}
		return &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      url,
			Version:  version,
			Filename: "step.yml",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported step reference: %q", refStr)
	}
}

type valueCompiler struct {
	v any
}

func (value *valueCompiler) compile() (*structpb.Value, error) {
	// We let structpb do all the heavy lifting
	// and verify the type matches our
	// expectations later.
	return structpb.NewValue(value.v)
}
