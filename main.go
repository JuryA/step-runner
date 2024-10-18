package main

import (
	"os"

	"gitlab.com/gitlab-org/step-runner/cmd"
	"gitlab.com/gitlab-org/step-runner/cmd/ci"
	"gitlab.com/gitlab-org/step-runner/cmd/proxy"
	"gitlab.com/gitlab-org/step-runner/cmd/run"
	"gitlab.com/gitlab-org/step-runner/cmd/serve"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(ci.NewCmd())
	rootCmd.AddCommand(run.NewCmd())
	rootCmd.AddCommand(serve.NewCmd())
	rootCmd.AddCommand(proxy.NewCmd())
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
