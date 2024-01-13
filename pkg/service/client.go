package service

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"

	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StepRunnerClient struct {
	proto.StepRunnerClient
	conn   *grpc.ClientConn
	client proto.StepRunnerClient
	// number of bytes already read from stdout/err.
	readStdout, readStderr int32
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

func (c *StepRunnerClient) RunAndFollow(ctx context.Context, jobID, workDir string, steps []*proto.Step, out *Output) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := c.client.Run(ctx, &proto.RunRequest{
		Id:      jobID,
		Steps:   steps,
		WorkDir: workDir,
	})
	if err != nil {
		return err
	}
	defer c.conn.Close()
	//nolint:errcheck
	defer c.client.Cancel(ctx, &proto.CancelRequest{Id: jobID})

	wg := sync.WaitGroup{}
	wg.Add(2)
	var stepResultErr, ioStreamError error
	go func() {
		defer wg.Done()
		stepResultErr = c.startFollow(ctx, jobID, out.StepResult)
		if stepResultErr != nil {
			cancel() // force startFollowIO to exit
		}
	}()
	go func() {
		defer wg.Done()
		ioStreamError = c.startFollowIO(ctx, jobID, out.Stdout, out.Stderr)
		if ioStreamError != nil {
			cancel() // force startFollow to exit
		}
	}()

	wg.Wait()
	return errors.Join(stepResultErr, ioStreamError)
}

func (c *StepRunnerClient) startFollow(ctx context.Context, jobID string, resultC chan<- *proto.StepResult) error {
	stepResultStream, err := c.client.Follow(ctx, &proto.FollowRequest{Id: jobID})
	if err != nil {
		return err
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		res, err := stepResultStream.Recv()
		if err == io.EOF {
			log.Println("step-result stream done")
			return nil
		}
		// TODO: reconnect here if the error was io.ErrClosedPipe or io.ErrUnexpectedEOF
		if err != nil {
			log.Println(err.Error())
			return err
		}

		resultC <- res.GetResult()
	}
}

func (c *StepRunnerClient) startFollowIO(ctx context.Context, jobID string, stdoutC, stderrC chan<- []byte) error {
	ioStream, err := c.client.FollowIO(ctx, &proto.FollowIORequest{Id: jobID})
	if err != nil {
		return err
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		res, err := ioStream.Recv()
		if err == io.EOF {
			log.Println("io-stream done")
			return nil
		}
		// TODO: reconnect here if the error was io.ErrClosedPipe or io.ErrUnexpectedEOF
		if err != nil {
			log.Println(err.Error())
			return err
		}

		switch res.StreamType {
		case proto.FollowIOResponse_stdout:
			c.readStdout += int32(len(res.Stream))
			stdoutC <- res.Stream
		case proto.FollowIOResponse_stderr:
			c.readStderr += int32(len(res.Stream))
			stderrC <- res.Stream
		}
	}
}
