package step

import (
	"gitlab.com/gitlab-org/step-runner/pkg/domain"
	"gitlab.com/gitlab-org/step-runner/pkg/domain/resource"
)

type StepFactory interface {
	CreateLazilyLoadedStep(name string, resource resource.Resource) domain.Step
}
