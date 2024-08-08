package schema

import (
	"fmt"

	"github.com/bufbuild/protovalidate-go"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func validateStepDefinition(stepDef *proto.SpecDefinition) error {
	v, err := protovalidate.New()
	if err != nil {
		return fmt.Errorf("failed to initialize validator: %w", err)
	}
	if err = v.Validate(stepDef); err != nil {
		return fmt.Errorf("error validating step definition: %w", err)
	}
	return nil
}
