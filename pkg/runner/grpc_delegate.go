package runner

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"gitlab.com/gitlab-org/step-runner/pkg/api/client/basic"
	"gitlab.com/gitlab-org/step-runner/pkg/api/client/extended"
	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCOutputer struct {
	id             string
	runUpClient    proto.StepRunner_RunUpClient
	extendedClient *extended.StepRunnerClient
	stopCh         chan struct{}
	ctx            context.Context
	stepResultCh   chan *proto.StepResult
	stepResult     *proto.StepResult
}

var _ basic.StepResultWriter = (*GRPCOutputer)(nil)

type alreadyDialed struct {
	conn *grpc.ClientConn
}

func (a *alreadyDialed) Dial() (*grpc.ClientConn, error) {
	return a.conn, nil
}

func NewFromDelegationFile(id string, filename string) (*GRPCOutputer, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file %v: %w", filename, err)
	}
	delegation := &proto.Delegation{}
	if err := protojson.Unmarshal(data, delegation); err != nil {
		return nil, fmt.Errorf("reading output_file as grpc delegate: %w", err)
	}
	address := "unix:///" + delegation.SocketFile
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	stepRunnerClient := proto.NewStepRunnerClient(conn)
	// Submit run request to delegation endpoint
	stepResultCh := make(chan *proto.StepResult)
	go func() {
		fmt.Printf("calling run\n")
		res, err := stepRunnerClient.Run(context.Background(), &proto.RunRequest{
			Id:      id,
			Context: delegation.Continuation.Context,
			FunctionOneof: &proto.RunRequest_Function{
				Function: delegation.Continuation.Function,
			},
		})
		fmt.Printf("finished run\n")
		if err != nil {
			panic(err)
		}
		stepResultCh <- res.Result
		fmt.Printf("delivered result\n")
	}()

	// We subscribe to run up requests for the specific job_id the delegate gave us
	ctx := metadata.AppendToOutgoingContext(context.Background(), "id", id)
	runUpClient, err := stepRunnerClient.RunUp(ctx)
	if err != nil {
		return nil, err
	}

	conn, err = grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	extendedClient, err := extended.New(&alreadyDialed{conn})
	if err != nil {
		return nil, err
	}
	return &GRPCOutputer{
		id:             id,
		runUpClient:    runUpClient,
		extendedClient: extendedClient,
		stopCh:         make(chan struct{}),
		ctx:            ctx,
	}, nil
}

func (o *GRPCOutputer) ServiceRunUp() {
	fmt.Printf("begin service run up\n")
	defer fmt.Printf("end service run up\n")
	for {
		select {
		case <-o.stopCh:
			return
		default:
			req, err := o.runUpClient.Recv()
			if err == io.EOF {
				// There has to be a better way of doing this.
				time.Sleep(time.Second)
				continue
			}
			if err != nil {
				// TODO errors should be send back down for the delegate to deal with
				fmt.Printf("error: %v\n", err)
				time.Sleep(time.Second)
				continue
			}
			log.Printf("got run up request\n")

			// Create global and steps contexts from the request
			env := NewEnvironment(req.Context.GetEnv())
			globalCtx, err := NewGlobalContext(env)
			// Run request should include inputs
			inputs := map[string]*structpb.Value{}
			stepsCtx := NewStepsContext(globalCtx, req.Context.WorkDir, inputs, req.Context.GetEnv())

			specDef := req.GetFunction()
			if err != nil {
				panic(err)
			}
			step, err := NewParser(globalCtx, CacheSingleton).Parse(specDef, &Params{}, StepDefinedInGitLabJob)
			if err != nil {
				panic(err)
			}

			// Run this request concurrently with other run up requests
			go func(id string, stepsCtx *StepsContext) {
				stepResult, err := step.Run(o.ctx, stepsCtx, specDef)
				if err != nil {

				}
				res := &proto.RunResponse{
					Id:     id,
					Result: stepResult,
				}
				o.runUpClient.Send(res)
			}(req.Id, stepsCtx)
		}
	}
}

// There's probably a better way to do this.
func (o *GRPCOutputer) Write(res *proto.StepResult) error {
	fmt.Printf("writing. why do we do this anyway?\n")
	o.stepResult = res
	res.Delegation = o.stepResult.Delegation
	res.Env = o.stepResult.Env
	res.ExecResult = o.stepResult.ExecResult
	res.Exports = o.stepResult.Exports
	res.Outputs = o.stepResult.Outputs
	res.SpecDefinition = o.stepResult.SpecDefinition
	res.Status = o.stepResult.Status
	res.Step = o.stepResult.Step
	res.SubStepResults = o.stepResult.SubStepResults
	return nil
}

func (o *GRPCOutputer) Outputs() (map[string]*structpb.Value, *proto.StepResult, error) {
	defer func() {
		// Stop servicing run up requests after we get our output
		close(o.stopCh)
	}()
	fmt.Printf("waiting on outputs\n")
	o.stepResult = <-o.stepResultCh
	fmt.Printf("got outputs\n")
	return o.stepResult.Outputs, o.stepResult, nil
}
