package main

import (
	"os"

	"gitlab.com/gitlab-org/step-runner/cmd"
	"gitlab.com/gitlab-org/step-runner/cmd/ci"
	"gitlab.com/gitlab-org/step-runner/cmd/service"
)

func main() {
	cmd.RootCmd.AddCommand(ci.Cmd)
	cmd.RootCmd.AddCommand(service.Cmd)
	err := cmd.RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
