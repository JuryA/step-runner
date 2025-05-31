package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/gitlab-org/step-runner/proto"
	"github.com/gitlab-org/step-runner/proto/reference/environment"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExecExecutor handles execution of system exec calls
type ExecExecutor struct {
	envManager *environment.Manager
}

// NewExecExecutor creates a new system exec executor
func NewExecExecutor(envManager *environment.Manager) *ExecExecutor {
	return &ExecExecutor{
		envManager: envManager,
	}
}

// Execute runs a system command and returns a Result
func (e *ExecExecutor) Execute(ctx context.Context, function *proto.Function, env *proto.Environment) (*proto.Result, error) {
	// Get the exec command from the function
	body := function.GetBody()
	if body == nil {
		return nil, fmt.Errorf("function body is nil")
	}

	exec := body.GetExec()
	if exec == nil {
		return nil, fmt.Errorf("function body does not contain an exec")
	}

	// Resolve the environment
	resolvedEnv, err := e.envManager.ResolveEnvironment(env)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve environment: %w", err)
	}

	// Prepare the command
	if len(exec.Command) == 0 {
		return nil, fmt.Errorf("exec command is empty")
	}

	// Create the command
	cmd := exec.CommandContext(ctx, exec.Command[0], exec.Command[1:]...)

	// Set the environment
	osEnv, err := e.envManager.GetOSEnvironment(resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to get OS environment: %w", err)
	}
	cmd.Env = osEnv

	// Set the working directory if specified
	if resolvedEnv.WorkDir != nil {
		cmd.Dir = *resolvedEnv.WorkDir
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Create timing information
	startTime := time.Now()
	startTimestamp := timestamppb.New(startTime)

	// Run the command
	err = cmd.Run()

	// Record end time
	endTime := time.Now()
	endTimestamp := timestamppb.New(endTime)
	durationMs := endTime.Sub(startTime).Milliseconds()

	// Create timing info
	timing := &proto.ExecutionTiming{
		StartTime:  startTimestamp,
		EndTime:    endTimestamp,
		DurationMs: &durationMs,
	}

	// Create result
	result := &proto.Result{
		Function: function,
		Return: &proto.Return{
			Outputs: make(map[string]*proto.Value),
			Exports: make(map[string]*proto.Value),
		},
		Timing: timing,
	}

	// Add stdout and stderr to the outputs
	result.Return.Outputs["stdout"] = &proto.Value{
		Type: &proto.Value_String_{
			String_: stdout.String(),
		},
	}
	result.Return.Outputs["stderr"] = &proto.Value{
		Type: &proto.Value_String_{
			String_: stderr.String(),
		},
	}

	// Handle exit code
	if err != nil {
		// Try to get the exit code
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			exitCode := int32(exitErr.ExitCode())
			result.Results = &proto.Result_ExitCode{
				ExitCode: exitCode,
			}
			
			// Add error information
			result.Error = &proto.Error{
				Code:    stringPtr(fmt.Sprintf("exit_code_%d", exitCode)),
				Message: stringPtr(err.Error()),
				Details: stringPtr(stderr.String()),
				Source:  stringPtr("system"),
			}
		} else {
			// Some other error occurred
			result.Results = &proto.Result_ExitCode{
				ExitCode: 1,
			}
			
			// Add error information
			result.Error = &proto.Error{
				Code:    stringPtr("execution_error"),
				Message: stringPtr(err.Error()),
				Details: stringPtr(stderr.String()),
				Source:  stringPtr("system"),
			}
		}
	} else {
		// Success
		result.Results = &proto.Result_ExitCode{
			ExitCode: 0,
		}
	}

	return result, nil
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}