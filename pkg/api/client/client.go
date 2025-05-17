package client

import (
	"time"

	"gitlab.com/gitlab-org/step-runner/proto"
)

// State is an enumerations of the states a step can be in during execution.
type State int32

const (
	StateUnspecified State = 0
	StateRunning     State = 1
	StateSuccess     State = 2
	StateFailure     State = 3
	StateCancelled   State = 4
)

var statusString = map[State]string{
	StateUnspecified: "unspecified",
	StateRunning:     "running",
	StateSuccess:     "success",
	StateFailure:     "failure",
	StateCancelled:   "cancelled",
}

func (s State) String() string { return statusString[s] }

type (
	// Status captures the overall status of a RunRequest execution
	Status struct {
		Id        string
		Message   string
		State     State
		StartTime time.Time
		EndTime   time.Time
		Result    *proto.StepResult
	}

	Variable struct {
		Key    string
		Value  string
		File   bool
		Masked bool
	}

	RunRequest struct {
		Id      string
		WorkDir string
		Env     map[string]string
		Steps   string

		Variables []Variable
		BuildDir  string
	}
)
