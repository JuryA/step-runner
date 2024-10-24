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
	"gitlab.com/gitlab-org/step-runner/schema/v1"
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
	stepRunnerClient.Run(context.Background(), &proto.RunRequest{
		Id:           id,
		Continuation: delegation.Continuation,
		SetupResult:  delegation.SetupResult,
	})

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
				panic(err)
			}
			log.Printf("got run up request\n")

			// Create global and steps contexts from the request
			req.GetEnv()
			env := NewEnvironment(req.GetEnv())
			globalCtx, err := NewGlobalContext(env)
			// Run request should include inputs
			inputs := map[string]*structpb.Value{}
			stepsCtx := NewStepsContext(globalCtx, req.WorkDir, inputs, req.GetEnv())

			// This should already be complied.
			specDef, err := loadSteps(req.Steps)
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
				res := &proto.FollowStepsResponse{
					Id:     id,
					Result: stepResult,
				}
				o.runUpClient.Send(res)
			}(req.Id, stepsCtx)
		}
	}
}

func (o *GRPCOutputer) Write(res *proto.StepResult) error {
	o.stepResult = res
	return nil
}

func (o *GRPCOutputer) Outputs() (map[string]*structpb.Value, *proto.StepResult, error) {
	defer func() {
		// Stop servicing run up requests after we get our output
		close(o.stopCh)
	}()
	// Use the extended client to get results
	var res *proto.StepResult
	follower := &extended.FollowOutput{
		Logs:        os.Stdout,
		StepResults: o,
	}
	_, err := o.extendedClient.Follow(context.Background(), o.id, follower)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
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
