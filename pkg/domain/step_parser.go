package domain

type StepParser interface {
	Parse(rawSteps string) (Step, error)
}
