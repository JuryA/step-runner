package runner

import (
	"fmt"
	"os"

	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCOutputer struct {
	id         string
	socketFile string
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
	return &GRPCOutputer{
		id:         grpcDelegate.Id,
		socketFile: grpcDelegate.SocketFile,
	}, nil
}

func Outputs() (map[string]*structpb.Value, *proto.StepResult, error) {

}
