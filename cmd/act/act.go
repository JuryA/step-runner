package act

import (
	ctx "context"
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/schema/v1"
	"gitlab.com/gitlab-org/step-runner/steps/action"
)

func NewCmd() *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "action",
		Short: "Run an action",
		RunE:  func(cmd *cobra.Command, args []string) error { return run(options) },
	}

	cmd.Flags().StringVar(options.WorkDir, "work dir", "", "provide work dir to action")
	cmd.Flags().Var(options.ProtoStep, "proto-step", "provide proto.Step to action")
	cmd.Flags().Var(options.Env, "env", "provide environment to action")
	cmd.Flags().Var(options.Job, "job", "provide job to action")
	return cmd
}

// Require:
//   the proto.Step. Use the proto to marshal and unmarshal it, pass in as a string. Should include the inputs
//   the steps context.
//     the job, as a JSON name/value map.
//     the env, as a JSON name/value map.
//     global context work dir
//     in theory, results of previously executed steps
func run(options *Options) error {
	// Can't do in-memory caching. Can't have in-memory locks.
	// Can't pass the context to another process.
	// I want "action" to be a step: speed of development, consistency with logging, minimize errors, provide output file, etc.
	// I also need a spec def. to build a step result, determine how to read the output_file, make sure inputs match those provided, etc.
	// how can we stop this causing issues if we run it more than once at a time? Can't use go singleflight

	if err := options.Validate(); err != nil {
		return err
	}

	globalCtx := runner.NewGlobalContext(runner.NewEnvironment(options.Env.Values()))
	globalCtx.WorkDir = *options.WorkDir
	globalCtx.Job = options.Job.Values()

	// TODO: define this in steps/action as a string, compile it here.
	spec := &schema.Spec{
		Spec: &schema.Signature{
			Inputs:  map[string]schema.Input{},
			Outputs: "delegate",
		},
	}

	protoSpec, err := spec.Compile()
	if err != nil {
		return fmt.Errorf("compile proto spec: %w", err)
	}

	// TODO: validate the inputs against the schema
	// TODO: set input defaults
	// TODO: set output defaults
	// TODO: pass in steps context as the input to act
	stepsCtx, err := runner.NewStepsContext(globalCtx, *options.WorkDir, options.ProtoStep.Step.Inputs, globalCtx.Env)

	if err != nil {
		return err
	}

	defer stepsCtx.Cleanup()

	// Hardcode action.Run here. Ideally, this would be passed in.
	// Interesting that action.Run implements domain.Step
	step := runner.NewInMemoryStep("action", protoSpec, action.Run)
	_, err = step.Run(ctx.Background(), stepsCtx)

	// errors will suck
	return err
}
