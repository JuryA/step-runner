package service

import (
	stdctx "context"
	"os"
	"sync"
	"time"

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
	id         string                 // The ID of the job to run/being run. Must be unique. Typically this will be the CI job ID.
	globCtx    *context.Global        // To capture stdout/err from all subprocesses
	ctx        stdctx.Context         // The context used to manage the Job's entire lifetime.
	cancel     func()                 // Used to cancel the ctx.
	err        error                  // Captures any error returned when executing steps.
	results    chan *proto.StepResult // To capture the step-results produced by Run so Follow can get at them.
	stdout     Buf                    // To capture stdout produced by processes FollowIO can get at them.
	stderr     Buf                    // To capture stdout produced by processes FollowIO can get at them.
	finished   bool                   // indicated whether all processing on this job has finished
	finishTime time.Time
	sync.RWMutex
}

func NewJob(jobID, workDir string) *Job {
	ctx, cancel := stdctx.WithCancel(stdctx.Background())
	job := Job{
		id:      jobID,
		globCtx: context.NewGlobal(),
		ctx:     ctx,
		cancel:  cancel,
		results: make(chan *proto.StepResult, 1),
		stdout:  newBuf(),
		stderr:  newBuf(),
	}
	job.globCtx.InheritEnv(os.Environ()...)
	job.globCtx.Stdout = &job.stdout
	job.globCtx.Stderr = &job.stdout
	job.globCtx.Dir = workDir
	return &job
}

func (j *Job) Ctx() stdctx.Context { return j.ctx }
func (j *Job) Err() error {
	j.RLock()
	defer j.RUnlock()
	return j.err
}

// Finish() cleans up all resources associated with managing a job
func (j *Job) Finish(err error) {
	j.Lock()
	defer j.Unlock()
	if j.finished {
		return
	}
	// really cancel context here???
	j.err = err
	j.finished = true
	j.finishTime = time.Now()

	close(j.results)
	j.stdout.Close()
	j.stderr.Close()

	j.cancel()
}

func (j *Job) Finished() (time.Time, bool, error) {
	j.RLock()
	defer j.RUnlock()
	return j.finishTime, j.finished, j.err
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
