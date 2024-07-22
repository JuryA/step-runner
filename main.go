package main

import (
	"os"

	"gitlab.com/gitlab-org/step-runner/cmd"
	"gitlab.com/gitlab-org/step-runner/cmd/ci"
	"gitlab.com/gitlab-org/step-runner/cmd/proxy"
	"gitlab.com/gitlab-org/step-runner/cmd/serve"
)

func main() {
	cmd.RootCmd.AddCommand(ci.Cmd)
	cmd.RootCmd.AddCommand(serve.Cmd)
	cmd.RootCmd.AddCommand(proxy.Cmd)
	err := cmd.RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
