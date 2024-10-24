// package basic implements a low-level client for the step-runner gRPC service with a (more or less) 1:1 mapping to
// the raw gRPC API. It abstracts all proto types except for StepResult. Use this API if you want full control over
// every stage of running, following and closing a job.
package basic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"google.golang.org/grpc"

	"gitlab.com/gitlab-org/step-runner/pkg/api/client"
	"gitlab.com/gitlab-org/step-runner/proto"
)

type (
	StepResultWriter interface {
		Write(*proto.StepResult) error
	}

	StepRunnerClient struct {
		client proto.StepRunnerClient
	}
)

func toProto(r *client.RunRequest) *proto.RunRequest {
	rr := proto.RunRequest{
		Id: r.Id,
		Context: &proto.Context{
			WorkDir: r.WorkDir,
			Env:     r.Env,
		},
		FunctionOneof: &proto.RunRequest_Steps{
			Steps: r.Steps,
		},
	}

	if len(r.Variables) != 0 {
		for _, v := range r.Variables {
			rr.Context.Job = append(rr.Context.Job, &proto.Variable{
				Key:    v.Key,
				Value:  v.Value,
				File:   v.File,
				Masked: v.Masked,
			})
		}
	}

	return &rr
}

func fromProto(statuses []*proto.Status) []client.Status {
	fromProto := func(st *proto.Status) client.Status {
		res := client.Status{
			Id:        st.Id,
			Message:   st.Message,
			State:     client.State(st.Status),
			StartTime: st.StartTime.AsTime(),
		}
		if st.EndTime != nil {
			res.EndTime = st.EndTime.AsTime()
		}
		return res
	}

	result := make([]client.Status, 0, len(statuses))
	for _, j := range statuses {
		result = append(result, fromProto(j))
	}
	return result
}

func New(conn *grpc.ClientConn) *StepRunnerClient {
	return &StepRunnerClient{
		client: proto.NewStepRunnerClient(conn),
	}
}

// Run initiates the job defined in runRequest on the connected step-runner service.
func (c *StepRunnerClient) Run(ctx context.Context, runRequest *client.RunRequest) error {
	// TODO: compile steps here when we separate step compilation and execution...
	if _, err := c.client.Run(ctx, toProto(runRequest)); err != nil {
		return fmt.Errorf("running job request: %w", err)
	}
	return nil
}

// Close cancelled (if running) the specified job-id, and frees all resources associated with the job.
func (c *StepRunnerClient) Close(ctx context.Context, jobID string) error {
	if _, err := c.client.Close(ctx, &proto.CloseRequest{Id: jobID}); err != nil {
		return fmt.Errorf("closing job: %w", err)
	}
	return nil
}

// Status returns the Status of the specified job.
func (c *StepRunnerClient) Status(ctx context.Context, jobID string) (client.Status, error) {
	job, err := c.client.Status(context.Background(), &proto.StatusRequest{Id: jobID})
	if err != nil {
		return client.Status{}, fmt.Errorf("getting status for job %q: %w", jobID, err)
	}
	return fromProto(job.Jobs)[0], nil
}

// ListJobs returns the Status for all jobs running on the connected step-runner service.
func (c *StepRunnerClient) ListJobs(ctx context.Context) ([]client.Status, error) {
	jobs, err := c.client.Status(context.Background(), &proto.StatusRequest{})
	if err != nil {
		return nil, fmt.Errorf("listing jobs: %w", err)
	}
	return fromProto(jobs.Jobs), nil
}

// FollowSteps streams StepResults emitted by the specified job to the specified StepResultWriter.
func (c *StepRunnerClient) FollowSteps(ctx context.Context, jobID string, offset int64, writer StepResultWriter) (int64, error) {
	if writer == nil {
		return -1, errors.New("nil StepResultWriter")
	}

	// TODO: add offset to FollowSteps
	stepResultStream, err := c.client.FollowSteps(ctx, &proto.FollowStepsRequest{Id: jobID})
	if err != nil {
		return -1, fmt.Errorf("following step-results: %w", err)
	}

	written := offset
	for {
		if ctx.Err() != nil {
			return written, ctx.Err()
		}

		res, err := stepResultStream.Recv()
		if err == io.EOF {
			log.Println("step-result stream done")
			return written, nil
		}
		if err != nil {
			return written, fmt.Errorf("following step-results: %w", err)
		}

		err = writer.Write(res.Result)
		written++
		if err != nil {
			return written, fmt.Errorf("writing to step-result sink: %w", err)
		}
	}
}

// FollowLogs streams logs emitted by the specified job to the specified io.Writer.
func (c *StepRunnerClient) FollowLogs(ctx context.Context, jobID string, offset int64, writer io.Writer) (int64, error) {
	if writer == nil {
		return -1, errors.New("nil io.Writer")
	}

	ioStream, err := c.client.FollowLogs(ctx, &proto.FollowLogsRequest{Id: jobID, Offset: offset})
	if err != nil {
		return -1, fmt.Errorf("following logs: %w", err)
	}

	written := offset
	for {
		if ctx.Err() != nil {
			return -1, ctx.Err()
		}
		res, err := ioStream.Recv()
		if err == io.EOF {
			log.Println("logs stream done")
			return written, nil
		}
		if err != nil {
			return written, fmt.Errorf("following logs: %w", err)
		}

		n, err := writer.Write(res.Data)
		written += int64(n)
		if err != nil {
			return written, fmt.Errorf("writing to log sink: %w", err)
		}
	}
}

// RunUp streams RunRequests from the server and responds with FollowStepsResponses.
func (c *StepRunnerClient) RunUp(ctx context.Context, jobID string) (proto.StepRunner_RunUpClient, error) {
	// I don't see any point is wrapping this.
	return c.client.RunUp(ctx)
}
