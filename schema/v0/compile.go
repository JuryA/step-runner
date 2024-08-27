package schema

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func (spec *Spec) Compile() (*proto.Spec, error) {
	protoSpec := &proto.Spec{Spec: &proto.Spec_Content{}}
	inputs := map[string]*proto.Spec_Content_Input{}
	for k, v := range spec.Spec.Inputs {
		protoV, err := v.compile()
		if err != nil {
			return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
		}
		inputs[k] = protoV
	}
	protoSpec.Spec.Inputs = inputs
	outputs := map[string]*proto.Spec_Content_Output{}
	switch o := spec.Spec.Outputs.(type) {
	case string:
		protoSpec.Spec.OutputMethod = proto.OutputMethod_delegate
	case Outputs:
		protoSpec.Spec.OutputMethod = proto.OutputMethod_outputs
		for k, v := range o {
			protoV, err := v.compile()
			if err != nil {
				return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
			}
			outputs[k] = protoV
		}
	default:
		return nil, fmt.Errorf("unsupported type: %T", spec.Spec.Outputs)
	}
	protoSpec.Spec.Outputs = outputs
	return protoSpec, nil
}

func (i Input) compile() (*proto.Spec_Content_Input, error) {
	i.defaultTypeToString()
	protoInput, err := i.compileToProto()
	if err != nil {
		return nil, err
	}
	err = i.verifyDefaultValueMatchesType(protoInput)
	if err != nil {
		return nil, err
	}
	return protoInput, nil
}

func (i Input) defaultTypeToString() {
	if i.Type == nil || *i.Type == "" {
		i.Type = &InputTypeString
	}
}

func (i Input) compileToProto() (*proto.Spec_Content_Input, error) {
	protoInput := &proto.Spec_Content_Input{}
	switch *i.Type {
	case InputTypeBoolean:
		protoInput.Type = proto.ValueType_boolean
	case InputTypeArray:
		protoInput.Type = proto.ValueType_array
	case InputTypeNumber:
		protoInput.Type = proto.ValueType_number
	case InputTypeString:
		protoInput.Type = proto.ValueType_string
	case InputTypeStruct:
		protoInput.Type = proto.ValueType_struct
	default:
		return nil, fmt.Errorf("unsupported input type: %v", i.Type)
	}
	if i.Default != nil {
		protoV, err := (&valueCompiler{i.Default}).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling default %v: %w", i.Default, err)
		}
		protoInput.Default = protoV
	}
	if i.Sensitive != nil && *i.Sensitive == true {
		protoInput.Sensitive = true
	}
	return protoInput, nil
}

func (i Input) verifyDefaultValueMatchesType(protoInput *proto.Spec_Content_Input) error {
	if i.Default == nil || protoInput.Default == nil {
		return nil
	}
	if i.Type == nil {
		return nil
	}
	var defaultType InputType
	switch *i.Type {
	case InputTypeBoolean:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_BoolValue); ok {
			defaultType = InputTypeBoolean
		}
	case InputTypeArray:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_ListValue); ok {
			defaultType = InputTypeArray
		}
	case InputTypeNumber:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_NumberValue); ok {
			defaultType = InputTypeNumber
		}
	case InputTypeString:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_StringValue); ok {
			defaultType = InputTypeString
		}
	case InputTypeStruct:
		if _, ok := protoInput.Default.Kind.(*structpb.Value_StructValue); ok {
			defaultType = InputTypeStruct
		}
	default:
		return fmt.Errorf("unsupported type: %v", i.Type)
	}
	if defaultType != *i.Type {
		return fmt.Errorf("input type %v and default value type %v must match", i.Type, defaultType)
	}
	return nil
}

func (o Output) compile() (*proto.Spec_Content_Output, error) {
	o.defaultTypeToRawString()
	protoOutput, err := o.compileToProto()
	if err != nil {
		return nil, err
	}
	err = o.verifyDefaultValueMatchesType(protoOutput)
	if err != nil {
		return nil, err
	}
	return protoOutput, nil
}

func (o Output) defaultTypeToRawString() {
	if o.Type == nil || *o.Type == "" {
		o.Type = &ValueTypeRawString
	}
}

func (o Output) compileToProto() (*proto.Spec_Content_Output, error) {
	protoOutput := &proto.Spec_Content_Output{}
	switch *o.Type {
	case OutputTypeBoolean:
		protoOutput.Type = proto.ValueType_boolean
	case OutputTypeArray:
		protoOutput.Type = proto.ValueType_array
	case OutputTypeNumber:
		protoOutput.Type = proto.ValueType_number
	case OutputTypeRawString:
		protoOutput.Type = proto.ValueType_raw_string
	case OutputTypeString:
		protoOutput.Type = proto.ValueType_string
	case OutputTypeStruct:
		protoOutput.Type = proto.ValueType_struct
	default:
		return nil, fmt.Errorf("unsupported output type: %v", o.Type)
	}
	if o.Default != nil {
		protoV, err := (&valueCompiler{o.Default}).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling default %v: %w", o.Default, err)
		}
		protoOutput.Default = protoV
	}
	if o.Sensitive != nil && *o.Sensitive == true {
		protoOutput.Sensitive = true
	}
	return protoOutput, nil
}

