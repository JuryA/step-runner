package service

import (
	"context"

	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StepRunnerClient struct {
	proto.StepRunnerClient
	conn   *grpc.ClientConn
	client proto.StepRunnerClient
}

type StepResultStream interface {
	Recv() (*proto.FollowResponse, error)
}

func NewClient(serverAddr string) (*StepRunnerClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := proto.NewStepRunnerClient(conn)

	return &StepRunnerClient{
		conn:   conn,
		client: client,
	}, nil
}

// func (c *StepRunnerClient) RunCIJob(ctx context.Context, jobID, script string) error {
// 	_, err := c.client.Run(ctx, &proto.RunRequest{
// 		Id: jobID,
// 		JobOneof: &proto.RunRequest_CiJob{
// 			CiJob: script,
// 		},
// 	})
// 	return err
// }

func (c *StepRunnerClient) RunStep(ctx context.Context, jobID string, steps []*proto.Step) error {
	_, err := c.client.Run(ctx, &proto.RunRequest{
		Id:    jobID,
		Steps: steps,
	})
	return err
}

// maybe could return a channel of FollowResponse or StepResult here instead, but it makes error handling more
// complicated
func (c *StepRunnerClient) Follow(ctx context.Context, jobID string) (StepResultStream, error) {
	return c.client.Follow(ctx, &proto.FollowRequest{
		Id: jobID,
	})
}

func (c *StepRunnerClient) Cancel(ctx context.Context, jobID string) error {
	_, err := c.client.Cancel(ctx, &proto.CancelRequest{Id: jobID})
	return err
}

func (c *StepRunnerClient) Close() error {
	return c.conn.Close()
}
