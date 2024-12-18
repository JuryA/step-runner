package debug

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

type Options struct {
	Endpoint string
}

func NewCmd() *cobra.Command {
	options := &Options{}
	cmd := &cobra.Command{
		Use:   "debug [endpoint]",
		Short: "Debug running steps",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(options)
		},
	}
	cmd.Flags().StringVar(&options.Endpoint, "endpoint", "", "step-runner service endpoint")
	return cmd
}

func run(options *Options) error {
	conn, err := grpc.Dial(options.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("error connecting to endpoint: %w", err)
	}
	stepRunnerClient := proto.NewStepRunnerClient(conn)
	debugClient, err := stepRunnerClient.Debug(context.Background())
	if err != nil {
		return fmt.Errorf("error calling debug: %w", err)
	}
	defer debugClient.CloseSend()
	err = debugClient.Send(&proto.DebugRequest{
		CommandOneof: &proto.DebugRequest_Stop_{
			Stop: &proto.DebugRequest_Stop{},
		},
	})
	if err != nil {
		return fmt.Errorf("error getting initial view: %w", err)
	}
	s := &session{
		debugClient: debugClient,
		stopCh:      make(chan struct{}),
		wg:          &sync.WaitGroup{},
	}
	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		defer s.stop()
		s.read()
	}()
	go func() {
		defer s.wg.Done()
		defer s.stop()
		s.write()
	}()
	s.wg.Wait()
	return nil
}

type session struct {
	debugClient proto.StepRunner_DebugClient
	stopCh      chan struct{}
	wg          *sync.WaitGroup
}

func (s *session) done() bool {
	select {
	case <-s.stopCh:
		return true
	default:
		return false
	}
}

func (s *session) stop() {
	s.stopCh <- struct{}{}
}

func (s *session) read() {
	for {
		if s.done() {
			return
		}
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("client error: %v", err)
			return
		}
		input = strings.TrimSpace(input)
		if len(input) == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(input, "stop"):
			s.debugClient.Send(&proto.DebugRequest{
				CommandOneof: &proto.DebugRequest_Stop_{
					Stop: &proto.DebugRequest_Stop{},
				},
			})
		case input == "l" || input == "list":
			s.debugClient.Send(&proto.DebugRequest{
				CommandOneof: &proto.DebugRequest_View_{
					View: &proto.DebugRequest_View{},
				},
			})
		case input == "s" || input == "step":
			s.debugClient.Send(&proto.DebugRequest{
				CommandOneof: &proto.DebugRequest_Step_{
					Step: &proto.DebugRequest_Step{},
				},
			})
		case input == "c" || input == "continue":
			s.debugClient.Send(&proto.DebugRequest{
				CommandOneof: &proto.DebugRequest_Continue_{
					Continue: &proto.DebugRequest_Continue{},
				},
			})
		case strings.HasPrefix(input, "eval "):
			s.debugClient.Send(&proto.DebugRequest{
				CommandOneof: &proto.DebugRequest_Print_{
					Print: &proto.DebugRequest_Print{
						Expression: input[len("eval "):],
					},
				},
			})
		case strings.HasPrefix(input, "set "):
			path, value, ok := strings.Cut(strings.TrimSpace(input[len("set "):]), " ")
			if !ok {
				fmt.Printf("need path and value\n")
				continue
			}
			var untyped any
			err := json.Unmarshal([]byte(value), &untyped)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				continue
			}
			protoValue, err := structpb.NewValue(untyped)
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				continue
			}
			s.debugClient.Send(&proto.DebugRequest{
				CommandOneof: &proto.DebugRequest_Set_{
					Set: &proto.DebugRequest_Set{
						Path:  path,
						Value: protoValue,
					},
				},
			})
		default:
			fmt.Printf(`
Usage:
  stop
  l(ist)
  s(tep)
  c(ontinue)
  eval [expression]
  set [path] [value]
`)
			continue
		}
	}
}

func (s *session) write() {
	for {
		if s.done() {
			return
		}
		res, err := s.debugClient.Recv()
		if err != nil {
			fmt.Printf("server error: %v", err)
		}
		fmt.Print(res.StepView)
		fmt.Print("> ")
	}
}