func (o Output) verifyDefaultValueMatchesType(protoOutput *proto.Spec_Content_Output) error {
	if o.Default == nil || protoOutput.Default == nil {
		return nil
	}
	if o.Type == nil {
		return nil
	}
	var defaultType OutputType
	switch *o.Type {
	case OutputTypeBoolean:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_BoolValue); ok {
			defaultType = OutputTypeBoolean
		}
	case OutputTypeArray:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_ListValue); ok {
			defaultType = OutputTypeArray
		}
	case OutputTypeNumber:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_NumberValue); ok {
			defaultType = OutputTypeNumber
		}
	case OutputTypeString:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_StringValue); ok {
			defaultType = OutputTypeString
		}
	case OutputTypeRawString:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_StringValue); ok {
			defaultType = OutputTypeRawString
		}
	case OutputTypeStruct:
		if _, ok := protoOutput.Default.Kind.(*structpb.Value_StructValue); ok {
			defaultType = OutputTypeStruct
		}
	default:
		return fmt.Errorf("unsupported type: %v", o.Type)
	}
	if defaultType != *o.Type {
		return fmt.Errorf("output type %v and default value type %v must match", o.Type, defaultType)
	}
	return nil
}

type definitionCompiler Definition

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
			protoStep, err := (*stepCompiler)(s).compile(i)
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
	protoDef.Delegate = def.Delegate
	return protoDef, nil
}

type stepCompiler Step

func (step *stepCompiler) compile(i int) (*proto.Step, error) {
	err := step.compileScriptKeywordToStep()
	if err != nil {
		return nil, err
	}
	err = step.compileActionKeywordToStep()
	if err != nil {
		return nil, err
	}
	step.defaultName(i)
	return step.compileToProto()
}

func (step *stepCompiler) compileScriptKeywordToStep() error {
	if step.Script == "" {
		return nil
	}
	// TODO replace these checks with JSON schema validation
	if !step.Step.IsEmpty() {
		return fmt.Errorf("the `script` keyword cannot be used with the `step` keyword")
	}
	if step.Action != "" {
		return fmt.Errorf("the `script` keyword cannot be used with the `action` keyword")
	}
	if len(step.Inputs) != 0 {
		return fmt.Errorf("the `script` keyword cannot be used with `inputs`")
	}
	step.Step = Reference{
		Short: scriptStep,
	}
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
	if !step.Step.IsEmpty() {
		return fmt.Errorf("the `action` keyword cannot be used with the `step` keyword")
	}
	if step.Script != "" {
		return fmt.Errorf("the `action` keyword cannot be used with the `script` keyword")
	}
	step.Step = Reference{
		Short: actionStep,
	}
	step.Inputs = map[string]any{
		"action": step.Action,
		"inputs": step.Inputs,
	}
	step.Action = ""
	return nil
}

func (step *stepCompiler) defaultName(i int) {
	if step.Name == "" {
		step.Name = strconv.Itoa(i)
	}
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

type referenceCompiler Reference

func (reference *referenceCompiler) compile() (*proto.Step_Reference, error) {
	if strings.HasPrefix(reference.Short, ".") {
		return reference.compileLocal()
	}
	err := reference.expandShortReference()
	if err != nil {
		return nil, fmt.Errorf("expanding short reference %q: %w", reference.Short, err)
	}
	return reference.compileToProto()
}

func (reference *referenceCompiler) compileLocal() (*proto.Step_Reference, error) {
	path, filename := pathFilename(reference.Short)
	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_local,
		Path:     path,
		Filename: filename,
	}, nil
}

func (reference *referenceCompiler) expandShortReference() error {
	if reference.Short == "" {
		return nil
	}
	url, rev, ok := strings.Cut(reference.Short, "@")
	if !ok {
		return fmt.Errorf("expecting url@rev. got %q", reference.Short)
	}
	reference.Git.Url = url
	reference.Git.Rev = rev
	reference.Git.Dir = ""
	reference.Short = ""
	return nil
}

func (reference *referenceCompiler) compileToProto() (*proto.Step_Reference, error) {
	switch {
	case !reference.Git.IsEmpty():
		url, err := defaultHTTPS(reference.Git.Url)
		if err != nil {
			return nil, fmt.Errorf("parsing url as url: %w", err)
		}
		path, filename := pathFilename(reference.Git.Dir)
		return &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_git,
			Url:      url,
			Path:     path,
			Filename: filename,
			Version:  reference.Git.Rev,
		}, nil
	default:
		return nil, fmt.Errorf("unhandled step reference type: %+v", reference)
	}
}

func defaultHTTPS(stepUrl string) (string, error) {
	parsedURL, err := url.Parse(stepUrl)
	if err != nil {
		return "", fmt.Errorf("invalid step reference url %q: %w", stepUrl, err)
	}
	switch parsedURL.Scheme {
	case "http", "https":
		// Valid
	case "":
		parsedURL.Scheme = "https"
	default:
		return "", fmt.Errorf("unsupported scheme %q in reference %q", parsedURL.Scheme, stepUrl)
	}
	return parsedURL.String(), nil
}

func pathFilename(pathStr string) (path []string, filename string) {
	filename = "step.yml"
	if pathStr == "" {
		return nil, filename
	}
	path = strings.Split(pathStr, "/")
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
