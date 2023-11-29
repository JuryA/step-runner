package step

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

func LoadSpecDef(filename string) (*proto.Spec, *proto.Definition, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("opening yaml file %v: %w", filename, err)
	}
	defer f.Close()
	var (
		spec any
		def  any
	)
	err = readYAML(f, &spec, &def)
	if err != nil {
		return nil, nil, fmt.Errorf("loading step definition: %w", err)
	}
	return specDefToProto(spec, def)
}

func ReadSpecDef(stepDefinitionYAML string) (*proto.Spec, *proto.Definition, error) {
	r := strings.NewReader(stepDefinitionYAML)
	var (
		spec any
		def  any
	)
	err := readYAML(r, &spec, &def)
	if err != nil {
		return nil, nil, fmt.Errorf("reading step spec and def: %w", err)
	}
	return specDefToProto(spec, def)
}

func specDefToProto(spec, def any) (*proto.Spec, *proto.Definition, error) {
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("converting spec to json: %w", err)
	}
	s := proto.Spec{}
	err = protojson.Unmarshal(specJSON, &s)
	if err != nil {
		return nil, nil, fmt.Errorf("converting spec to proto: %w", err)
	}
	defJSON, err := json.Marshal(def)
	if err != nil {
		return nil, nil, fmt.Errorf("converting def to json: %w", err)
	}
	d := proto.Definition{}
	err = protojson.Unmarshal(defJSON, &d)
	if err != nil {
		return nil, nil, fmt.Errorf("converting def to proto: %w", err)
	}
	return &s, &d, nil
}

func StoreSpecDef(spec *proto.Spec, def *proto.Definition, filename string) error {
	specAny, defAny, err := protoToSpecDef(spec, def)
	if err != nil {
		return err
	}
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening output file %v: %w", filename, err)
	}
	defer f.Close()
	err = writeYaml(f, specAny, defAny)
	if err != nil {
		return fmt.Errorf("storing step spec and def: %w", err)
	}
	return nil
}

func WriteSpecDef(spec *proto.Spec, def *proto.Definition) (string, error) {
	specAny, defAny, err := protoToSpecDef(spec, def)
	if err != nil {
		return "", fmt.Errorf("writing step spec and def: %w", err)
	}
	w := &strings.Builder{}
	err = writeYaml(w, specAny, defAny)
	if err != nil {
		return "", fmt.Errorf("writing step spec and def yaml: %w", err)
	}
	return w.String(), nil
}

func protoToSpecDef(spec *proto.Spec, def *proto.Definition) (any, any, error) {
	specJSON, err := protojson.Marshal(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("converting spec to json: %w", err)
	}
	var s any
	err = json.Unmarshal(specJSON, &s)
	if err != nil {
		return nil, nil, fmt.Errorf("converting spec to any: %w", err)
	}
	defJSON, err := protojson.Marshal(def)
	if err != nil {
		return nil, nil, fmt.Errorf("conversting def to json: %w", err)
	}
	var d any
	err = json.Unmarshal(defJSON, &d)
	if err != nil {
		return nil, nil, fmt.Errorf("converting def to any: %w", err)
	}
	return s, d, nil
}

func LoadSteps(filename string) (*proto.Definition, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening yaml file %v: %w", filename, err)
	}
	var steps any
	defer f.Close()
	err = readYAML(f, &steps)
	if err != nil {
		return nil, fmt.Errorf("loading steps: %w", err)
	}
	return stepsToProto(steps)
}

func ReadSteps(stepsYAML string) (*proto.Definition, error) {
	r := strings.NewReader(stepsYAML)
	var steps any
	err := readYAML(r, &steps)
	if err != nil {
		return nil, fmt.Errorf("reading steps: %w", err)
	}
	return stepsToProto(steps)
}

func stepsToProto(steps any) (*proto.Definition, error) {
	defJSON, err := json.Marshal(steps)
	if err != nil {
		return nil, fmt.Errorf("converting def to json: %w", err)
	}
	d := proto.Definition{}
	err = protojson.Unmarshal(defJSON, &d)
	if err != nil {
		return nil, fmt.Errorf("converting def to proto: %w", err)
	}
	if d.Type != proto.DefinitionType_steps {
		return nil, fmt.Errorf("want a definition of type steps. got %v", d.Type)
	}
	return &d, nil
}

func StoreSteps(def *proto.Definition, filename string) error {
	if def.Type != proto.DefinitionType_steps {
		return fmt.Errorf("want a definition of type steps. got %v", def.Type)
	}
	defAny, err := protoToDef(def)
	if err != nil {
		return err
	}
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening output file %v: %w", filename, err)
	}
	defer f.Close()
	err = writeYaml(f, defAny)
	if err != nil {
		return fmt.Errorf("storing step definition: %w", err)
	}
	return nil
}

func WriteSteps(def *proto.Definition) (string, error) {
	if def.Type != proto.DefinitionType_steps {
		return "", fmt.Errorf("want a definition of type steps. got %v", def.Type)
	}
	defAny, err := protoToDef(def)
	if err != nil {
		return "", fmt.Errorf("writing step def: %w", err)
	}
	w := &strings.Builder{}
	err = writeYaml(w, defAny)
	if err != nil {
		return "", fmt.Errorf("writing step def: %w", err)
	}
	return w.String(), nil
}

func protoToDef(def *proto.Definition) (any, error) {
	defJSON, err := protojson.Marshal(def)
	if err != nil {
		return nil, fmt.Errorf("converting def to json: %w", err)
	}
	var d any
	err = json.Unmarshal(defJSON, &d)
	if err != nil {
		return nil, fmt.Errorf("converting def to any: %w", err)
	}
	return &d, nil
}

func readYAML(reader io.Reader, subjects ...any) (err error) {
	d := yaml.NewDecoder(reader)
	d.KnownFields(true)
	for _, subject := range subjects {
		err := d.Decode(subject)
		if err != nil {
			return fmt.Errorf("decoding yaml file: %w", err)
		}
	}
	return nil
}

func writeYaml(writer io.Writer, subjects ...any) (err error) {
	d := yaml.NewEncoder(writer)
	for _, subject := range subjects {
		err := d.Encode(subject)
		if err != nil {
			return fmt.Errorf("encoding yaml file: %w", err)
		}
	}
	return nil
}
