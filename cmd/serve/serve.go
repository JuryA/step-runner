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
	"gitlab.com/gitlab-org/step-runner/pkg/di"
	"gitlab.com/gitlab-org/step-runner/proto"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the step-runner gRPC service",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sigChan := make(chan os.Signal, 1)
			diContainer := di.NewContainer()

			stepRunnerService, err := diContainer.StepRunnerService()
			if err != nil {
				return fmt.Errorf("initializing step-runner: %w", err)
			}

			socketAddr, err := GetSocketAddr(args)
			if err != nil {
				return fmt.Errorf("initializing step-runner: %w", err)

			}

			if err := NewServeCmd(stepRunnerService, socketAddr, sigChan).Run(); err != nil {
				return fmt.Errorf("serving step-runner: %w", err)
			}

			return nil
		},
	}
}

func GetSocketAddr(args []string) (*net.UnixAddr, error) {
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

type ServeCmd struct {
	stepRunnerService *service.StepRunnerService
	grpcServer        *grpc.Server
	sigChan           chan os.Signal
	socketAddr        *net.UnixAddr
}

func NewServeCmd(stepRunnerService *service.StepRunnerService, socketAddr *net.UnixAddr, sigChan chan os.Signal) *ServeCmd {
	return &ServeCmd{
		stepRunnerService: stepRunnerService,
		grpcServer:        grpc.NewServer(),
		sigChan:           sigChan,
		socketAddr:        socketAddr,
	}
}

func (sc *ServeCmd) Run() error {
	listener, err := sc.Listen()
	if err != nil {
		return err
	}

	return sc.Serve(listener)
}

func (sc *ServeCmd) Listen() (net.Listener, error) {
	go func() {
		signal.Notify(sc.sigChan, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sc.sigChan
		log.Printf("received '%s' signal; shutting down.", sig)
		sc.grpcServer.GracefulStop()
	}()

	listener, err := net.ListenUnix("unix", sc.socketAddr)
	if err != nil {
		return nil, fmt.Errorf("opening socket: %w", err)
	}

	return listener, nil
}

func (sc *ServeCmd) Serve(listener net.Listener) error {
	proto.RegisterStepRunnerServer(sc.grpcServer, sc.stepRunnerService)
	return sc.grpcServer.Serve(listener)
}
