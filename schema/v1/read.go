package schema

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadSteps(filename string) (*StepFile, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return ReadSteps(string(buf), filepath.Dir(filename))
}

func ReadSteps(content, dir string) (*StepFile, error) {
	var (
		spec Spec
		step Step
	)

	if err := unmarshalSchema(content, &spec, &step); err != nil {
		return nil, fmt.Errorf("unmarshaling: %w", err)
	}

	return &StepFile{
		Spec: &spec,
		Step: &step,
		Dir:  dir,
	}, nil
}

func WriteSteps(stepFile *StepFile) (string, error) {
	var buf bytes.Buffer
	e := yaml.NewEncoder(&buf)

	err := e.Encode(stepFile.Spec)
	if err != nil {
		return "", fmt.Errorf("encoding spec: %w", err)
	}

	err = e.Encode(stepFile.Step)
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
