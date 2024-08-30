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
		t := InputTypeString
		i.Type = &t
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
		t := OutputTypeRawString
		o.Type = &t
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

func (s *Step) compileDefinition() (*proto.Definition, error) {
	err := s.verifyOneTypeProvided()
	if err != nil {
		return nil, err
	}
	return s.compileToDefinitionProto()
}

func (s *Step) verifyOneTypeProvided() error {
	have := 0
	if len(s.Exec.Command) > 0 || (s.Exec.WorkDir != nil && *s.Exec.WorkDir != "") {
		// Exec type step
		have++
	}
	if s.Steps != nil {
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

func (s *Step) compileToDefinitionProto() (*proto.Definition, error) {
	protoDef := &proto.Definition{}
	switch {
	case len(s.Exec.Command) > 0:
		// Exec type step
		protoDef.Type = proto.DefinitionType_exec
		protoDef.Exec = &proto.Definition_Exec{
			Command: s.Exec.Command,
		}
		if s.Exec.WorkDir != nil {
			protoDef.Exec.WorkDir = *s.Exec.WorkDir
		}
	case s.Steps != nil:
		// Steps type step
		protoDef.Type = proto.DefinitionType_steps
		protoDef.Steps = make([]*proto.Step, len(s.Steps))
		for i, ss := range s.Steps {
			protoStep, err := (&ss).CompileStep(i)
			if err != nil {
				return nil, fmt.Errorf("compiling steps[%v]: %q: %w", i, s.Name, err)
			}
			protoDef.Steps[i] = protoStep
		}
		protoDef.Outputs = map[string]*structpb.Value{}
		for k, v := range s.Outputs {
			protoV, err := (&valueCompiler{v}).compile()
			if err != nil {
				return nil, fmt.Errorf("compiling output[%q]: %v: %w", k, v, err)
			}
			protoDef.Outputs[k] = protoV
		}
	default:
		return nil, fmt.Errorf("could not determine step type")
	}
	protoDef.Env = s.Env
	if s.Delegate != nil {
		protoDef.Delegate = *s.Delegate
	}
	return protoDef, nil
}

func (s *Step) CompileStep(i int) (*proto.Step, error) {
	err := s.compileScriptKeywordToStep()
	if err != nil {
		return nil, err
	}
	err = s.compileActionKeywordToStep()
	if err != nil {
		return nil, err
	}
	s.defaultName(i)
	return s.compileToStepProto()
}

func (s *Step) compileScriptKeywordToStep() error {
	if s.Script == nil || *s.Script == "" {
		return nil
	}
	// TODO replace these checks with JSON schema validation
	if s.Step != nil {
		return fmt.Errorf("the `script` keyword cannot be used with the `step` keyword")
	}
	if s.Action != nil && *s.Action != "" {
		return fmt.Errorf("the `script` keyword cannot be used with the `action` keyword")
	}
	if len(s.Inputs) != 0 {
		return fmt.Errorf("the `script` keyword cannot be used with `inputs`")
	}
	s.Step = scriptStep
	s.Inputs = map[string]any{
		"script": s.Script,
	}
	s.Script = nil
	return nil
}

func (s *Step) compileActionKeywordToStep() error {
	if s.Action == nil || *s.Action == "" {
		return nil
	}
	// TODO replace these checks with JSON schema validation
	if s.Step != nil {
		return fmt.Errorf("the `action` keyword cannot be used with the `step` keyword")
	}
	if s.Script != nil && *s.Script != "" {
		return fmt.Errorf("the `action` keyword cannot be used with the `script` keyword")
	}
	s.Step = actionStep
	s.Inputs = map[string]any{
		"action": s.Action,
		"inputs": s.Inputs,
	}
	s.Action = nil
	return nil
}

func (s *Step) defaultName(i int) {
	if s.Name == nil || *s.Name == "" {
		n := strconv.Itoa(i)
		s.Name = &n
	}
}

func (s *Step) compileToStepProto() (*proto.Step, error) {
	protoStep := &proto.Step{}
	protoInputs := map[string]*structpb.Value{}
	for k, v := range s.Inputs {
		protoValue, err := (&valueCompiler{v}).compile()
		if err != nil {
			return nil, err
		}
		protoInputs[k] = protoValue
	}
	var (
		ref *proto.Step_Reference
		err error
	)
	switch v := s.Step.(type) {
	case string:
		ref, err = shortReference(v).compile()
	case *GitReference:
		ref, err = v.compile()
	default:
		err = fmt.Errorf("unsupported type: %T", v)
	}
	if err != nil {
		return nil, fmt.Errorf("compiling reference: %w", err)
	}
	if s.Name != nil {
		protoStep.Name = *s.Name
	}
	protoStep.Env = s.Env
	protoStep.Step = ref
	protoStep.Inputs = protoInputs
	return protoStep, nil
}

type shortReference string

func (sr shortReference) compile() (*proto.Step_Reference, error) {
	if strings.HasPrefix(string(sr), ".") {
		return sr.compileLocal()
	}
	return sr.compileRemote()
}

func (sr shortReference) compileLocal() (*proto.Step_Reference, error) {
	path, filename := pathFilename((*string)(&sr))
	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_local,
		Path:     path,
		Filename: filename,
	}, nil
}

func (sr shortReference) compileRemote() (*proto.Step_Reference, error) {
	url, rev, ok := strings.Cut(string(sr), "@")
	if !ok {
		return nil, fmt.Errorf("expecting url@rev. got %q", sr)
	}
	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_git,
		Url:      url,
		Version:  rev,
	}, nil
}

func (gr *GitReference) compile() (*proto.Step_Reference, error) {
	url, err := defaultHTTPS(gr.Url)
	if err != nil {
		return nil, fmt.Errorf("parsing url as url: %w", err)
	}
	path, filename := pathFilename(gr.Dir)
	version := ""
	if gr.Rev != nil {
		version = *gr.Rev
	}
	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_git,
		Url:      url,
		Path:     path,
		Filename: filename,
		Version:  version,
	}, nil
}

func defaultHTTPS(stepUrl *string) (string, error) {
	if stepUrl == nil {
		return "", nil
	}
	parsedURL, err := url.Parse(*stepUrl)
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

func pathFilename(pathStr *string) (path []string, filename string) {
	if pathStr == nil {
		return nil, ""
	}
	filename = "step.yml"
	if *pathStr == "" {
		return nil, filename
	}
	path = strings.Split(*pathStr, "/")
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
