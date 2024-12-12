package runner

import (
	"os"
	"sync"

	"gitlab.com/gitlab-org/step-runner/proto"
)

// Singleton breakpoint object for entire step-runner instance
var Breakpoint *Bp = &Bp{
	atPoint: make(chan struct{}),
	release: make(chan struct{}),
}

func init() {
	if os.Getenv("STEP_RUNNER_DEBUG_STOP") == "true" {
		return // begin stopped
	}
	close(Breakpoint.release) // begin started
}

type Bp struct {
	mux            sync.Mutex
	atSpecDef      *proto.SpecDefinition
	atStepsContext *StepsContext
	atPoint        chan struct{}
	release        chan struct{}
}

func (b *Bp) At(specDef *proto.SpecDefinition, stepsContext *StepsContext) {
	// This should be in another package but we don't have a non-runner context (e.g. proto.Context)
	b.mux.Lock()
	b.atSpecDef = specDef
	b.atStepsContext = stepsContext
	b.mux.Unlock()
	close(b.atPoint) // currently at a breakpoint
	<-b.release
	b.mux.Lock()
	b.atSpecDef = nil
	b.atStepsContext = nil
	b.atPoint = make(chan struct{}) // leaving breakpoint
	b.mux.Unlock()
}

func (b *Bp) Stop() {
	b.mux.Lock()
	b.release = make(chan struct{}) // stop at next breakpoint
	b.mux.Unlock()
}

func (b *Bp) Continue() {
	b.mux.Lock()
	b.atSpecDef = nil
	b.atStepsContext = nil
	b.mux.Unlock()
	close(b.release) // release all breakpoints
}

func (b *Bp) Step() {
	b.mux.Lock()
	b.atSpecDef = nil
	b.atStepsContext = nil
	b.mux.Unlock()
	b.release <- struct{}{} // release to next breakpoint
}

func (b *Bp) Next() {
	// not implemented
	b.Step()
}

func (b *Bp) State() (*proto.SpecDefinition, *StepsContext) {
	<-b.atPoint // wait for next breakpoint
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.atSpecDef, b.atStepsContext
}
