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

const port = 8765

type Serve struct {
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

	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	grpcServer = grpc.NewServer(opts...)
	proto.RegisterStepRunnerServer(grpcServer, newServer())

	log.Printf("listening on %v", lis.Addr())
	return grpcServer.Serve(lis)
}

func newServer() *service.StepRunnerServer {
	s, _ := service.NewServer()

	// more init here...
	return s
}
