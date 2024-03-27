package step

import (
	"fmt"
	"net/url"
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
		protoInput.Type = proto.ValueType_bool
	case schema.ValueTypeList:
		protoInput.Type = proto.ValueType_list
	case schema.ValueTypeNumber:
		protoInput.Type = proto.ValueType_number
	case schema.ValueTypeString:
		protoInput.Type = proto.ValueType_string
	case schema.ValueTypeStruct:
		protoInput.Type = proto.ValueType_struct
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

func (input *inputCompiler) verifyDefaultValueMatchesType(protoInput *proto.Spec_Content_Input) error {
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
	output.defaultTypeToRawString()
	protoOutput, err := output.compileToProto()
	if err != nil {
		return nil, err
	}
	err = output.verifyDefaultValueMatchesType(protoOutput)
	if err != nil {
		return nil, err
	}
	return protoOutput, nil
}

func (output *outputCompiler) defaultTypeToRawString() {
	if output.Type != "" {
		return
	}
	output.Type = schema.ValueTypeRawString
}

func (output *outputCompiler) compileToProto() (*proto.Spec_Content_Output, error) {
	protoOutput := &proto.Spec_Content_Output{}
	switch output.Type {
	case schema.ValueTypeBool:
		protoOutput.Type = proto.ValueType_bool
	case schema.ValueTypeList:
		protoOutput.Type = proto.ValueType_list
	case schema.ValueTypeNumber:
		protoOutput.Type = proto.ValueType_number
	case schema.ValueTypeRawString:
		protoOutput.Type = proto.ValueType_raw_string
	case schema.ValueTypeString:
		protoOutput.Type = proto.ValueType_string
	case schema.ValueTypeStruct:
		protoOutput.Type = proto.ValueType_struct
	default:
		return nil, fmt.Errorf("unsupported output type: %v", output.Type)
	}
	if output.Default != nil {
		protoV, err := (&valueCompiler{output.Default}).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling default %v: %w", output.Default, err)
		}
		protoOutput.Default = protoV
	}
	return protoOutput, nil
}

func (output *outputCompiler) verifyDefaultValueMatchesType(protoOutput *proto.Spec_Content_Output) error {
	if output.Default == nil || protoOutput.Default == nil {
		return nil
	}
	var defaultType schema.ValueType
	switch output.Type {
	case schema.ValueTypeBool:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_BoolValue); ok {
			defaultType = schema.ValueTypeBool
		}
	case schema.ValueTypeList:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_ListValue); ok {
			defaultType = schema.ValueTypeList
		}
	case schema.ValueTypeNumber:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_NumberValue); ok {
			defaultType = schema.ValueTypeNumber
		}
	case schema.ValueTypeString:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_StringValue); ok {
			defaultType = schema.ValueTypeString
		}
	case schema.ValueTypeRawString:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_StringValue); ok {
			defaultType = schema.ValueTypeRawString
		}
	case schema.ValueTypeStruct:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_StructValue); ok {
			defaultType = schema.ValueTypeStruct
		}
	default:
		return fmt.Errorf("unsupported type: %v", output.Type)
	}
	if defaultType != output.Type {
		return fmt.Errorf("output type %v and default value type %v must match", output.Type, defaultType)
	}
	return nil
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
		protoDef.Outputs = map[string]*structpb.Value{}
		for k, v := range def.Outputs {
			protoV, err := (&valueCompiler{v}).compile()
			if err != nil {
				return nil, fmt.Errorf("compiling output[%q]: %v: %w", k, v, err)
			}
			protoDef.Outputs[k] = protoV
		}
	default:
		return nil, fmt.Errorf("could not determine step type")
	}
	protoDef.Env = def.Env
	return protoDef, nil
}

type stepCompiler schema.Step

func (step *stepCompiler) compile() (*proto.Step, error) {
	err := step.compileScriptKeywordToStep()
	if err != nil {
		return nil, err
	}
	err = step.compileActionKeywordToStep()
	if err != nil {
		return nil, err
	}
	return step.compileToProto()
}

func (step *stepCompiler) compileScriptKeywordToStep() error {
	if step.Script == "" {
		return nil
	}
	// TODO replace these checks with JSON schema validation
	if step.Step != "" {
		return fmt.Errorf("the `script` keyword cannot be used with the `step` keyword")
	}
	if step.Action != "" {
		return fmt.Errorf("the `script` keyword cannot be used with the `action` keyword")
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

func (step *stepCompiler) compileActionKeywordToStep() error {
	if step.Action == "" {
		return nil
	}
	// TODO replace these checks with JSON schema validation
	if step.Step != "" {
		return fmt.Errorf("the `action` keyword cannot be used with the `step` keyword")
	}
	if step.Script != "" {
		return fmt.Errorf("the `action` keyword cannot be used with the `script` keyword")
	}
	step.Step = actionStep
	step.Inputs = map[string]any{
		"action": step.Action,
		"inputs": step.Inputs,
	}
	step.Action = ""
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
	ref := string(*reference)
	// Local file
	if strings.HasPrefix(ref, ".") {
		path, filename := pathFilename(ref)
		return &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_local,
			// Step references always use '/' as a path
			// separator, regardless of operating system.
			Path:     path,
			Filename: filename,
		}, nil
	}
	// A non-local step reference is a valid URL
	parsedURL, err := url.Parse(ref)
	if err != nil {
		return nil, fmt.Errorf("invalid step reference %q: %w", ref, err)
	}
	// Parse fragment
	if parsedURL.Fragment == "" {
		return nil, fmt.Errorf("invalid step reference %q. must have fragment specifying protocol and version", ref)
	}
	var nestedPathStr, protocolVersion string
	before, after, haveNestedPath := strings.Cut(parsedURL.Fragment, ",")
	if haveNestedPath {
		nestedPathStr = before
		protocolVersion = after
	} else {
		protocolVersion = before
	}
	protocol, version, ok := strings.Cut(protocolVersion, "@")
	if !ok {
		return nil, fmt.Errorf("invalid protocol and version %q", protocolVersion)
	}
	nestedPath, filename := pathFilename(nestedPathStr)
	// Reassemble the URL sans fragment
	parsedURL.Fragment = ""
	switch parsedURL.Scheme {
	case "http", "https":
		// Valid
	case "":
		// Default
		parsedURL.Scheme = "https"
	default:
		return nil, fmt.Errorf("unsupported scheme %q in reference %q", parsedURL.Scheme, ref)
	}
	url := parsedURL.String()

	protoRef := &proto.Step_Reference{
		Url:      url,
		Path:     nestedPath,
		Filename: filename,
		Version:  version,
	}
	switch protocol {
	case "git":
		protoRef.Protocol = proto.StepReferenceProtocol_git
	case "oci":
		protoRef.Protocol = proto.StepReferenceProtocol_oci
	default:
		return nil, fmt.Errorf("unsupported protocol %q", protocol)
	}
	return protoRef, nil
}

func pathFilename(pathStr string) (path []string, filename string) {
	filename = "step.yml"
	if pathStr == "" {
		return nil, filename
	}
	path = strings.Split(pathStr, "/")
	if len(path) > 1 {
		lastElement := path[len(path)-1]
		if strings.HasSuffix(lastElement, ".yml") || strings.HasSuffix(lastElement, ".yaml") {
			path = path[:len(path)-1]
			filename = lastElement
		}
	}
	return path, filename
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
