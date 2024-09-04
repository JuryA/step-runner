package proxy

import (
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/step-runner/pkg/api"
	proxyapi "gitlab.com/gitlab-org/step-runner/pkg/api/proxy"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "proxy",
		Short: "Tunnel gRPC requests/responses from stdin/stdout to the service listening on a local socket",
		Args:  cobra.ExactArgs(0),
		RunE:  run,
	}
}

func run(cmd *cobra.Command, args []string) error {
	conn, err := net.Dial("unix", api.DefaultSocketPath())
	if err != nil {
		return fmt.Errorf("dialing: %w", err)
	}
	return proxyapi.Proxy(os.Stdin, os.Stdout, conn)
}
