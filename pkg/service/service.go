package service

import (
	"bytes"
	stdctx "context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sync"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/context"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type Buf struct {
	buffer []byte
	c      chan []byte
	lock   sync.RWMutex
	once   sync.Once
}

func newBuf() Buf { return Buf{c: make(chan []byte)} }

func (b *Buf) Write(p []byte) (int, error) {
	b.lock.Lock()
	b.buffer = append(b.buffer, p...)
	b.lock.Unlock()

	// TODO: this won't always work. if no client has called FollowIO this will block, and if more that one client has
	// called FollowIO, which once receives each write to the channel is non-deterministic. We need one channel per
	// client that called FollowIO (including 0).
	b.c <- p
	return len(p), nil
}

func (b *Buf) Read(offset int32) []byte {
	b.lock.RLock()
	defer b.lock.RUnlock()

	// if the offset is out of range, just return a nil slice.
	if int(offset) >= len(b.buffer) {
		return nil
	}

	return b.buffer[offset:]
}

func (b *Buf) Close() {
	b.once.Do(func() {
		close(b.c)
	})
}

type Job struct {
	id            string
	ctx           *context.Global        // to capture procs stdout/err
	ctx2          stdctx.Context         // the context used in Run. choose a better name
	results       chan *proto.StepResult // to stream the results so Follow can get at them
	err           error                  // capture error when running steps
	cancel        func()                 // cancel the context
	chanCloseOnce sync.Once
	stdout        bytes.Buffer
}

func (j *Job) closeChan() {
	j.chanCloseOnce.Do(func() {
		close(j.results)
	})
}

type StepRunnerServer struct {
	proto.StepRunnerServer
	cache cache.Cache
	jobs  map[string]*Job // probably synchronize this
}

func NewServer() (*StepRunnerServer, error) {
	c, err := cache.New()
	if err != nil {
		return nil, fmt.Errorf("creating cache: %w", err)
	}
	return &StepRunnerServer{
		cache: c,
		jobs:  map[string]*Job{},
	}, nil
}

func (s *StepRunnerServer) Run(ctx stdctx.Context, request *proto.RunRequest) (*proto.RunResponse, error) {
	execution, err := runner.New(s.cache)
	if err != nil {
		return nil, fmt.Errorf("creating execution: %w", err)
	}

	ctx2, cancel := stdctx.WithCancel(stdctx.Background())
	job := Job{
		id:      request.Id,
		ctx:     context.NewGlobal(),
		results: make(chan *proto.StepResult, 1),
		ctx2:    ctx2,
		cancel:  cancel,
	}
	job.ctx.InheritEnv(os.Environ()...)
	job.ctx.Stdout = &job.stdout
	job.ctx.Stderr = &job.stdout
	job.ctx.Dir = request.WorkDir

	s.jobs[request.Id] = &job

	go func() {
		defer job.closeChan()
		// this needs to change to truly stream results back to the caller.
		result, err := execution.Run(ctx2, getOrMakeStep(request), &runner.Params{}, job.ctx)
		if err != nil {
			log.Printf("an error occurred executing the job: %s", err)
			job.err = fmt.Errorf("execution failed: %w", err)
			// } else if req.ctx2.Err() != nil {
		} else {
			job.results <- result
			writeStepResult(job.globCtx.Dir, result)
		}
	}()

	return &proto.RunResponse{}, nil
}

func writeStepResult(destDir string, result *proto.StepResult) error {
	bytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling step results: %w", err)
	}
	outputFile := path.Join(destDir, "step-results.json")
	err = os.WriteFile(outputFile, bytes, 0640)
	if err != nil {
		return fmt.Errorf("writing step results to %v: %w", outputFile, err)
	}
	log.Printf("trace written to %v\n", outputFile)
	return nil
}

func getOrMakeStep(request *proto.RunRequest) *proto.StepDefinition {
	return &proto.StepDefinition{
		Spec: &proto.Spec{
			Spec: &proto.Spec_Content{
				Inputs:  map[string]*proto.Spec_Content_Input{},
				Outputs: map[string]*proto.Spec_Content_Output{},
			},
		},
		Definition: &proto.Definition{
			Type:  proto.DefinitionType_steps,
			Steps: request.Steps,
		},
	}
}

func (s *StepRunnerServer) getJob(jid string) (*Job, error) {
	req, ok := s.jobs[jid]
	if !ok {
		log.Printf("follow: no such job %s", jid)
		return nil, fmt.Errorf("follow: no such job %s", jid)
	}
	return req, nil
}

// NOTE: Errors returned from this function will only appear on the client side on the first call to
// StepRunner_FollowClient.Recv(), NOT in the error returned from calling this API directly.
func (s *StepRunnerServer) Follow(request *proto.FollowRequest, writer proto.StepRunner_FollowServer) error {
	log.Println("request to follow job", request.Id, s.jobs)

	job, err := s.getJob(request.Id)
	if err != nil {
		return err
	}

stop:
	for {
		select {
		case <-job.ctx2.Done():
			// TODO: maybe just break here and handle the error below?
			// context was cancelled
			defer s.cancel(job)
			return fmt.Errorf("error executing steps for job %s : %w", request.Id, job.ctx2.Err())
		case res, ok := <-job.results:
			if !ok {
				// channel was closed
				break stop
			}

			if err := writer.Send(&proto.FollowResponse{Result: res}); err != nil {
				log.Printf("follow: send error for job %s: %s", request.Id, err.Error())
				return fmt.Errorf("send error for job %s: %w", request.Id, err)
			}
		}
	}

	if job.err != nil {
		// do we really want to remove the job from the list of jobs here? doing so precludes being able to call follow
		// again.
		s.cancel(job)
		return job.err
	}

	return nil
}

func (s *StepRunnerServer) Cancel(_ stdctx.Context, request *proto.CancelRequest) (*proto.CancelResponse, error) {
	log.Println("request to cancel job", request.Id, s.jobs)

	job, err := s.getJob(request.Id)
	if err != nil {
		return &proto.CancelResponse{}, nil
	}
	s.cancel(job)
	return &proto.CancelResponse{}, nil
}

func (s *StepRunnerServer) cancel(job *Job) {
	job.cancel()
	job.closeChan()
	delete(s.jobs, job.id)
}
