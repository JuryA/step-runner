package delegate

import (
	"fmt"
	"os"

	"gitlab.com/gitlab-org/step-runner/pkg/api/client/basic"
	"gitlab.com/gitlab-org/step-runner/pkg/api/client/extended"
	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCOutputer struct {
	jobId          string
	basicClient    *basic.StepRunnerClient
	extendedClient *extended.StepRunnerClient
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
	basicClient := basic.New(conn)
	conn, err = grpc.Dial(grpcDelegate.SocketFile, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	extendedClient, err := extended.New(&alreadyDialed{conn})
	if err != nil {
		return nil, err
	}
	return &GRPCOutputer{
		jobId:          grpcDelegate.Id,
		basicClient:    basicClient,
		extendedClient: extendedClient,
	}, nil
}

func (o *GRPCOutputer) Outputs() (map[string]*structpb.Value, *proto.StepResult, error) {

	return nil, nil, nil
}
