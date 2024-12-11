// package service implements the gRPC API declared in ../../../proto/step.proto
package service

import (
	"context"
	"fmt"

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

	env  *runner.Environment
	jobs *syncmap.SyncMap[string, *jobs.Job]
}

func New(stepCache runner.Cache, env *runner.Environment) *StepRunnerService {
	return &StepRunnerService{
		cache: stepCache,
		env:   env,
		jobs:  syncmap.New[string, *jobs.Job](),
	}
}

// Run parses, prepares, and initiates execution of a RunRequest.
func (s *StepRunnerService) Run(ctx context.Context, request *proto.RunRequest) (response *proto.RunResponse, err error) {
	if _, ok := s.jobs.Get(request.Id); ok {
		return &proto.RunResponse{}, nil
	}

	specDef, err := s.loadSteps(request.Steps, request)
	if err != nil {
		return nil, fmt.Errorf("loading step: %w", err)
	}

	job, err := jobs.New(request.Id, specDef.Dir)
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

	globCtx := runner.NewGlobalContext(s.env.AddLexicalScope(request.Env))
	globCtx.Job = jobVars
	globCtx.WorkDir = job.WorkDir
	globCtx.Stdout, globCtx.Stderr = job.Logs()

	params := &runner.Params{}
	step, err := runner.NewParser(globCtx, s.cache).Parse(specDef, params, runner.StepDefinedInGitLabJob)
	if err != nil {
		return nil, fmt.Errorf("failed to start step runner service: %w", err)
	}

	inputs := params.NewInputsWithDefault(specDef.Spec.Spec.Inputs)
	stepsCtx, err := runner.NewStepsContext(globCtx, specDef.Dir, inputs, globCtx.Env)
	if err != nil {
		return nil, err
	}

	// actually execute the steps request
	s.jobs.Put(request.Id, job)
	go job.Run(stepsCtx, step)
	return &proto.RunResponse{}, nil
}

func (s *StepRunnerService) loadSteps(stepsStr string, request *proto.RunRequest) (*proto.SpecDefinition, error) {
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

	protoStepDef.Dir = request.WorkDir
	if request.Job != nil && request.Job.BuildDir != "" {
		protoStepDef.Dir = request.Job.BuildDir
	}
	return protoStepDef, nil
}

func (s *StepRunnerService) Close(ctx context.Context, request *proto.CloseRequest) (*proto.CloseResponse, error) {
	job, ok := s.jobs.Get(request.Id)
	if !ok {
		return &proto.CloseResponse{}, nil
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

func (s *StepRunnerService) Debug(debugServer proto.StepRunner_DebugServer) error {
	for {
		_, err := debugServer.Recv()
		if err != nil {
			return err
		}
		debugServer.Send(&proto.DebugResponse{
			StepView: "(step view)\n",
		})
	}
}
