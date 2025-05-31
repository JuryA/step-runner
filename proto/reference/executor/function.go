package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/gitlab-org/step-runner/proto"
	"github.com/gitlab-org/step-runner/proto/reference/environment"
	"github.com/gitlab-org/step-runner/proto/reference/expression"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FunctionExecutor handles execution of functions
type FunctionExecutor struct {
	envManager         *environment.Manager
	evaluator          *expression.Evaluator
	execExecutor       *ExecExecutor
	compositionExecutor *CompositionExecutor
}

// NewFunctionExecutor creates a new function executor
func NewFunctionExecutor(envManager *environment.Manager, evaluator *expression.Evaluator) *FunctionExecutor {
	executor := &FunctionExecutor{
		envManager: envManager,
		evaluator:  evaluator,
	}
	
	// Create the exec executor
	executor.execExecutor = NewExecExecutor(envManager)
	
	// Create the composition executor with a reference to this function executor
	executor.compositionExecutor = NewCompositionExecutor(envManager, evaluator, executor)
	
	return executor
}

// Execute runs a function and returns a Result
func (f *FunctionExecutor) Execute(ctx context.Context, function *proto.Function, env *proto.Environment) (*proto.Result, error) {
	// Resolve the function body if it's an expression
	resolvedFunction, err := f.resolveFunction(function, env)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve function: %w", err)
	}
	
	// Get the body from the function
	body := resolvedFunction.GetBody()
	if body == nil {
		return nil, fmt.Errorf("function body is nil")
	}
	
	// Resolve the body's environment if provided
	var bodyEnv *proto.Environment
	switch {
	case body.GetEnvironment() != nil:
		// Overlay the body environment on the input environment
		bodyEnv, err = f.envManager.OverlayEnvironment(env, body.GetEnvironment())
		if err != nil {
			return nil, fmt.Errorf("failed to overlay body environment: %w", err)
		}
	case body.GetEnvironmentExp() != nil:
		// Evaluate the environment expression
		evalResult, err := f.evaluator.Evaluate(body.GetEnvironmentExp(), env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate body environment expression: %w", err)
		}
		
		// Convert the value to an environment (simplified implementation)
		// In a real implementation, you would convert the Value to an Environment
		return nil, fmt.Errorf("environment expression evaluation not fully implemented")
	default:
		// Use the input environment as is
		bodyEnv = env
	}
	
	// Create timing information
	startTime := time.Now()
	startTimestamp := timestamppb.New(startTime)
	
	// Check the function type and execute accordingly
	var result *proto.Result
	switch {
	case body.GetExec() != nil:
		// Execute the function as a system exec
		result, err = f.execExecutor.Execute(ctx, resolvedFunction, bodyEnv)
		if err != nil {
			return nil, fmt.Errorf("failed to execute exec function: %w", err)
		}
	case body.GetComposition() != nil:
		// Execute the function as a composition
		result, err = f.compositionExecutor.Execute(ctx, resolvedFunction, bodyEnv)
		if err != nil {
			return nil, fmt.Errorf("failed to execute composition function: %w", err)
		}
	default:
		return nil, fmt.Errorf("function body has no execution type")
	}
	
	// If the result doesn't have timing info, add it
	if result.Timing == nil {
		// Record end time
		endTime := time.Now()
		endTimestamp := timestamppb.New(endTime)
		durationMs := endTime.Sub(startTime).Milliseconds()
		
		result.Timing = &proto.ExecutionTiming{
			StartTime:  startTimestamp,
			EndTime:    endTimestamp,
			DurationMs: &durationMs,
		}
	}
	
	return result, nil
}

// resolveFunction resolves a function, handling the case where the body is an expression
func (f *FunctionExecutor) resolveFunction(function *proto.Function, env *proto.Environment) (*proto.Function, error) {
	if function == nil {
		return nil, fmt.Errorf("function is nil")
	}
	
	// If the body is a direct Body, no resolution needed
	if function.GetBody() != nil {
		return function, nil
	}
	
	// If the body is an expression, evaluate it
	bodyExp := function.GetBodyExp()
	if bodyExp != nil {
		// Evaluate the body expression
		evalResult, err := f.evaluator.Evaluate(bodyExp, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate body expression: %w", err)
		}
		
		// Convert the value to a Body (simplified implementation)
		// In a real implementation, you would convert the Value to a Body
		return nil, fmt.Errorf("body expression evaluation not fully implemented")
	}
	
	return nil, fmt.Errorf("function has no body")
}