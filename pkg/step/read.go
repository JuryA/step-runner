package step

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema"
)

const defaultStorePerm = 0o640

func LoadSteps(filename string) (*schema.StepDefinition, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return ReadSteps(string(buf), filepath.Dir(filename))
}

func ReadSteps(content, dir string) (*schema.StepDefinition, error) {
	var (
		spec       schema.Spec
		definition schema.Definition
	)

	if err := unmarshal(content, &spec, &definition); err != nil {
		return nil, fmt.Errorf("unmarshaling: %w", err)
	}

	return &schema.StepDefinition{
		Spec:       &spec,
		Definition: &definition,
		Dir:        dir,
	}, nil
}

func LoadProto(filename string) (*proto.StepDefinition, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return ReadProto(string(buf), filepath.Dir(filename))
}

func ReadProto(content, dir string) (*proto.StepDefinition, error) {
	var (
		spec       proto.Spec
		definition proto.Definition
	)

	if err := unmarshal(content, &spec, &definition); err != nil {
		return nil, fmt.Errorf("unmarshaling proto: %w", err)
	}
	stepDef := &proto.StepDefinition{
		Spec:       &spec,
		Definition: &definition,
		Dir:        dir,
	}
	if err := ValidateStepDefinition(stepDef); err != nil {
		return nil, err
	}
	return stepDef, nil
}

func unmarshal(input string, subjects ...any) error {
	d := yaml.NewDecoder(strings.NewReader(input))
	d.KnownFields(true)

	for _, subject := range subjects {
		err := d.Decode(subject)
		if err != nil {
			return fmt.Errorf("decoding: %w", err)
		}
	}

	return nil
}

func marshalProto(subjects ...protoreflect.ProtoMessage) (string, error) {
	var sb strings.Builder
	d := yaml.NewEncoder(&sb)

	for _, subject := range subjects {
		encoded, err := protojson.Marshal(subject)
		if err != nil {
			return "", fmt.Errorf("converting to json: %w", err)
		}

		var val any
		if err := json.Unmarshal(encoded, &val); err != nil {
			return "", fmt.Errorf("unmarshaling: %w", err)
		}

		if err := d.Encode(val); err != nil {
			return "", fmt.Errorf("marshal: %w", err)
		}
	}

	return sb.String(), nil
}
