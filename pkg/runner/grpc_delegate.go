package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCOutputer struct {
	id           string
	runUpClient  proto.StepRunner_RunUpClient
	stopCh       chan struct{}
	ctx          context.Context
	stepResultCh chan *proto.StepResult
	stepResult   *proto.StepResult
}

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
		if err == nil {
			stepResultCh <- res.Result
		} else {
			stepResultCh <- &proto.StepResult{
				Status: proto.StepResult_failure,
			}
		}
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
	return &GRPCOutputer{
		id:           id,
		runUpClient:  runUpClient,
		stopCh:       make(chan struct{}),
		ctx:          ctx,
		stepResultCh: stepResultCh,
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
				fmt.Printf("run up connection closed\n")
				return
			}
			if err != nil {
				// TODO errors should be send back down for the delegate to deal with
				fmt.Printf("error getting run request: %v\n", err)
				time.Sleep(time.Second)
				continue
			}
			fmt.Printf("got run up request %v\n", req.Id)

			// Create global and steps contexts from the request
			env := NewEnvironment(req.Context.GetEnv())
			globalCtx, err := NewGlobalContext(env)
			// Run request should include inputs
			inputs := map[string]*structpb.Value{}
			stepsCtx := NewStepsContext(globalCtx, req.Context.WorkDir, inputs, req.Context.GetEnv())

			specDef, err := loadSteps(req.GetSteps())
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
				err = o.runUpClient.Send(res)
				if err != nil {
					fmt.Printf("error send run response: %v\n", err)
				}
			}(req.Id, stepsCtx)
		}
	}
}

func (o *GRPCOutputer) Write(res *proto.StepResult) error {
	return fmt.Errorf("write unimplemented")
}

func (o *GRPCOutputer) Outputs() (map[string]*structpb.Value, *proto.StepResult, error) {
	defer func() {
		// Stop servicing run up requests after we get our output
		close(o.stopCh)
	}()
	fmt.Printf("waiting on outputs\n")
	o.stepResult = <-o.stepResultCh
	fmt.Printf("got outputs\n")
	var err error
	if o.stepResult.Status == proto.StepResult_failure {
		err = fmt.Errorf("step failed")
	}
	return o.stepResult.Outputs, o.stepResult, err
}

// Shamelessly copied from pkg/api/service/service.go
func loadSteps(stepsStr string) (*proto.SpecDefinition, error) {
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
