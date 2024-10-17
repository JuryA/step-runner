package main

import (
	"fmt"
	"os"

	"gitlab.com/gitlab-org/step-runner/cmd"
	"gitlab.com/gitlab-org/step-runner/cmd/ci"
	"gitlab.com/gitlab-org/step-runner/cmd/proxy"
	"gitlab.com/gitlab-org/step-runner/cmd/run"
	"gitlab.com/gitlab-org/step-runner/cmd/serve"
)

// stepRunnerVersion is set when the step runner is compiled in the Dockerfile 
var stepRunnerVersion = "UNKNOWN (unset in build flags)"

func init() {
	fmt.Printf("\nStep Runner version: %s\n", stepRunnerVersion)
	fmt.Printf("See https://gitlab.com/gitlab-org/step-runner/-/blob/main/CHANGELOG.md for changes.\n\n")
}

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
