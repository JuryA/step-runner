package runner

import (
	"context"
	"fmt"
	"os"

	"gitlab.com/gitlab-org/step-runner/pkg/api/client/extended"
	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCOutputer struct {
	jobId          string
	runUpClient    proto.StepRunner_RunUpClient
	extendedClient *extended.StepRunnerClient
	stopCh         chan struct{}
	ctx            context.Context
}

type alreadyDialed struct {
	conn *grpc.ClientConn
}

func (a *alreadyDialed) Dial() (*grpc.ClientConn, error) {
	return a.conn, nil
}

func LoadFromFile(filename string) (*GRPCOutputer, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file %v: %w", filename, err)
	}
	grpcDelegate := &proto.GrpcDelegate{}
	if err := protojson.Unmarshal(data, grpcDelegate); err != nil {
		return nil, fmt.Errorf("reading output_file as grpc delegate: %w", err)
	}
	conn, err := grpc.Dial(grpcDelegate.SocketFile, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	jobId := grpcDelegate.Id
	stepRunnerClient := proto.NewStepRunnerClient(conn)

	// We subscribe to run up requests for the specific job_id the delegate gave us
	ctx := context.WithValue(context.Background(), "job_id", jobId)
	runUpClient, err := stepRunnerClient.RunUp(ctx)

	conn, err = grpc.Dial(grpcDelegate.SocketFile, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	extendedClient, err := extended.New(&alreadyDialed{conn})
	if err != nil {
		return nil, err
	}
	return &GRPCOutputer{
		jobId:          jobId,
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
			if err != nil {
				// TODO errors should be send back down for the delegate to deal with
				panic(err)
			}

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

func (o *GRPCOutputer) Outputs() (map[string]*structpb.Value, *proto.StepResult, error) {
	defer func() {
		// Stop servicing run up requests after we get our output
		close(o.stopCh)
	}()
	// Use the extended client to get results
	return nil, nil, nil
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
