package serve

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"gitlab.com/gitlab-org/step-runner/pkg/api"
	"gitlab.com/gitlab-org/step-runner/pkg/api/service"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"

	"gitlab.com/gitlab-org/step-runner/proto"
)

var Cmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the step-runner gRPC service",
	Args:  cobra.ExactArgs(0),
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {
	var grpcServer *grpc.Server
	sigChan := make(chan os.Signal, 1)

	go func() {
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigChan
		log.Printf("received '%s' signal; shutting down.", sig.String())
		grpcServer.GracefulStop()
	}()

	stepCache, err := cache.New()

	if err != nil {
		return fmt.Errorf("failed to run service: %w", err)
	}

	srv := service.New(stepCache)

	listener, err := net.Listen("unix", api.DefaultSocketPath())
	if err != nil {
		return fmt.Errorf("failed to open open socket %q for listening: %w", api.DefaultSocketPath(), err)
	}

	grpcServer = grpc.NewServer()
	proto.RegisterStepRunnerServer(grpcServer, srv)

	log.Printf("step-runner service listening on %v", listener.Addr())
	return grpcServer.Serve(listener)
}
