package schema

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/proto"
)

var (
	_ yaml.Unmarshaler = &Step{}
	_ json.Unmarshaler = &Step{}
)

// Inputs is a map of step input names to structured values.
type StepInputs map[string]interface{}

// Outputs are the output values for a sequence. They can reference the outputs of
// sub-steps.
type StepOutputs map[string]interface{}

// Step is a unit of execution.
type Step struct {
	// Action is a GitHub action to run.
	Action *string `json:"action,omitempty" yaml:"action,omitempty" mapstructure:"action,omitempty"`

	// Delegate selects a step by name which will produce the outputs a run.
	Delegate *string `json:"delegate,omitempty" yaml:"delegate,omitempty" mapstructure:"delegate,omitempty"`

	// Env is a map of environment variable names to string values.
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty" mapstructure:"env,omitempty"`

	// Exec is a command to run.
	Exec *Exec `json:"exec,omitempty" yaml:"exec,omitempty" mapstructure:"exec,omitempty"`

	// Inputs is a map of step input names to structured values.
	Inputs StepInputs `json:"inputs,omitempty" yaml:"inputs,omitempty" mapstructure:"inputs,omitempty"`

	// Name is a unique identifier for this step.
	Name *string `json:"name,omitempty" yaml:"name,omitempty" mapstructure:"name,omitempty"`

	// Outputs are the output values for a sequence. They can reference the outputs of
	// sub-steps.
	Outputs StepOutputs `json:"outputs,omitempty" yaml:"outputs,omitempty" mapstructure:"outputs,omitempty"`

	// Script is a shell script to evaluate.
	Script *string `json:"script,omitempty" yaml:"script,omitempty" mapstructure:"script,omitempty"`

	// Step is a reference to another step to invoke.
	Step any `json:"step,omitempty" yaml:"step,omitempty" mapstructure:"step,omitempty"`

	// Run is a list of sub-steps to run.
	Run []Step `json:"run,omitempty" yaml:"run,omitempty" mapstructure:"run,omitempty"`
}

func (s *Step) UnmarshalYAML(value *yaml.Node) error {
	type Default Step
	d := (*Default)(s)
	err := value.Decode(d)
	if err != nil {
		return err
	}
	return s.unmarshalStep()
}

func (s *Step) UnmarshalJSON(data []byte) error {
	type Default Step
	d := (*Default)(s)
	err := json.Unmarshal(data, d)
	if err != nil {
		return err
	}
	return s.unmarshalStep()
}

func (s *Step) unmarshalStep() error {
	if s.Step == nil {
		return nil
	}
	switch v := s.Step.(type) {
	case string:
		return nil
	case map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("reifying step: %w", err)
		}
		ref := &Reference{}
		err = json.Unmarshal(data, ref)
		if err != nil {
			return fmt.Errorf("reifying step: %w", err)
		}
		s.Step = ref
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
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
		protoDef.Type = proto.DefinitionType_exec
		protoDef.Exec = &proto.Definition_Exec{
			Command: s.Exec.Command,
		}
		if s.Exec.WorkDir != nil {
			protoDef.Exec.WorkDir = *s.Exec.WorkDir
		}
	case s.Run != nil:
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

	s.Step = &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_dist,
		Path:     []string{"script"},
		Filename: "step.yml",
	}

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
	case *proto.Step_Reference:
		ref = v
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
