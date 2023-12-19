package service

import (
	"fmt"
	"log"
	"net"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/step-runner/pkg/service"
	"google.golang.org/grpc"

	"gitlab.com/gitlab-org/step-runner/proto"
)

const port = 8765

var Cmd = &cobra.Command{
	Use:   "serve",
	Short: "Run StepRunner server",
	Args:  cobra.ExactArgs(0),
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	proto.RegisterStepRunnerServer(grpcServer, newServer())

	log.Printf("listening on port %d", port)
	return grpcServer.Serve(lis)
}

func newServer() *service.StepRunnerServer {
	s, _ := service.NewServer()

	// more init here...
	return s
}
