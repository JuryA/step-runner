package expression

import (
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
)

func InterpolateString(stepsCtx *context.Steps, value string) string {
	matches, err := convertValuesToInterpolationReplace(stepsCtx.GetMatches())
	if err != nil {
		// TODO: Error handling
		return ""
	}
	return replaceAll(structpb.NewStringValue(value), matches).GetStringValue()
}

func InterpolateProtoValue(stepsCtx *context.Steps, value *structpb.Value) *structpb.Value {
	matches, err := convertValuesToInterpolationReplace(stepsCtx.GetMatches())
	if err != nil {
		// TODO: Error handling
		return structpb.NewStringValue("")
	}
	return replaceAll(value, matches)
}

func convertValuesToInterpolationReplace(m map[string]*structpb.Value) (map[string]string, error) {
	r := make(map[string]string)
	for k, v := range m {
		str, err := ValueToString(v)
		if err != nil {
			return nil, err
		}
		r["${{"+k+"}}"] = str
	}
	return r, nil
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
