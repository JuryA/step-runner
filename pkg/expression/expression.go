package expression

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func InterpolateInputs(globalCtx *context.Global, stepsCtx *context.Steps, step *proto.Step) error {
	matches := merge(globalMatches(globalCtx), outputMatches(stepsCtx))
	return replaceInputMatches(matches, step)
}

func InterpolateOutputs(globalCtx *context.Global, stepsCtx *context.Steps, spec *proto.Spec_Content, def *proto.Definition) (map[string]string, error) {
	matches := merge(globalMatches(globalCtx), outputMatches(stepsCtx))
	return replaceOutputMatches(matches, spec, def)
}

func InterpolateExec(globalCtx *context.Global, inputs map[string]*structpb.Value, spec *proto.Spec_Content, exec *proto.Definition_Exec) error {
	matches := globalMatches(globalCtx)
	types := map[string]proto.InputType{}
	match := func(k string, t proto.InputType, v *structpb.Value) error {
		key := "${{inputs." + k + "}}"
		switch v.Kind.(type) {
		case *structpb.Value_StringValue:
			if t != proto.InputType_string {
				return fmt.Errorf("want type %v. got string", t)
			}
			matches[key] = v.GetStringValue()
		case *structpb.Value_NumberValue:
			if t != proto.InputType_number {
				return fmt.Errorf("want type %v. got number", t)
			}
			n, _ := json.Marshal(v.GetNumberValue())
			matches[key] = string(n)
		case *structpb.Value_BoolValue:
			if t != proto.InputType_bool {
				return fmt.Errorf("want type %v. got bool", t)
			}
			b, _ := json.Marshal(v.GetBoolValue())
			matches[key] = string(b)
		case *structpb.Value_StructValue:
			if t != proto.InputType_struct {
				return fmt.Errorf("want type %v. got struct", t)
			}
			s, err := json.Marshal(v.GetStructValue())
			if err != nil {
				return fmt.Errorf("json marshaling struct value %v", v)
			}
			matches[key] = string(s)
		default:
			return fmt.Errorf("unsupported type %T", v.Kind)
		}
		return nil
	}
	for k, v := range spec.Inputs {
		types[k] = v.Type
		if v.Default != nil {
			err := match(k, v.Type, v.Default)
			if err != nil {
				return fmt.Errorf("setting default for %q: %w", k, err)
			}
		}
	}
	for k, v := range inputs {
		t, ok := types[k]
		if !ok {
			return fmt.Errorf("undefined input %q", k)
		}
		err := match(k, t, v)
		if err != nil {
			return fmt.Errorf("setting input %q: %w", k, err)
		}
	}
	return replaceDefinitionMatches(matches, exec)
}

func replaceDefinitionMatches(matches map[string]string, exec *proto.Definition_Exec) error {
	for i := range exec.Command {
		for match, value := range matches {
			exec.Command[i] = strings.ReplaceAll(exec.Command[i], match, value)
		}
	}
	for match, value := range matches {
		exec.WorkDir = strings.ReplaceAll(exec.WorkDir, match, value)
	}
	return nil
}

func replaceInputMatches(matches map[string]string, step *proto.Step) error {
	for k := range step.Env {
		for match, value := range matches {
			step.Env[k] = strings.ReplaceAll(step.Env[k], match, value)
		}
	}
	for k, v := range step.Inputs {
		step.Inputs[k] = replaceAll(v, matches)
	}
	return nil
}

func replaceOutputMatches(matches map[string]string, spec *proto.Spec_Content, def *proto.Definition) (map[string]string, error) {
	outputs := map[string]string{}
	for k := range spec.Outputs {
		outputs[k] = spec.Outputs[k].Default
	}
	for k := range def.Outputs {
		if _, ok := spec.Outputs[k]; !ok {
			return nil, fmt.Errorf("undeclared output: %v", k)
		}
		outputs[k] = def.Outputs[k]
		for match, value := range matches {
			outputs[k] = strings.ReplaceAll(outputs[k], match, value)
		}
	}
	return outputs, nil
}

func replaceAll(v *structpb.Value, matches map[string]string) *structpb.Value {
	s := v.GetStringValue()
	if s != "" {
		for match, value := range matches {
			s = strings.ReplaceAll(s, match, value)
		}
		return structpb.NewStringValue(s)
	}
	str := v.GetStructValue()
	if str != nil {
		for k, v := range str.Fields {
			str.Fields[k] = replaceAll(v, matches)
		}
		return structpb.NewStructValue(str)
	}
	l := v.GetListValue()
	if l != nil {
		for i, v := range l.Values {
			l.Values[i] = replaceAll(v, matches)
		}
		return structpb.NewListValue(l)
	}
	return v
}

func merge(to, from map[string]string) map[string]string {
	for k, v := range from {
		to[k] = v
	}
	return to
}

func globalMatches(globalCtx *context.Global) map[string]string {
	if globalCtx == nil {
		return map[string]string{}
	}
	matches := map[string]string{}
	for k, v := range globalCtx.Job {
		matches["${{job."+k+"}}"] = v
	}
	for k, v := range globalCtx.Env {
		matches["${{env."+k+"}}"] = v
	}
	return matches
}

func outputMatches(stepsCtx *context.Steps) map[string]string {
	if stepsCtx == nil {
		return map[string]string{}
	}
	matches := map[string]string{}
	for name, outputs := range stepsCtx.Outputs {
		for k, v := range outputs {
			matches["${{steps."+name+".outputs."+k+"}}"] = v
		}
	}
	return matches
}
