package step

import (
	"fmt"
	"regexp"

	"gitlab.com/gitlab-org/step-runner/proto"
)

// TODO replace this validation with a JSON schema for the steps model with syntactic sugar.

func ValidateStepDefinition(stepDef *proto.StepDefinition) error {
	errs := validateStepDefinitionIdentifiers(stepDef)
	if len(errs) > 0 {
		return fmt.Errorf("errors validating step definition: %v", errs)
	}
	return nil
}

func validateStepDefinitionIdentifiers(stepDef *proto.StepDefinition) []error {
	errs := []error{}
	validateIdentifier := func(k string) {
		if !isAlphaNumeric(k) {
			errs = append(errs, fmt.Errorf("identifer must be alphanumeric. got %q", k))
		}
	}
	if stepDef.Spec != nil && stepDef.Spec.Spec != nil {
		for k := range stepDef.Spec.Spec.Inputs {
			validateIdentifier(k)
		}
		for k := range stepDef.Spec.Spec.Outputs {
			validateIdentifier(k)
		}
	}
	if stepDef.Definition != nil {
		if stepDef.Definition != nil && stepDef.Definition.Steps != nil {
			for _, s := range stepDef.Definition.Steps {
				validateIdentifier(s.Name)
				for k := range s.Env {
					validateIdentifier(k)
				}
				for k := range s.Inputs {
					validateIdentifier(k)
				}
			}
			for k := range stepDef.Definition.Outputs {
				validateIdentifier(k)
			}
		}
	}
	return errs
}

var alphaNumeric *regexp.Regexp

func init() {
	alphaNumeric = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
}

func isAlphaNumeric(k string) bool {
	return alphaNumeric.MatchString(k)
}
