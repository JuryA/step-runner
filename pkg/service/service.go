package service

import (
	stdctx "context"
	"fmt"
	"log"

	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type StepRunnerServer struct {
	proto.StepRunnerServer
	cache cache.Cache
	jobs  *ConcurrentMap[string, *Job]
}

func NewServer() (*StepRunnerServer, error) {
	c, err := cache.New()
	if err != nil {
		return nil, fmt.Errorf("creating cache: %w", err)
	}
	return &StepRunnerServer{
		cache: c,
		jobs:  New[string, *Job](),
	}, nil
}

func (s *StepRunnerServer) Run(ctx stdctx.Context, request *proto.RunRequest) (*proto.RunResponse, error) {
	log.Println("request to run job", request.Id, s.jobs.Keys())

	_, ok := s.jobs.Get(request.Id)
	if ok {
		log.Printf("job %s already exists", request.Id)
		return &proto.RunResponse{}, nil
	}

	if request.Type != proto.RunRequest_step {
		return nil, fmt.Errorf("unsupported script-type %q",
			proto.RunRequest_StepType_name[int32(request.Type)])
	}

	execution, err := runner.New(s.cache)
	if err != nil {
		return nil, fmt.Errorf("creating execution: %w", err)
	}

	job := NewJob(request.Id, request.WorkDir)

	s.jobs.Put(request.Id, job)

	go func() {
		defer job.Finish(nil)
		// this needs to change to truly stream results back to the caller.
		result, err := execution.Run(job.Ctx(), getOrMakeStep(request), &runner.Params{}, job.globCtx)
		if err != nil {
			log.Printf("an error occurred executing the job: %s", err)
			job.Finish(fmt.Errorf("execution failed: %w", err))
		} else {
			job.results <- result
		}
	}()

	return &proto.RunResponse{}, nil
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
	req, ok := s.jobs.Get(jid)
	if !ok {
		log.Printf("no such job %s", jid)
		return nil, fmt.Errorf("no such job %s", jid)
	}
	return req, nil
}

// NOTE: Errors returned from this function will only appear on the client side on the first call to
// StepRunner_FollowClient.Recv(), NOT in the error returned from calling this API directly.
func (s *StepRunnerServer) Follow(request *proto.FollowRequest, writer proto.StepRunner_FollowServer) error {
	log.Println("request to follow job", request.Id, s.jobs.Keys())

	job, err := s.getJob(request.Id)
	if err != nil {
		return err
	}

stop:
	for {
		select {
		case <-job.Ctx().Done():
			break stop
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

	if job.Err() != nil {
		return job.Err()
	}

	//nolint:staticcheck
	if job.Ctx().Err() != nil {
		// TODO: this will always be true because we canceled the context in Job.Finish()
		// return fmt.Errorf("error executing steps for job %s : %w", request.Id, job.Ctx().Err())
	}

	return nil
}

func (s *StepRunnerServer) FollowIO(request *proto.FollowIORequest, writer proto.StepRunner_FollowIOServer) error {
	log.Println("request to follow IO for job", request.Id, s.jobs.Keys())

	job, err := s.getJob(request.Id)
	if err != nil {
		return err
	}

	// TODO: do we want/need to do this with step results too???
	if err := s.sendOutputSoFar(job, request, writer); err != nil {
		return err
	}

stop:
	for {
		select {
		case <-job.Ctx().Done():
			break stop
		case bytes, ok := <-job.stdout.c:
			if !ok {
				// channel was closed
				break stop
			}
			if err := s.writeStream(bytes, proto.FollowIOResponse_stdout, writer); err != nil {
				return fmt.Errorf("stdout error for job %s: %w", request.Id, err)
			}

		case bytes, ok := <-job.stderr.c:
			if !ok {
				// channel was closed
				break stop
			}
			if err := s.writeStream(bytes, proto.FollowIOResponse_stderr, writer); err != nil {
				return fmt.Errorf("stderr error for job %s: %w", request.Id, err)
			}
		}
	}

	if job.Err() != nil {
		return job.Err()
	}
	//nolint:staticcheck
	if job.Ctx().Err() != nil {
		// TODO: this will always be true because we canceled the context in Job.Finish()
		// return fmt.Errorf("error executing steps for job %s : %w", request.Id, job.Ctx().Err())
	}

	return nil
}

// When a client requests to follow IO, send them the IO buffered so far FROM the requested offset.
func (s *StepRunnerServer) sendOutputSoFar(job *Job, request *proto.FollowIORequest, writer proto.StepRunner_FollowIOServer) error {
	if err := s.writeStream(job.stdout.Read(request.ReadStdout), proto.FollowIOResponse_stdout, writer); err != nil {
		return fmt.Errorf("failed to write stdout stream for job %s: %w", request.Id, err)
	}

	if err := s.writeStream(job.stderr.Read(request.ReadStderr), proto.FollowIOResponse_stderr, writer); err != nil {
		return fmt.Errorf("failed to write stderr stream for job %s: %w", request.Id, err)
	}

	return nil
}

// TODO: chunk up the stream writes so we don't blow past grpc's max message size
func (s *StepRunnerServer) writeStream(stream []byte, streamType proto.FollowIOResponse_StreamType, w proto.StepRunner_FollowIOServer) error {
	if len(stream) == 0 {
		return nil
	}
	resp := proto.FollowIOResponse{
		StreamType: streamType,
		Stream:     stream,
	}

	if err := w.Send(&resp); err != nil {
		return err
	}
	return nil
}

func (s *StepRunnerServer) Cancel(_ stdctx.Context, request *proto.CancelRequest) (*proto.CancelResponse, error) {
	log.Println("request to cancel job", request.Id, s.jobs.Keys())

	job, err := s.getJob(request.Id)
	if err != nil {
		return &proto.CancelResponse{}, nil
	}
	job.Finish(nil)
	s.jobs.Remove(job.id)
	return &proto.CancelResponse{}, nil
}

func (s *StepRunnerServer) List(_ stdctx.Context, request *proto.ListRequest) (*proto.ListResponse, error) {
	result := proto.ListResponse{}
	s.jobs.ForEach(func(_ string, v *Job) {
		result.Jobs = append(result.Jobs, jobToProtoJob(v))
	})
	return &result, nil
}

func jobToProtoJob(j *Job) *proto.Job {
	status := proto.Job_running
	finishedTime, finished, err := j.Finished()

	if finished {
		status = proto.Job_suceeded
		if err != nil {
			status = proto.Job_failed
		}
	}
	pj := proto.Job{
		Id:           j.id,
		Status:       status,
		FinishedTime: timestamppb.New(finishedTime),
	}

	return &pj
}
