package serve

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"gitlab.com/gitlab-org/step-runner/pkg/api"
	"gitlab.com/gitlab-org/step-runner/pkg/api/service"
	"gitlab.com/gitlab-org/step-runner/pkg/cache"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"

	"gitlab.com/gitlab-org/step-runner/proto"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the step-runner gRPC service",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sigChan := make(chan os.Signal, 1)
			if err := run(cmd, args, sigChan); err != nil {
				return fmt.Errorf("serving step-runner: %w", err)
			}
			return nil
		},
	}
}

func run(_ *cobra.Command, args []string, sigChan chan os.Signal) error {
	var grpcServer *grpc.Server

	go func() {
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigChan
		log.Printf("received '%s' signal; shutting down.", sig.String())
		grpcServer.GracefulStop()
	}()

	socketAddr, err := getSocketAddr(args)
	if err != nil {
		return err
	}

	stepCache, err := cache.New()
	if err != nil {
		return fmt.Errorf("initializing cache: %w", err)
	}

	env, err := runner.NewEnvironmentFromOS()
	if err != nil {
		return fmt.Errorf("initializing environment: %w", err)
	}

	srv := service.New(stepCache, env)

	listener, err := net.ListenUnix("unix", socketAddr)
	if err != nil {
		return fmt.Errorf("opening socket: %w", err)
	}

	grpcServer = grpc.NewServer()
	proto.RegisterStepRunnerServer(grpcServer, srv)

	log.Printf("step-runner service listening on %v", listener.Addr())
	return grpcServer.Serve(listener)
}

func getSocketAddr(args []string) (*net.UnixAddr, error) {
	if len(args) == 0 {
		return api.SocketAddr(api.DefaultSocketPath()), nil
	}
	socketDir := strings.TrimSpace(args[0])

	if socketDir == "" {
		return nil, fmt.Errorf("invalid empty socket dir")
	}

	fi, err := os.Stat(socketDir)
	if err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("invalid socket dir %s", socketDir)
	}
	return api.SocketAddr(filepath.Join(socketDir, "step-runner.sock")), nil
}
