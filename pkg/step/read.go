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

func Read(filename string) (*proto.StepDefinition, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return Compile(string(buf), filepath.Dir(filename))
}

func Compile(content, dir string) (*proto.StepDefinition, error) {
	var (
		spec       schema.Spec
		definition schema.Definition
	)

	if err := unmarshal(content, &spec, &definition); err != nil {
		return nil, fmt.Errorf("unmarshaling: %w", err)
	}

	protoSpec, err := compileTo[*proto.Spec](&spec)
	if err != nil {
		return nil, fmt.Errorf("compiling spec: %w", err)
	}
	protoDef, err := compileTo[*proto.Definition](&definition)
	if err != nil {
		return nil, fmt.Errorf("compiling definition: %w", err)
	}

	stepDef := &proto.StepDefinition{
		Spec:       *protoSpec,
		Definition: *protoDef,
		Dir:        dir,
	}
	if err := ValidateStepDefinition(stepDef); err != nil {
		return nil, err
	}
	return stepDef, nil
}

func Write(stepDef *proto.StepDefinition, filename string) error {
	encoded, err := Serialize(stepDef)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(encoded), defaultStorePerm)
}

func Serialize(stepDef *proto.StepDefinition) (string, error) {
	return marshal(stepDef.Spec, stepDef.Definition)
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

func marshal(subjects ...protoreflect.ProtoMessage) (string, error) {
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
