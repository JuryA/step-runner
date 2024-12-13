package runner

import (
	"os"
	"sync"

	"gitlab.com/gitlab-org/step-runner/proto"
)

// Singleton breakpoint object for entire step-runner instance
var Breakpoint *Bp = &Bp{
	state:     Running,
	listeners: []chan View{},
}

func init() {
	if os.Getenv("STEP_RUNNER_DEBUG_STOP") == "true" {
		Breakpoint.state = Stopping
	}
}

type State int

const (
	Running State = iota
	Stopping
	Stopped
	Stepping
)

type Point struct {
	AtSpecDef      *proto.SpecDefinition
	AtStepsContext *StepsContext
	release        chan struct{}
}

type View struct {
	Point *Point
	State State
}

type Bp struct {
	mux        sync.Mutex
	state      State
	foreground *Point
	background []*Point
	listeners  []chan View
}

// At is for step-runner to call when it arrives at interesting
// places.
func (b *Bp) At(specDef *proto.SpecDefinition, stepsContext *StepsContext) {
	// This should be in another package but we don't have a non-runner context (e.g. proto.Context)
	point := &Point{
		AtSpecDef:      specDef,
		AtStepsContext: stepsContext,
		release:        make(chan struct{}),
	}
	b.mux.Lock()
	switch b.state {
	case Running:
		// Just keep running past this break point
		close(point.release)
	case Stopping:
		// Stop and place this point in the foreground
		b.foreground = point
		b.state = Stopped
		b.broadcast()
	case Stopped:
		//
		b.background = append(b.background, point)
	}
	b.mux.Unlock()
	<-point.release
}

// Stop will stop all requests at their next breakpoint. The first
// request to stop will be in the foreground.
func (b *Bp) Stop() {
	b.mux.Lock()
	defer b.mux.Unlock()
	if b.state == Stopped {
		return
	}
	b.state = Stopping
	b.broadcast()
}

// Continue will release all breakpoints.
func (b *Bp) Continue() {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.state = Running
	b.releaseAll()
	b.broadcast()
}

// Step will release only the foreground breakpoint.
func (b *Bp) Step() {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.state = Stopping
	b.releaseForeground()
	b.broadcast()
}

func (b *Bp) Next() {
	// Not implemented. Needs tracking breakpoints by request path or source location.
	b.Step()
}

// State is for the debug server to call when it is waiting on the
// results of a command or just wants an updated view.
func (b *Bp) State() chan View {
	viewCh := make(chan View, 1)
	b.mux.Lock()
	defer b.mux.Unlock()
	if b.state == Stopped {
		viewCh <- View{
			Point: b.foreground,
			State: b.state,
		}
	} else {
		b.listeners = append(b.listeners, viewCh)
	}
	return viewCh
}

func (b *Bp) broadcast() {
	for _, l := range b.listeners {
		l <- View{
			Point: b.foreground,
			State: b.state,
		}
	}
	b.listeners = nil
}

func (b *Bp) releaseForeground() {
	if b.foreground != nil {
		close(b.foreground.release)
		b.foreground = nil
	}
}

func (b *Bp) releaseAll() {
	b.releaseForeground()
	for _, p := range b.background {
		close(p.release)
	}
	b.background = nil
}
