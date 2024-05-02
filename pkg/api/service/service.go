// package service implements the gRPC API declared in ../../../proto/step.proto
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/jobs"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/step"
	"gitlab.com/gitlab-org/step-runner/pkg/syncmap"
	"gitlab.com/gitlab-org/step-runner/pkg/variables"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type StepRunnerService struct {
	proto.StepRunnerServer
	cache cache.Cache

	jobs *syncmap.SyncMap[string, *jobs.Job]
}

func New() (*StepRunnerService, error) {
	c, err := cache.New()
	if err != nil {
		return nil, fmt.Errorf("creating step-runner-service: %w", err)
	}
	return &StepRunnerService{
		cache: c,
		jobs:  syncmap.New[string, *jobs.Job](),
	}, nil
}

// Run parses, prepares, and initiates execution of a RunRequest.
func (s *StepRunnerService) Run(ctx context.Context, request *proto.RunRequest) (*proto.RunResponse, error) {
	execution, err := runner.New(s.cache)
	if err != nil {
		return nil, fmt.Errorf("creating execution: %w", err)
	}

	steps, err := s.loadSteps(request.Steps)
	if err != nil {
		return nil, fmt.Errorf("loading step: %w", err)
	}

	job, err := jobs.New(request)
	if err != nil {
		return nil, fmt.Errorf("initializing request: %w", err)
	}

	jobVars, err := variables.Prepare(request.Job, job.TmpDir)
	if err != nil {
		return nil, fmt.Errorf("preparing environment: %w", err)
	}

	job.GlobCtx.Job = variables.Expand(jobVars)
	if request.Env != nil {
		job.GlobCtx.Env = request.Env
	}

	// last chance to bail...
	if ctx.Err() != nil {
		job.Close()
		return nil, ctx.Err()
	}

	// actually execute the steps request
	s.jobs.Put(request.Id, job)
	go s.run(execution, job, steps)
	return &proto.RunResponse{}, nil
}

func (s *StepRunnerService) loadSteps(stepsStr string) (*proto.StepDefinition, error) {
	stepDef, err := step.ReadSteps(stepsStr, "")
	if err != nil {
		return nil, fmt.Errorf("reading steps %q: %w", stepsStr, err)
	}

	steps, err := step.CompileSteps(stepDef)
	if err != nil {
		return nil, fmt.Errorf("compiling steps: %w", err)
	}

	return steps, nil
}

// run actually starts execution of the steps request and captures the result. It is intended to be run in a goroutine.
func (s *StepRunnerService) run(execution *runner.Execution, job *jobs.Job, steps *proto.StepDefinition) {
	// TODO: Add streaming of step-results as they are produced.
	result, err := execution.Run(job.Ctx, job.GlobCtx, &runner.Params{}, steps)
	job.Finish(result, err)
	if err != nil {
		// TODO: better logging
		log.Printf("an error occurred executing the job: %s", err)
	}
}

// TODO: this is very much a temporary/throwaway implementation until we implement step-result streaming and job timeout.
func (s *StepRunnerService) FollowSteps(request *proto.FollowStepsRequest, writer proto.StepRunner_FollowStepsServer) error {
	job, ok := s.jobs.Get(request.Id)
	if !ok {
		return fmt.Errorf("no job with id %q", request.Id)
	}

	// TODO: this is temporary until we implement step-result streaming and job timeout.
	err := waitFor(job.Finished, time.Millisecond*250, time.Hour)
	if err != nil {
		return fmt.Errorf("job times out: %w", err)
	}

	result, err := job.Result()
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	return writer.Send(&proto.FollowStepsResponse{Result: result})
}

// TODO: this is temporary until we implement step-result streaming
func waitFor(f func() bool, pollInterval, maxWait time.Duration) error {
	start := time.Now()

	for time.Since(start) < maxWait {
		if f() {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timed out waiting for operation")
}
