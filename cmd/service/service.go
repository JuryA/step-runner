package service

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gitlab.com/gitlab-org/step-runner/pkg/service"
	"google.golang.org/grpc"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Networking string

const (
	UnixSocket Networking = "unix"
	TCPAddress Networking = "tcp"
)

type Serve struct {
	Address    string     `arg:"-a,--address" default:"127.0.0.1:8765" help:"tcp networking address"`
	Socket     string     `arg:"-s,--socket" default:"/tmp/step-runner.sock" help:"unix domain socket path"`
	Networking Networking `arg:"-n,--networking" default:"unix" help:"networking type [unix,tcp]"`
}

func (s *Serve) Run() error {
	var grpcServer *grpc.Server
	sigChan := make(chan os.Signal, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigChan
		log.Printf("received '%s' signal; shutting down.", sig.String())
		grpcServer.GracefulStop()
	}()

	lis, err := s.listener()
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	grpcServer = grpc.NewServer(opts...)
	proto.RegisterStepRunnerServer(grpcServer, newServer())

	log.Printf("listening on %v", lis.Addr())
	return grpcServer.Serve(lis)
}

func (s *Serve) listener() (net.Listener, error) {
	switch s.Networking {
	case TCPAddress:
		return net.Listen("tcp", s.Address)
	case UnixSocket:
		return net.Listen("unix", s.Socket)
	}
	return nil, fmt.Errorf("invalid networking %q", s.Networking)
}

func newServer() *service.StepRunnerServer {
	s, _ := service.NewServer()

	// more init here...
	return s
}
