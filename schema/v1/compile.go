package schema

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

const builtInPrefix = "builtin://"

func (spec *Spec) Compile() (*proto.Spec, error) {
	protoSpec := &proto.Spec{Spec: &proto.Spec_Content{}}
	inputs := map[string]*proto.Spec_Content_Input{}
	if spec.Spec == nil {
		spec.Spec = &Signature{}
	}
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
	case *Outputs:
		protoSpec.Spec.OutputMethod = proto.OutputMethod_outputs
		for k, v := range *o {
			protoV, err := v.compile()
			if err != nil {
				return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
			}
			outputs[k] = protoV
		}
	case nil:
		protoSpec.Spec.OutputMethod = proto.OutputMethod_outputs
	default:
		return nil, fmt.Errorf("unsupported type: %T", spec.Spec.Outputs)
	}
	protoSpec.Spec.Outputs = outputs
	return protoSpec, nil
}

func (i *Input) compile() (*proto.Spec_Content_Input, error) {
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

func (i *Input) compileToProto() (*proto.Spec_Content_Input, error) {
	protoInput := &proto.Spec_Content_Input{}

	if i.Type == nil {
		return nil, fmt.Errorf("missing input type")
	}

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
	if i.Sensitive != nil && *i.Sensitive {
		protoInput.Sensitive = true
	}
	return protoInput, nil
}

func (i *Input) verifyDefaultValueMatchesType(protoInput *proto.Spec_Content_Input) error {
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

func (o *Output) compile() (*proto.Spec_Content_Output, error) {
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

func (o *Output) compileToProto() (*proto.Spec_Content_Output, error) {
	protoOutput := &proto.Spec_Content_Output{}

	if o.Type == nil {
		return nil, fmt.Errorf("missing output type")
	}

	switch *o.Type {
	case OutputTypeBoolean:
		protoOutput.Type = proto.ValueType_boolean
	case OutputTypeArray:
		protoOutput.Type = proto.ValueType_array
	case OutputTypeNumber:
		protoOutput.Type = proto.ValueType_number
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
	if o.Sensitive != nil && *o.Sensitive {
		protoOutput.Sensitive = true
	}
	return protoOutput, nil
}

func (o *Output) verifyDefaultValueMatchesType(protoOutput *proto.Spec_Content_Output) error {
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

func (s *Step) Compile() (*proto.Definition, error) {
	err := s.verifyOneTypeProvided()
	if err != nil {
		return nil, err
	}
	return s.compileToDefinitionProto()
}

func (s *Step) verifyOneTypeProvided() error {
	have := 0
	if s.Exec != nil {
		// Exec type step
		have++
	}
	if s.Run != nil {
		// Run type step
		have++
	}
	if have == 0 {
		return fmt.Errorf("at least one of `script, `action`, `run` or `exec` must be provided")
	}
	if have > 1 {
		return fmt.Errorf("only one of `script`, `action`, `run` or `exec` may be provided. have %v", have)
	}
	return nil
}

func (s *Step) compileToDefinitionProto() (*proto.Definition, error) {
	protoDef := &proto.Definition{}
	switch {
	case s.Exec != nil:
		// Exec step
		protoDef.Type = proto.DefinitionType_exec
		protoDef.Exec = &proto.Definition_Exec{
			Command: s.Exec.Command,
		}
		if s.Exec.WorkDir != nil {
			protoDef.Exec.WorkDir = *s.Exec.WorkDir
		}
	case s.Run != nil:
		// Run step
		protoDef.Type = proto.DefinitionType_steps
		protoDef.Steps = make([]*proto.Step, len(s.Run))
		for i, ss := range s.Run {
			protoStep, err := (&ss).CompileStep(i)
			if err != nil {
				return nil, fmt.Errorf("compiling run[%v]: %v: %w", i, s.Name, err)
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
	return s.compileToStepProto()
}

func (s *Step) compileScriptKeywordToStep() error {
	if s.Script == nil || *s.Script == "" {
		return nil
	}
	if s.Step != nil {
		return fmt.Errorf("the `script` keyword cannot be used with the `step` keyword")
	}
	if s.Action != nil && *s.Action != "" {
		return fmt.Errorf("the `script` keyword cannot be used with the `action` keyword")
	}
	if len(s.Inputs) != 0 {
		return fmt.Errorf("the `script` keyword cannot be used with `inputs`")
	}
	s.Step = &Reference{Git: NewGitReference("https://gitlab.com/components/script", "v2")}
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
	if s.Step != nil {
		return fmt.Errorf("the `action` keyword cannot be used with the `step` keyword")
	}
	if s.Script != nil && *s.Script != "" {
		return fmt.Errorf("the `action` keyword cannot be used with the `script` keyword")
	}
	s.Step = &Reference{Git: NewGitReference("https://gitlab.com/components/action-runner", "main")}
	s.Inputs = map[string]any{
		"action": s.Action,
		"inputs": s.Inputs,
	}
	s.Action = nil
	return nil
}

func (s *Step) compileToStepProto() (*proto.Step, error) {
	protoStep := &proto.Step{}
	protoInputs := map[string]*structpb.Value{}
	for k, v := range (map[string]any)(s.Inputs) {
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
	case *Reference:
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

	if strings.HasPrefix(string(sr), builtInPrefix) {
		return sr.compileBuiltIn()
	}

	return sr.compileRemote()
}

func (sr shortReference) compileBuiltIn() (*proto.Step_Reference, error) {
	stepDir := strings.TrimPrefix(string(sr), builtInPrefix)

	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_builtin,
		Path:     strings.Split(stepDir, "/"),
		Filename: "step.yml",
	}, nil
}

func (sr shortReference) compileLocal() (*proto.Step_Reference, error) {
	path, filename := pathFilename(string(sr))
	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_local,
		Path:     path,
		Filename: filename,
	}, nil
}

func (sr shortReference) compileRemote() (*proto.Step_Reference, error) {
	parts := strings.Split(string(sr), "@")
	if len(parts) < 2 {
		return nil, fmt.Errorf("expecting url@rev. got %q", sr)
	}
	rest := strings.Join(parts[0:len(parts)-1], "@")
	rev := parts[len(parts)-1]
	url, rest, _ := strings.Cut(rest, "/-/")
	url = defaultHTTPS(url)
	path, filename := pathFilename(rest)
	path = append([]string{"steps"}, path...)
	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_git,
		Url:      url,
		Version:  rev,
		Path:     path,
		Filename: filename,
	}, nil
}

func (r *Reference) compile() (*proto.Step_Reference, error) {
	if r.Git == nil && r.OCI == nil {
		return nil, fmt.Errorf("compiling reference: git or oci not specified")
	}

	if r.Git != nil && r.OCI != nil {
		return nil, fmt.Errorf("compiling reference: git and oci specified")
	}

	if r.Git != nil {
		return r.compileGit()
	}

	return r.compileOCI()
}

func (r *Reference) compileGit() (*proto.Step_Reference, error) {
	url := defaultHTTPS(r.Git.Url)
	s := &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_git,
		Url:      url,
		Filename: "step.yml",
		Version:  r.Git.Rev,
	}
	if r.Git.Dir != nil {
		s.Path = strings.Split(*r.Git.Dir, "/")
	}
	if r.Git.File != nil {
		s.Filename = *r.Git.File
	}
	return s, nil
}

func (r *Reference) compileOCI() (*proto.Step_Reference, error) {
	s := &proto.Step_Reference{
		Protocol:   proto.StepReferenceProtocol_oci,
		Registry:   r.OCI.Registry,
		Repository: r.OCI.Repository,
		Tag:        r.OCI.Tag,
		Filename:   "step.yml",
	}
	if r.OCI.Dir != nil {
		s.Path = strings.Split(*r.OCI.Dir, "/")
	}
	if r.OCI.File != nil {
		s.Filename = *r.OCI.File
	}
	if r.OCI.Tag == "" {
		s.Tag = "latest"
	}
	return s, nil
}

func defaultHTTPS(stepUrl string) string {
	if strings.HasPrefix(stepUrl, "http://") || strings.HasPrefix(stepUrl, "https://") {
		return stepUrl
	}
	return "https://" + stepUrl
}

func pathFilename(pathStr string) (path []string, filename string) {
	filename = "step.yml"
	if pathStr == "" {
		return nil, filename
	}
	path = strings.Split(pathStr, "/")
	if strings.HasSuffix(path[len(path)-1], ".yml") {
		filename = path[len(path)-1]
		path = path[:len(path)-1]
	}
	return path, filename
}

type valueCompiler struct {
	v any
}

func (value *valueCompiler) compile() (*structpb.Value, error) {
	var simplifyTypes func(any) any
	simplifyTypes = func(v any) any {
		// Map a few types from our model to ones that
		// structpb can handle.
		switch v := v.(type) {
		case *string:
			if v != nil {
				return *v
			}
		case StepInputs:
			simpleMap := map[string]any{}
			for k, v := range v {
				simpleMap[k] = simplifyTypes(v)
			}
			return simpleMap
		}
		return v
	}
	// We let structpb do all the heavy lifting
	// and verify the type matches our
	// expectations later.
	return structpb.NewValue(simplifyTypes(value.v))
}
