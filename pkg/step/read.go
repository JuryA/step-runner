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
)

func LoadProto(filename string) (*proto.SpecDefinition, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return ReadProto(string(buf), filepath.Dir(filename))
}

func ReadProto(content, dir string) (*proto.SpecDefinition, error) {
	var (
		spec       proto.Spec
		definition proto.Definition
	)

	if err := unmarshalProto(content, &spec, &definition); err != nil {
		return nil, fmt.Errorf("unmarshaling proto: %w", err)
	}
	stepDef := &proto.SpecDefinition{
		Spec:       &spec,
		Definition: &definition,
		Dir:        dir,
	}
	if err := ValidateStepDefinition(stepDef); err != nil {
		return nil, err
	}
	return stepDef, nil
}

func unmarshalProto(input string, subjects ...protoreflect.ProtoMessage) error {
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
