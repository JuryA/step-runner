package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/protobuf/encoding/protojson"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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
	SpecDef      *proto.SpecDefinition
	StepsContext *StepsContext
	release      chan struct{}
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
		SpecDef:      specDef,
		StepsContext: stepsContext,
		release:      make(chan struct{}),
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

func (b *Bp) Set(path string, value *structpb.Value) error {
	b.mux.Lock()
	defer b.mux.Unlock()
	if b.foreground == nil {
		return fmt.Errorf("no foreground breakpoint")
	}
	if b.foreground.SpecDef == nil {
		return fmt.Errorf("no foreground spec definition (context not yet supported)")
	}
	untypedContainer, err := untypeProto(b.foreground.SpecDef)
	if err != nil {
		return err
	}
	untypedValue, err := untypeProto(value)
	if err != nil {
		return err
	}
	keys := strings.Split(path, ".")
	key := keys[0]
	var rest []string
	if len(path) > 1 {
		rest = keys[1:]
	}
	untypedContainer, err = mutate(untypedContainer, key, rest, untypedValue)
	if err != nil {
		return err
	}
	err = retypeProto(untypedContainer, b.foreground.SpecDef)
	if err != nil {
		return err
	}
	return nil
}

func untypeProto(m protobuf.Message) (any, error) {
	bytes, err := protojson.Marshal(m)
	if err != nil {
		return nil, err
	}
	var untyped any
	err = json.Unmarshal(bytes, &untyped)
	if err != nil {
		return nil, err
	}
	return untyped, nil
}

func retypeProto(a any, m protobuf.Message) error {
	bytes, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(bytes, m)
}

// mutate a structure to set value at path
func mutate(a any, key string, rest []string, value any) (any, error) {
	switch a := a.(type) {
	case map[string]any:
		if len(rest) == 0 {
			a[key] = value
		} else {
			if _, ok := a[key]; !ok {
				fmt.Errorf("key %q does not exist", key)
			}
			child, err := mutate(a[key], rest[0], rest[1:], value)
			if err != nil {
				return nil, err
			}
			a[key] = child
		}
	case []any:
		i, err := strconv.Atoi(key)
		if err != nil {
			return nil, err
		}
		if len(a) < i-1 || i < 0 {
			return nil, fmt.Errorf("index out of bounds: %v", i)
		}
		if len(rest) == 0 {
			a[i] = value
		} else {
			child, err := mutate(a[i], rest[0], rest[1:], value)
			if err != nil {
				return nil, err
			}
			a[i] = child
		}
	default:
		return nil, fmt.Errorf("invalid key %q for type %T", key, a)
	}
	return a, nil
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
