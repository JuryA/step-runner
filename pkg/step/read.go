package step

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"

	"gitlab.com/gitlab-org/step-runner/proto"
)

const defaultStorePerm = 0o640

func LoadSpecDef(filename string) (*proto.Spec, *proto.Definition, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("reading file: %w", err)
	}

	return ReadSpecDef(string(buf))
}

func ReadSpecDef(stepDefinitionYAML string) (*proto.Spec, *proto.Definition, error) {
	var (
		spec proto.Spec
		def  proto.Definition
	)

	if err := unmarshal(stepDefinitionYAML, &spec, &def); err != nil {
		return nil, nil, err
	}

	return &spec, &def, nil
}

func StoreSpecDef(spec *proto.Spec, def *proto.Definition, filename string) error {
	encoded, err := WriteSpecDef(spec, def)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(encoded), defaultStorePerm)
}

func WriteSpecDef(spec *proto.Spec, def *proto.Definition) (string, error) {
	return marshal(spec, def)
}

func LoadSteps(filename string) (*proto.Definition, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return ReadSteps(string(buf))
}

func ReadSteps(stepsYAML string) (*proto.Definition, error) {
	var (
		def proto.Definition
	)

	if err := unmarshal(stepsYAML, &def); err != nil {
		return nil, err
	}

	return &def, nil
}

func StoreSteps(def *proto.Definition, filename string) error {
	encoded, err := WriteSteps(def)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(encoded), defaultStorePerm)
}

func WriteSteps(def *proto.Definition) (string, error) {
	if def.Type != proto.DefinitionType_steps {
		return "", fmt.Errorf("want a definition of type steps. got %v", def.Type)
	}

	return marshal(def)
}

func unmarshal(input string, subjects ...protoreflect.ProtoMessage) error {
	d := yaml.NewDecoder(strings.NewReader(input))
	d.KnownFields(true)

	for _, subject := range subjects {
		var decoded any
		err := d.Decode(&decoded)
		if err != nil {
			return fmt.Errorf("decoding: %w", err)
		}

		// convert to json
		encoded, err := json.Marshal(decoded)
		if err != nil {
			return fmt.Errorf("converting to json: %w", err)
		}

		// convert to proto
		if err := protojson.Unmarshal(encoded, subject); err != nil {
			return fmt.Errorf("converting to proto: %w", err)
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
