// package service implements the gRPC API declared in ../../../proto/step.proto
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/jobs"
	"gitlab.com/gitlab-org/step-runner/pkg/api/internal/variables"
	"gitlab.com/gitlab-org/step-runner/pkg/internal/syncmap"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
)

type errBadJobID struct{ id string }

func (e *errBadJobID) Error() string { return fmt.Sprintf("no job with id %q", e.id) }

type StepRunnerService struct {
	proto.StepRunnerServer
	cache runner.Cache

	jobs *syncmap.SyncMap[string, *jobs.Job]
}

func New(stepCache runner.Cache) *StepRunnerService {
	return &StepRunnerService{
		cache: stepCache,
		jobs:  syncmap.New[string, *jobs.Job](),
	}
}

// Run parses, prepares, and initiates execution of a RunRequest.
func (s *StepRunnerService) Run(ctx context.Context, request *proto.RunRequest) (response *proto.RunResponse, err error) {
	specDef, err := s.loadSteps(request.Steps)
	if err != nil {
		return nil, fmt.Errorf("loading step: %w", err)
	}

	specDef.Dir = request.WorkDir
	if request.Job != nil && request.Job.BuildDir != "" {
		specDef.Dir = request.Job.BuildDir
	}

	job, err := jobs.New(request)
	if err != nil {
		return nil, fmt.Errorf("initializing request: %w", err)
	}

	defer func() {
		if err != nil {
			job.Close()
		}
	}()

	jobVars, err := variables.Prepare(request.Job, job.TmpDir)
	if err != nil {
		return nil, fmt.Errorf("preparing environment: %w", err)
	}

	job.GlobCtx.Job = variables.Expand(jobVars)
	if request.Env != nil {
		job.GlobCtx.Env = request.Env
	}

	step, err := runner.NewParser(job.GlobCtx, s.cache).Parse(specDef, &runner.Params{}, runner.StepDefinedInGitLabJob)
	if err != nil {
		return nil, fmt.Errorf("failed to start step runner service: %w", err)
	}

	params := &runner.Params{}
	env := job.GlobCtx.NewEnvMergedFrom(params.Env)
	inputs := params.NewInputsWithDefault(specDef.Spec.Spec.Inputs)
	stepsCtx := runner.NewStepsContext(job.GlobCtx, specDef.Dir, inputs, env)

	// last chance to bail...
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// actually execute the steps request
	s.jobs.Put(request.Id, job)
	go s.run(job, stepsCtx, step, specDef)
	return &proto.RunResponse{}, nil
}

func (s *StepRunnerService) loadSteps(stepsStr string) (*proto.SpecDefinition, error) {
	spec, step, err := schema.ReadSteps(stepsStr)
	if err != nil {
		return nil, fmt.Errorf("reading steps %q: %w", stepsStr, err)
	}
	protoSpec, err := spec.Compile()
	if err != nil {
		return nil, fmt.Errorf("compiling steps: %w", err)
	}
	protoDef, err := step.Compile()
	if err != nil {
		return nil, fmt.Errorf("compiling steps: %w", err)
	}
	protoStepDef := &proto.SpecDefinition{
		Spec:       protoSpec,
		Definition: protoDef,
	}
	return protoStepDef, nil
}

// run actually starts execution of the steps request and captures the result. It is intended to be run in a goroutine.
func (s *StepRunnerService) run(job *jobs.Job, stepsCtx *runner.StepsContext, step runner.Step, specDef *proto.SpecDefinition) {
	// TODO: Add streaming of step-results as they are produced.
	result, err := step.Run(job.Ctx, stepsCtx, specDef)
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
		return &errBadJobID{id: request.Id}
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

func (s *StepRunnerService) Close(ctx context.Context, request *proto.CloseRequest) (*proto.CloseResponse, error) {
	job, ok := s.jobs.Get(request.Id)
	if !ok {
		return nil, &errBadJobID{id: request.Id}
	}

	job.Close()
	s.jobs.Remove(request.Id)

	return &proto.CloseResponse{}, nil
}

// toIOWriter can be used to "cast" a func([]byte)(int, error) to an io.Writer.
type toIOWriter func([]byte) (int, error)

func (w toIOWriter) Write(p []byte) (int, error) { return w(p) }

func (s *StepRunnerService) FollowLogs(request *proto.FollowLogsRequest, writer proto.StepRunner_FollowLogsServer) error {
	job, ok := s.jobs.Get(request.Id)
	if !ok {
		return &errBadJobID{id: request.Id}
	}

	return job.FollowLogs(writer.Context(), request.Offset, toIOWriter(func(p []byte) (int, error) {
		err := writer.Send(&proto.FollowLogsResponse{Data: p})
		if err != nil {
			return 0, err
		}

		return len(p), nil
	}))
}

func (s *StepRunnerService) Status(ctx context.Context, request *proto.StatusRequest) (*proto.StatusResponse, error) {
	stats := []*proto.Status{}
	if request.Id != "" {
		job, ok := s.jobs.Get(request.Id)
		if !ok {
			return nil, &errBadJobID{id: request.Id}
		}
		stats = append(stats, job.Status())
	} else {
		s.jobs.ForEach(func(_ string, j *jobs.Job) {
			stats = append(stats, j.Status())
		})
	}

	return &proto.StatusResponse{Jobs: stats}, nil
}
