package step

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema"
	"google.golang.org/protobuf/types/known/structpb"
)

func Compile(def *schema.Definition) (*proto.Definition, error) {
	protoDef, err := compileTo[*proto.Definition](def)
	if err != nil {
		return nil, err
	}
	return *protoDef, nil
}

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
		return nil, fmt.Errorf("failed to compile to %T", result)
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

		// Compiling *schema.Definition

		{
			name: "top level `script` keyword is compiled into a single script step",
			rule: whenType[*schema.Definition](func(def *schema.Definition) (any, error) {
				if len(def.Steps) != 0 {
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
				if def.Container != "" {
					return nil, fmt.Errorf("definition `container` keyword was not compiled")
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
				step.Inputs = map[string]schema.Value{
					"script": schema.StringValue(step.Script),
				}
				step.Script = ""
				return step, nil
			}),
		}, {
			name: "step `container` keyword is compiled to a docker run step",
			rule: whenType[*schema.Step](func(step *schema.Step) (any, error) {
				if step.Container == "" {
					return step, nil
				}
				container := step.Container
				step.Container = ""
				stepValue := toValue(step)
				return &schema.Step{
					Name: "run in container " + container,
					Step: "gitlab.com/gitlab-org/components/docker/run@1.0",
					Inputs: map[string]schema.Value{
						"step": stepValue,
					},
				}, nil
			}),
		}, {
			name: "compile schema step into proto step",
			rule: whenType[*schema.Step](func(step *schema.Step) (any, error) {
				protoInputs := map[string]*structpb.Value{}
				for k, v := range step.Inputs {
					protoInput, err := compileTo[*structpb.Value](v)
					if err != nil {
						return nil, err
					}
					protoInputs[k] = *protoInput
				}
				return &proto.Step{
					Name:   step.Name,
					Env:    step.Env,
					Inputs: protoInputs,
				}, nil
			}),
		},

		// Compiling *schema.Input

		{
			name: "compile schema value into proto value",
			rule: whenType[schema.Value](func(value schema.Value) (any, error) {
				switch value := value.(type) {
				case schema.NullValue:
					return structpb.NewNullValue(), nil
				case schema.BoolValue:
					return structpb.NewBoolValue(bool(value)), nil
				case schema.NumberValue:
					return structpb.NewNumberValue(float64(value)), nil
				case schema.StringValue:
					return structpb.NewStringValue(string(value)), nil
				case schema.StructValue:
					protoValue := &structpb.Struct{
						Fields: map[string]*structpb.Value{},
					}
					for k, v := range value {
						protoV, err := compileTo[*structpb.Value](v)
						if err != nil {
							return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
						}
						protoValue.Fields[k] = *protoV
					}
					return structpb.NewStructValue(protoValue), nil

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

func toValue(v any) schema.Value {
	return nil
}
