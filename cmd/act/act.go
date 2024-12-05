package act

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	"gitlab.com/gitlab-org/step-runner/steps/action"
)

func NewCmd() *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "action",
		Short: "Run an action",
		RunE:  func(cmd *cobra.Command, args []string) error { return run(options) },
	}

	cmd.Flags().Var(options.CLIStepsContext, "steps-context", "provide steps context for action")
	return cmd
}

// New process limitations.
// Can't do in-memory caching.
// Can't have in-memory locks.
// Can't pass the context to another process. (can use signals, gRPC context)
// Can't use packages like singleflight to optimize fetching remote resources
func run(options *Options) error {
	if err := options.Validate(); err != nil {
		return err
	}

	// TODO: validate the inputs against the schema OR has it already been done?
	stepsCtx, err := options.ToStepsContext()
	if err != nil {
		return fmt.Errorf("parsing steps context: %w", err)
	}

	// don't call stepsCtx clean up. It should be run by the caller.
	return action.Run(context.Background(), stepsCtx)
}
