package step

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema"
	"google.golang.org/protobuf/types/known/structpb"
)

func compileTo[K any](value any) (*K, error) {
	var err error
	for _, r := range rules {
		value, err = r.rule(value)
		if err != nil {
			return nil, fmt.Errorf("failed rule %q: %w", r.name, err)
		}
	}
	result, ok := value.(K)
	if !ok {
		return nil, fmt.Errorf("failed to compile to %T. likely missing a rule", result)
	}
	return &result, nil
}

var rules []compilerRule

func init() {

	// The order of `rules` determines the order of
	// compilation. E.g. we should compile the `script` keyword
	// into a script step before we compile the `container`
	// keyword which will encapsulate the rest of the step in a
	// docker run step.

	rules = []compilerRule{

		// Compiling *scheme.Spec

		{
			name: "compile schema spec into proto spec",
			rule: whenType[*schema.Spec](func(spec *schema.Spec) (any, error) {
				inputs := map[string]*proto.Spec_Content_Input{}
				for k, v := range spec.Spec.Inputs {
					protoV, err := compileTo[*proto.Spec_Content_Input](v)
					if err != nil {
						return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
					}
					inputs[k] = *protoV
				}
				outputs := map[string]*proto.Spec_Content_Output{}
				for k, v := range spec.Spec.Inputs {
					protoV, err := compileTo[*proto.Spec_Content_Output](v)
					if err != nil {
						return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
					}
					outputs[k] = *protoV
				}
				return &proto.Spec{
					Spec: &proto.Spec_Content{
						Inputs:  inputs,
						Outputs: outputs,
					},
				}, nil
			}),
		},

		// Compiling *schema.Definition

		{
			name: "top level `script` keyword is compiled into a single script step",
			rule: whenType[*schema.Definition](func(def *schema.Definition) (any, error) {
				if len(def.Script) > 0 && len(def.Steps) > 0 {
					return nil, fmt.Errorf("definition `script` keyword cannot be used with the `steps` keyword")
				}
				def.Steps = []*schema.Step{{
					Script: def.Script,
				}}
				def.Script = ""
				return def, nil
			}),
		}, {
			name: "compile schema definition into proto definition",
			rule: whenType[*schema.Definition](func(def *schema.Definition) (any, error) {
				if def.Script != "" {
					return nil, fmt.Errorf("definition `script` keyword was not compiled")
				}
				protoDef := &proto.Definition{}
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
						protoStep, err := compileTo[*proto.Step](s)
						if err != nil {
							return nil, fmt.Errorf("compiling definition steps[%v]: %q: %w", i, s.Name, err)
						}
						protoDef.Steps[i] = *protoStep
					}
				}
				return protoDef, nil
			}),
		},

		// Compiling *schema.Step

		{
			name: "step `script` keyword is compiled to a script step",
			rule: whenType[*schema.Step](func(step *schema.Step) (any, error) {
				if step.Script == "" {
					return step, nil
				}
				if step.Step != "" {
					return nil, fmt.Errorf("the `script` keyword cannot be used with the `step` keyword")
				}
				if len(step.Inputs) != 0 {
					return nil, fmt.Errorf("the `script` keyword cannot be used with `inputs`")
				}
				step.Step = "gitlab.com/gitlab-org/components/script@1.0"
				step.Inputs = map[string]*schema.Value{
					"script": schema.StringValue(step.Script),
				}
				step.Script = ""
				return step, nil
			}),
		}, {
			name: "compile schema step into proto step",
			rule: whenType[*schema.Step](func(step *schema.Step) (any, error) {
				protoInputs := map[string]*structpb.Value{}
				for k, v := range step.Inputs {
					protoValue, err := compileTo[*structpb.Value](v)
					if err != nil {
						return nil, err
					}
					protoInputs[k] = *protoValue
				}
				return &proto.Step{
					Name:   step.Name,
					Env:    step.Env,
					Inputs: protoInputs,
				}, nil
			}),
		},

		// Compiling schema.Input

		{
			name: "compile schema input into proto input",
			rule: whenType[schema.Input](func(input schema.Input) (any, error) {
				protoInput := &proto.Spec_Content_Input{}
				switch input.Type {
				case schema.ValueTypeBool:
					protoInput.Type = proto.InputType_bool
				case schema.ValueTypeNumber:
					protoInput.Type = proto.InputType_number
				case schema.ValueTypeString:
					protoInput.Type = proto.InputType_string
				case schema.ValueTypeStruct:
					protoInput.Type = proto.InputType_struct

				default:
					return nil, fmt.Errorf("unsupported input type: %v", input.Type)
				}
				protoV, err := compileTo[*structpb.Value](input.Default)
				if err != nil {
					return nil, fmt.Errorf("compiling default type %v: %v: %w", input.Type, input.Default, err)
				}
				protoInput.Default = *protoV
				return protoInput, nil
			}),
		},

		// Compiling schema.Output

		{
			name: "compile schema input into proto input",
			rule: whenType[schema.Output](func(output schema.Output) (any, error) {
				protoOutput := &proto.Spec_Content_Output{}
				protoV, err := compileTo[*structpb.Value](output.Default)
				if err != nil {
					return nil, fmt.Errorf("compiling default type %v: %v: %w", output.Type, output.Default, err)
				}
				return protoOutput, nil
			}),
		},

		// Compiling *schema.Value to proto.Value

		{
			name: "compile schema value into proto value",
			rule: whenType[*schema.Value](func(value *schema.Value) (any, error) {
				switch value.Type {
				case schema.ValueTypeBool:
					return structpb.NewBoolValue(value.Bool), nil
				case schema.ValueTypeNumber:
					return structpb.NewNumberValue(value.Number), nil
				case schema.ValueTypeString:
					return structpb.NewStringValue(value.String), nil
				case schema.ValueTypeStruct:
					protoValue := &structpb.Struct{
						Fields: map[string]*structpb.Value{},
					}
					for k, v := range value.Struct {
						protoV, err := compileTo[*structpb.Value](v)
						if err != nil {
							return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
						}
						protoValue.Fields[k] = *protoV
					}
					return structpb.NewStructValue(protoValue), nil
				case schema.ValueTypeList:
					protoValue := &structpb.ListValue{
						Values: make([]*structpb.Value, len(value.List)),
					}
					for i, v := range value.List {
						protoV, err := compileTo[*structpb.Value](v)
						if err != nil {
							return nil, fmt.Errorf("compiling input[%v]: %v: %w", i, v, err)
						}
						protoValue.Values[i] = *protoV
					}
					return structpb.NewListValue(protoValue), nil
				default:
					return nil, fmt.Errorf("unsupported schema value type %T", value)
				}
			}),
		},
	}
}

type compilerFn func(any) (any, error)

type compilerRule struct {
	name string
	rule compilerFn
}

func whenType[K any](apply func(value K) (any, error)) compilerFn {
	return func(value any) (any, error) {
		v, ok := value.(K)
		if !ok {
			return value, nil
		}
		return apply(v)
	}
}

func toValue(v any) *schema.Value {
	return nil
}
