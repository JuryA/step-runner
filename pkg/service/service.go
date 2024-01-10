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

type Request struct {
	id            string
	ctx           *context.Global        // to capture procs stdout/err
	ctx2          stdctx.Context         // the context used in Run. choose a better name
	results       chan *proto.StepResult // to stream the results so Follow can get at them
	err           error                  // capture error when running steps
	cancel        func()                 // cancel the context
	chanCloseOnce sync.Once
	stdout        bytes.Buffer
}

func (r *Request) closeChan() {
	r.chanCloseOnce.Do(func() {
		close(r.results)
	})
}

type StepRunnerServer struct {
	proto.StepRunnerServer
	cache    cache.Cache
	requests map[string]*Request // probably synchronize this
}

func NewServer() (*StepRunnerServer, error) {
	c, err := cache.New()
	if err != nil {
		return nil, fmt.Errorf("creating cache: %w", err)
	}
	return &StepRunnerServer{
		cache:    c,
		requests: map[string]*Request{},
	}, nil
}

func (s *StepRunnerServer) Run(ctx stdctx.Context, request *proto.RunRequest) (*proto.RunResponse, error) {
	execution, err := runner.New(s.cache)
	if err != nil {
		return nil, fmt.Errorf("creating execution: %w", err)
	}

	ctx2, cancel := stdctx.WithCancel(stdctx.Background())
	req := Request{
		id:      request.Id,
		ctx:     context.NewGlobal(),
		results: make(chan *proto.StepResult, 1),
		ctx2:    ctx2,
		cancel:  cancel,
	}
	req.ctx.InheritEnv(os.Environ()...)
	req.ctx.Stdout = &req.stdout
	req.ctx.Stderr = &req.stdout
	req.ctx.Dir = request.WorkDir

	s.requests[request.Id] = &req

	go func() {
		defer req.closeChan()
		// this needs to change to truly stream results back to the caller.
		result, err := execution.Run(ctx2, getOrMakeStep(request), &runner.Params{}, req.ctx)
		if err != nil {
			log.Printf("an error occurred executing the job: %s", err)
			req.err = fmt.Errorf("execution failed: %w", err)
			// } else if req.ctx2.Err() != nil {
		} else {
			req.results <- result
			writeStepResult(req.ctx.Dir, result)
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

// NOTE: Errors returned from this function will only appear on the client side on the first call to
// StepRunner_FollowClient.Recv(), NOT in the error returned from calling this API directly.
func (s *StepRunnerServer) Follow(request *proto.FollowRequest, writer proto.StepRunner_FollowServer) error {
	log.Println("request to follow job", request.Id, s.requests)

	req, ok := s.requests[request.Id]
	if !ok {
		log.Printf("follow: no such job %s", request.Id)
		return fmt.Errorf("follow: no such job %s", request.Id)
	}

	for res := range req.results {
		// the context was cancelled. exit.
		if req.ctx2.Err() != nil {
			defer s.cancel(req)
			log.Printf("follow: error reading results for job %s: %v", request.Id, req.ctx2.Err().Error())
			return fmt.Errorf("error reading results for job %s: %w", request.Id, req.ctx2.Err())
		}

		resp := proto.FollowResponse{
			Result: res,
			Output: req.stdout.Bytes(),
		}

		if err := writer.Send(&resp); err != nil {
			log.Printf("follow: send error for job %s: %s", request.Id, err.Error())
			return fmt.Errorf("send error for job %s: %w", request.Id, err)
		}
	}

	if req.err != nil {
		// do we really want to remove the job from the list of jobs here? doing so precludes being able to call follow
		// again.
		s.cancel(req)
		return req.err
	}

	return nil
}

func (s *StepRunnerServer) Cancel(_ stdctx.Context, request *proto.CancelRequest) (*proto.CancelResponse, error) {
	req, ok := s.requests[request.Id]
	if !ok {
		log.Printf("cancel: no such job %s", request.Id)
		return nil, fmt.Errorf("cancel: no such job %s", request.Id)
	}
	s.cancel(req)
	return &proto.CancelResponse{}, nil
}

func (s *StepRunnerServer) cancel(req *Request) {
	req.cancel()
	req.closeChan()
	delete(s.requests, req.id)
}
