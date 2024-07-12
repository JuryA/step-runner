// package extended implements a well-behaved, higher-level client for the step-runner gRPC service. The primary entry
// point, RunAndFollow(), will initiate a steps job, Follow*() the step-results and logs to to completion, and on
// completion get the job's Status() and Close() the job, releasing all resources.
//
// While it does not do so currently, this client will in the future automatically reconnect and re-initiate
// Following on connection errors using the specified DialFunc.
//
// Callers can use the FollowOutput type to receive streaming log and step-results.
//
// Note that if it is cancelled or times out, the context passed to RunAndFollow will cancel the client AND also call
// Close() on the job, effectively cancelling it on the server side too.
package extended

import (
	"context"
	"errors"
	"fmt"
	"io"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"gitlab.com/gitlab-org/step-runner/pkg/api/client"
	"gitlab.com/gitlab-org/step-runner/pkg/api/client/basic"
)

type (
	StepResultWriteCloser interface {
		basic.StepResultWriter
		Close() error
	}

	FollowOutput struct {
		Logs        io.WriteCloser
		StepResults StepResultWriteCloser

		readLogs, readStepResults int64
	}

	Dialer interface {
		Dial() (*grpc.ClientConn, error)
	}

	StepRunnerClient struct {
		*basic.StepRunnerClient
		conn   *grpc.ClientConn
		dialer Dialer
	}
)

func New(dialer Dialer) (*StepRunnerClient, error) {
	conn, err := dialer.Dial()
	if err != nil {
		return nil, fmt.Errorf("dialing: %w", err)
	}

	return &StepRunnerClient{
		StepRunnerClient: basic.New(conn),
		conn:             conn,
		// TODO: this will change when we add reconnection
		dialer: dialer,
	}, nil
}

// RunAndFollow manages the complete lifecycle of a step run request, including initiating the run request, following
// output streams (as configured by FollowOutput), querying the final job status, and finally Closing the job.
//
// Note that if ctx is cancelled or times out, Close will be called on the job, effectively cancelling it on the server
// too.
func (c *StepRunnerClient) RunAndFollow(ctx context.Context, runRequest *client.RunRequest, out *FollowOutput) (client.Status, error) {
	err := c.Run(ctx, runRequest)
	if err != nil {
		return client.Status{}, err
	}

	// TODO: Do we always want to call Close here? If we don't, we may need to distinguish between recoverable and
	// non-recoverable errors. Do we want to capture and return Close errors too? Do we want to call Close() on context
	// cancellation/timeout (probably yes)?
	//nolint:errcheck
	defer c.Close(context.Background(), runRequest.Id)

	return c.Follow(ctx, runRequest.Id, out)
}

// Follow follows log and step-result streams as configured by FollowOutput, and return the job's final status. If nil
// is specified for either sink, that stream will not be followed. At least one sink must be specified.
func (c *StepRunnerClient) Follow(ctx context.Context, jobID string, out *FollowOutput) (client.Status, error) {
	if out.Logs == nil && out.StepResults == nil {
		return client.Status{}, errors.New("at least one stream sink must be specified")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg := errgroup.Group{}

	if out.StepResults != nil {
		eg.Go(func() error {
			// TODO: do we want to close this here?
			defer out.StepResults.Close()
			// TODO: add reconnection
			n, err := c.FollowSteps(ctx, jobID, out.readStepResults, out.StepResults)
			out.readStepResults += n
			if err != nil {
				cancel() // force followLogs to exit
			}
			return err
		})
	}

	if out.Logs != nil {
		eg.Go(func() error {
			// TODO: do we want to close this here?
			defer out.Logs.Close()
			// TODO: add reconnection
			n, err := c.FollowLogs(ctx, jobID, out.readLogs, out.Logs)
			out.readLogs += n
			if err != nil {
				cancel() // force followSteps to exit
			}
			return err
		})
	}

	var followErr error
	if err := eg.Wait(); err != nil {
		followErr = fmt.Errorf("following job %q: %w", jobID, err)
	}

	status, statErr := c.Status(context.Background(), jobID)
	if statErr != nil {
		return client.Status{}, errors.Join(followErr, statErr)
	}
	return status, followErr
}

// CloseConn closes the connection to the step-runner service.
func (c *StepRunnerClient) CloseConn() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
