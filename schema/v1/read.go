package schema

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadSteps(filename string) (*Spec, *Step, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("reading file: %w", err)
	}

	return ReadSteps(string(buf))
}

func ReadSteps(content string) (*Spec, *Step, error) {
	var (
		spec Spec
		step Step
	)

	if err := unmarshalSchema(content, &spec, &step); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling: %w", err)
	}

	err := validateSpec(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("validating spec: %w", err)
	}
	err = validateStep(step)
	if err != nil {
		return nil, nil, fmt.Errorf("validating step: %w", err)
	}

	return &spec, &step, nil
}

func WriteSteps(spec *Spec, step *Step) (string, error) {
	var buf bytes.Buffer
	e := yaml.NewEncoder(&buf)

	err := e.Encode(spec)
	if err != nil {
		return "", fmt.Errorf("encoding spec: %w", err)
	}

	err = e.Encode(step)
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
