package main

import (
	"os"

	"gitlab.com/gitlab-org/step-runner/cmd"
	"gitlab.com/gitlab-org/step-runner/cmd/ci"
)

func main() {
	cmd.RootCmd.AddCommand(ci.Cmd)
	err := cmd.RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
