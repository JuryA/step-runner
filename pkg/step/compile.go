package step

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema"
	"google.golang.org/protobuf/types/known/structpb"
)

func CompileSteps(steps *schema.StepDefinition) (*proto.StepDefinition, error) {
	protoStepDef := &proto.StepDefinition{
		Dir: steps.Dir,
	}
	if steps.Spec != nil {
		protoSpec, err := specCompiler(*steps.Spec).compile()
		if err != nil {
			return nil, fmt.Errorf("compiling spec: %w", err)
		}
		protoStepDef.Spec = protoSpec
	}
	if steps.Definition != nil {
		protoDef, err := definitionCompiler(*steps.Definition).compile()
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

type compileRule struct {
	name string
	rule func() error
}

type specCompiler schema.Spec

func (spec specCompiler) compile() (*proto.Spec, error) {
	protoSpec := &proto.Spec{Spec: &proto.Spec_Content{}}
	for _, r := range []compileRule{{
		name: "compiling schema spec into proto spec",
		rule: func() error {
			inputs := map[string]*proto.Spec_Content_Input{}
			for k, v := range spec.Spec.Inputs {
				protoV, err := inputCompiler(v).compile()
				if err != nil {
					return fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
				}
				inputs[k] = protoV
			}
			outputs := map[string]*proto.Spec_Content_Output{}
			for k, v := range spec.Spec.Outputs {
				protoV, err := outputCompiler(v).compile()
				if err != nil {
					return fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
				}
				outputs[k] = protoV
			}
			protoSpec.Spec.Inputs = inputs
			protoSpec.Spec.Outputs = outputs
			return nil
		},
	}} {
		err := r.rule()
		if err != nil {
			return nil, fmt.Errorf("%v: %w", r.name, err)
		}
	}
	return protoSpec, nil
}

type inputCompiler schema.Input

func (input inputCompiler) compile() (*proto.Spec_Content_Input, error) {
	protoInput := &proto.Spec_Content_Input{}
	for _, r := range []compileRule{{
		name: "defaulting type to string",
		rule: func() error {
			if input.Type != "" {
				return nil
			}
			input.Type = schema.ValueTypeString
			return nil
		},
	}, {
		name: "compiling schema input to proto input",
		rule: func() error {
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
				return fmt.Errorf("unsupported input type: %v", input.Type)
			}
			if input.Default != nil {
				protoV, err := valueCompiler{input.Default}.compile()
				if err != nil {
					return fmt.Errorf("compiling default %v: %w", input.Default, err)
				}
				protoInput.Default = protoV
			}
			return nil
		},
	}, {
		name: "making sure default value matches the type",
		rule: func() error {
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
		},
	}} {
		err := r.rule()
		if err != nil {
			return nil, fmt.Errorf("%v: %w", r.name, err)
		}
	}
	return protoInput, nil
}

type outputCompiler schema.Output

func (output outputCompiler) compile() (*proto.Spec_Content_Output, error) {
	protoOutput := &proto.Spec_Content_Output{}
	protoOutput.Default = output.Default
	return protoOutput, nil
}

type definitionCompiler schema.Definition

func (def definitionCompiler) compile() (*proto.Definition, error) {
	protoDef := &proto.Definition{}
	for _, r := range []compileRule{{
		name: "compiling top-level script into single script step",
		rule: func() error {
			if def.Script == "" {
				return nil
			}
			if def.Type != "" {
				return fmt.Errorf("definition `script` keyword doesn't need a `type`")
			}
			if len(def.Script) > 0 && def.Steps != nil {
				return fmt.Errorf("definition `script` keyword cannot be used with the `steps` keyword")
			}
			def.Type = schema.DefinitionTypeSteps
			def.Steps = []*schema.Step{{
				Name:   "run a script",
				Script: def.Script,
			}}
			def.Script = ""
			return nil
		},
	}, {
		name: "compiling schema definition into proto definition",
		rule: func() error {
			switch def.Type {
			case schema.DefinitionTypeExec:
				protoDef.Type = proto.DefinitionType_exec
				protoDef.Exec = &proto.Definition_Exec{
					Command: def.Exec.Command,
					WorkDir: def.Exec.WorkDir,
				}
			case schema.DefinitionTypeSteps:
				protoDef.Type = proto.DefinitionType_steps
				protoDef.Steps = make([]*proto.Step, len(def.Steps))
				for i, s := range def.Steps {
					protoStep, err := stepCompiler(*s).compile()
					if err != nil {
						return fmt.Errorf("compiling steps[%v]: %q: %w", i, s.Name, err)
					}
					protoDef.Steps[i] = protoStep
				}
				protoDef.Outputs = def.Outputs
			default:
				return fmt.Errorf("type not specified. must be `exec` or `steps`")
			}
			return nil
		},
	}} {
		err := r.rule()
		if err != nil {
			return nil, fmt.Errorf("%v: %w", r.name, err)
		}
	}
	return protoDef, nil
}

type stepCompiler schema.Step

func (step stepCompiler) compile() (*proto.Step, error) {
	protoStep := &proto.Step{}
	for _, r := range []compileRule{{
		name: "compiling step `script` keyword to a script step",
		rule: func() error {
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
		},
	}, {
		name: "compiling schema step into proto step",
		rule: func() error {
			protoInputs := map[string]*structpb.Value{}
			for k, v := range step.Inputs {
				protoValue, err := valueCompiler{v}.compile()
				if err != nil {
					return err
				}
				protoInputs[k] = protoValue
			}
			protoStep.Name = step.Name
			protoStep.Env = step.Env
			protoStep.Step = step.Step
			protoStep.Inputs = protoInputs
			return nil
		},
	}} {
		err := r.rule()
		if err != nil {
			return nil, fmt.Errorf("%v: %w", r.name, err)
		}
	}
	return protoStep, nil
}

type valueCompiler struct {
	v any
}

func (value valueCompiler) compile() (*structpb.Value, error) {
	var protoValue *structpb.Value
	for _, r := range []compileRule{{
		name: "compiling generic data into proto value",
		rule: func() error {
			var err error
			// We let structpb do all the heavy lifting
			// and verify the type matches our
			// expectations later.
			protoValue, err = structpb.NewValue(value.v)
			return err
		},
	}} {
		err := r.rule()
		if err != nil {
			return nil, fmt.Errorf("%v: %w", r.name, err)
		}
	}

	return protoValue, nil
}
