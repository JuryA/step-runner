package serve

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"gitlab.com/gitlab-org/step-runner/pkg/api/service"

	"gitlab.com/gitlab-org/step-runner/proto"
)

var socketPath = path.Join(os.TempDir(), "step-runner.sock")

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

	srv, err := service.New()
	if err != nil {
		return fmt.Errorf("failed to create step-runner request handler: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to open open socket %q for listening: %w", socketPath, err)
	}

	grpcServer = grpc.NewServer()
	proto.RegisterStepRunnerServer(grpcServer, srv)

	log.Printf("step-runner service listening on %v", listener.Addr())
	return grpcServer.Serve(listener)
}
