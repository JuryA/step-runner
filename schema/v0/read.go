package schema

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadSteps(filename string) (*StepDefinition, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return ReadSteps(string(buf), filepath.Dir(filename))
}

func ReadSteps(content, dir string) (*StepDefinition, error) {
	var (
		spec       Spec
		definition Definition
	)

	if err := unmarshalSchema(content, &spec, &definition); err != nil {
		return nil, fmt.Errorf("unmarshaling: %w", err)
	}

	return &StepDefinition{
		Spec:       &spec,
		Definition: &definition,
		Dir:        dir,
	}, nil
}

func WriteSteps(stepDef *StepDefinition) (string, error) {
	var buf bytes.Buffer
	e := yaml.NewEncoder(&buf)

	err := e.Encode(stepDef.Spec)
	if err != nil {
		return "", fmt.Errorf("encoding spec: %w", err)
	}

	err = e.Encode(stepDef.Definition)
	if err != nil {
		return "", fmt.Errorf("encoding definition: %w", err)
	}

	err = e.Close()
	if err != nil {
		return "", fmt.Errorf("closing: %w", err)
	}
	return buf.String(), nil
}

func unmarshalSchema(input string, subjects ...any) error {
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
