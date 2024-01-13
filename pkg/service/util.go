package service

import (
	stdctx "context"
	"sync"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// Buf is intended to buffer everything written to it AND also forward it to the embedded channel.
type Buf struct {
	buffer []byte
	c      chan []byte
	lock   sync.RWMutex
	once   sync.Once
}

func newBuf() Buf { return Buf{c: make(chan []byte)} }

// Write buffers p AND writes it to the embedded channel. Note: Write will block on the channel write, so something had
// better be reading it at the other end.
func (b *Buf) Write(p []byte) (int, error) {
	b.lock.Lock()
	b.buffer = append(b.buffer, p...)
	b.lock.Unlock()

	// TODO: this won't always work. If no client has called FollowIO this will block, and if more that one client has
	// called FollowIO, which once receives each write to the channel is non-deterministic. We need one channel per
	// client that called FollowIO (including 0), or to require a since well-behaved client.
	b.c <- p
	return len(p), nil
}

// Read returns all the writes buffered so far, starting from offset. If offset is out of range Read will return a nil
// slice.
func (b *Buf) Read(offset int32) []byte {
	b.lock.RLock()
	defer b.lock.RUnlock()

	// if the offset is out of range, just return a nil slice.
	if int(offset) >= len(b.buffer) {
		return nil
	}

	return b.buffer[offset:]
}

// Close closes the embedded channel exactly once.
func (b *Buf) Close() {
	b.once.Do(func() {
		close(b.c)
	})
}

// Job represents an active request to run (and follow and cancel) a collection of steps.
type Job struct {
	id      string                 // The ID of the job to run/being run. Must be unique. Typically this will be the CI job ID.
	globCtx *context.Global        // To capture stdout/err from all subprocesses
	ctx     stdctx.Context         // The context used to manage the Job's entire lifetime.
	cancel  func()                 // Used to cancel the ctx.
	err     error                  // Captures any error returned when executing steps.
	once    sync.Once              // Used to ensure channels are only closed once.
	results chan *proto.StepResult // To capture the step-results produced by Run so Follow can get at them.
	stdout  Buf                    // To capture stdout produced by processes FollowIO can get at them.
	stderr  Buf                    // To capture stdout produced by processes FollowIO can get at them.
}

// finsh() cleans up all resources associated with managing a job
func (j *Job) finish() {
	j.once.Do(func() {
		close(j.results)
		j.stdout.Close()
		j.stderr.Close()
	})
}

// DevNullChan is a channel that discards everything written to it and never blocks. Use this in places where a
// write-channel is expected but you don't care about the data written to the channel.
type DevNullChan[T any] struct {
	sink chan T
	once sync.Once
}

func NewDevNullChan[T any]() *DevNullChan[T] {
	dnc := DevNullChan[T]{
		sink: make(chan T),
	}
	go dnc.discard()
	return &dnc
}

func (dnc *DevNullChan[T]) Sink() chan<- T {
	return dnc.sink
}

func (dnc *DevNullChan[T]) Close() {
	dnc.once.Do(func() {
		close(dnc.sink)
	})
}

func (dnc *DevNullChan[T]) discard() {
	for range dnc.sink {
	}
}

type Output struct {
	Stdout, Stderr chan<- []byte
	StepResult     chan<- *proto.StepResult
}
