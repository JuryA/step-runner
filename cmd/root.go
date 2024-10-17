package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stepRunnerVersion is set when the step runner is compiled in the Dockerfile
var stepRunnerVersion = "UNKNOWN (unset in build flags)"

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "step-runner",
		Short:        "Step Runner executes a series of CI steps",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.CalledAs() != "proxy" {
				fmt.Printf("\nStep Runner version: %s\n", stepRunnerVersion)
				fmt.Printf("See https://gitlab.com/gitlab-org/step-runner/-/blob/main/CHANGELOG.md for changes.\n\n")
			}
		},
	}
}
