package schema

import (
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"

	_ "embed"
)

var (
	//go:embed spec.json
	specSchemaString string
	//go:embed step.json
	stepSchemaString string

	specSchema *jsonschema.Schema
	stepSchema *jsonschema.Schema
)

func init() {
	specSchema = jsonschema.MustCompileString("spec.json", specSchemaString)
	stepSchema = jsonschema.MustCompileString("step.json", stepSchemaString)
}

func validateSpec(spec Spec) error {
	untyped, err := untype(spec)
	if err != nil {
		return err
	}
	return specSchema.Validate(untyped)
}

func validateStep(step Step) error {
	untyped, err := untype(step)
	if err != nil {
		return err
	}
	return stepSchema.Validate(untyped)
}

func untype(v any) (any, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var untyped any
	err = yaml.Unmarshal(data, &untyped)
	if err != nil {
		return nil, err
	}
	return untyped, nil
}
