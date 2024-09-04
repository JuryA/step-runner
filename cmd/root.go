package cmd

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "step-runner",
		Short:        "Step Runner executes a series of CI steps",
		SilenceUsage: true,
	}
}
