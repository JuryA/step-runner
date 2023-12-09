package main

import (
	"os"

	"gitlab.com/gitlab-org/step-runner/cmd"
	"gitlab.com/gitlab-org/step-runner/cmd/ci"
	"gitlab.com/gitlab-org/step-runner/cmd/replay"
)

func main() {
	cmd.RootCmd.AddCommand(ci.Cmd)
	cmd.RootCmd.AddCommand(replay.Cmd)
	err := cmd.RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
