package executor

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/proto/reference/environment"
	"gitlab.com/gitlab-org/step-runner/proto/reference/expression"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CompositionExecutor handles execution of function compositions
type CompositionExecutor struct {
	envManager   *environment.Manager
	evaluator    *expression.Evaluator
	funcExecutor *FunctionExecutor
}

// NewCompositionExecutor creates a new composition executor
func NewCompositionExecutor(envManager *environment.Manager, evaluator *expression.Evaluator, funcExecutor *FunctionExecutor) *CompositionExecutor {
	return &CompositionExecutor{
		envManager:   envManager,
		evaluator:    evaluator,
		funcExecutor: funcExecutor,
	}
}

// Execute runs a composition and returns a Result
func (c *CompositionExecutor) Execute(ctx context.Context, function *proto.Function, env *proto.Environment) (*proto.Result, error) {
	// Get the composition from the function
	body := function.GetBody()
	if body == nil {
		return nil, fmt.Errorf("function body is nil")
	}

	composition := body.GetComposition()
	if composition == nil {
		return nil, fmt.Errorf("function body does not contain a composition")
	}

	// Create timing information
	startTime := time.Now()
	startTimestamp := timestamppb.New(startTime)

	// Initialize the result
	result := &proto.Result{
		Function: function,
		Return: &proto.Return{
			Outputs: make(map[string]*proto.Value),
			Exports: make(map[string]*proto.Value),
		},
		Results: &proto.Result_InvocationResults_{
			InvocationResults: &proto.Result_InvocationResults{
				Results: []*proto.Result{},
			},
		},
		Timing: &proto.ExecutionTiming{
			StartTime: startTimestamp,
		},
	}

	// Keep track of current environment as we execute each invocation
	currentEnv := env

	// Execute each invocation in sequence
	for _, invocExpr := range composition.Invocations {
		// Resolve the invocation if it's an expression
		var invocation *proto.Invocation
		
		switch inv := invocExpr.GetInvocationOneof().(type) {
		case *proto.InvocationOrExpression_Invocation:
			invocation = inv.Invocation
		case *proto.InvocationOrExpression_InvocationExp:
			// Evaluate the expression to get the invocation
			val, err := c.evaluator.Evaluate(inv.InvocationExp, currentEnv)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate invocation expression: %w", err)
			}
			
			// Check that the value is a map that can be converted to an Invocation
			mapVal, err := expression.GetValueAsMap(val)
			if err != nil {
				return nil, fmt.Errorf("invocation expression did not evaluate to a map: %w", err)
			}
			
			// Create invocation from map (simplified - in a real implementation would need to handle all fields)
			invocation = &proto.Invocation{}
			
			// Extract name if present
			if nameVal, ok := mapVal["name"]; ok {
				nameStr, err := expression.GetValueAsString(nameVal)
				if err != nil {
					return nil, fmt.Errorf("invocation name is not a string: %w", err)
				}
				invocation.Name = &nameStr
			}
			
			// Would need to handle reference, environment, etc.
			return nil, fmt.Errorf("dynamic invocation evaluation not fully implemented")
		default:
			return nil, fmt.Errorf("unexpected invocation type: %T", inv)
		}
		
		if invocation == nil {
			return nil, fmt.Errorf("invocation is nil")
		}
		
		// Get the function reference
		reference := invocation.GetReference()
		if reference == nil {
			referenceExp := invocation.GetReferenceExp()
			if referenceExp != nil {
				// Evaluate the reference expression
				val, err := c.evaluator.Evaluate(referenceExp, currentEnv)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate reference expression: %w", err)
				}
				
				// Would need to convert the value to a Reference
				return nil, fmt.Errorf("dynamic reference evaluation not fully implemented")
			} else {
				return nil, fmt.Errorf("invocation has no reference")
			}
		}
		
		// Resolve the reference to a Function
		functionToInvoke, err := c.resolveReference(reference)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve function reference: %w", err)
		}
		
		// Get the invocation environment
		var invocationEnv *proto.Environment
		switch {
		case invocation.GetEnvironment() != nil:
			// Use the provided environment
			invocationEnv, err = c.envManager.OverlayEnvironment(currentEnv, invocation.GetEnvironment())
			if err != nil {
				return nil, fmt.Errorf("failed to overlay invocation environment: %w", err)
			}
		case invocation.GetEnvironmentExp() != nil:
			// Evaluate the environment expression
			val, err := c.evaluator.Evaluate(invocation.GetEnvironmentExp(), currentEnv)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate environment expression: %w", err)
			}
			
			// Would need to convert the value to an Environment
			return nil, fmt.Errorf("dynamic environment evaluation not fully implemented")
		default:
			// Use the current environment as is
			invocationEnv = currentEnv
		}
		
		// Execute the function
		invocationResult, err := c.funcExecutor.Execute(ctx, functionToInvoke, invocationEnv)
		if err != nil {
			return nil, fmt.Errorf("failed to execute invocation: %w", err)
		}
		
		// Store the result in the invocation
		invocation.Result = invocationResult
		
		// Add the result to the composition result
		if result, ok := result.Results.(*proto.Result_InvocationResults_); ok {
			result.InvocationResults.Results = append(result.InvocationResults.Results, invocationResult)
		}
		
		// If the invocation has a name, store its outputs in the environment
		if invocation.Name != nil && *invocation.Name != "" {
			currentEnv, err = c.envManager.StoreInvocationResult(currentEnv, *invocation.Name, invocationResult)
			if err != nil {
				return nil, fmt.Errorf("failed to store invocation result: %w", err)
			}
		}
		
		// Apply any exports from the invocation to the environment
		currentEnv, err = c.envManager.ApplyExports(currentEnv, invocationResult)
		if err != nil {
			return nil, fmt.Errorf("failed to apply invocation exports: %w", err)
		}
	}
	
	// Record end time
	endTime := time.Now()
	endTimestamp := timestamppb.New(endTime)
	durationMs := endTime.Sub(startTime).Milliseconds()
	
	// Update timing info
	result.Timing.EndTime = endTimestamp
	result.Timing.DurationMs = &durationMs
	
	// Transfer the return parameters from the final environment to the result outputs
	for k, v := range currentEnv.ReturnParameters {
		result.Return.Outputs[k] = v
	}
	
	return result, nil
}

// resolveReference resolves a reference to a Function
func (c *CompositionExecutor) resolveReference(ref *proto.Reference) (*proto.Function, error) {
	if ref == nil {
		return nil, fmt.Errorf("reference is nil")
	}
	
	// Check the type of reference
	switch r := ref.GetMaterializedOneof().(type) {
	case *proto.Reference_Local_:
		// Handle local file reference
		return c.resolveLocalReference(r.Local)
	case *proto.Reference_Git_:
		// Handle git reference (not implemented in this example)
		return nil, fmt.Errorf("git references are not implemented")
	case *proto.Reference_Oci_:
		// Handle OCI reference (not implemented in this example)
		return nil, fmt.Errorf("OCI references are not implemented")
	case *proto.Reference_Dist_:
		// Handle dist reference (not implemented in this example)
		return nil, fmt.Errorf("dist references are not implemented")
	default:
		return nil, fmt.Errorf("unsupported reference type: %T", r)
	}
}

// resolveLocalReference resolves a local file reference to a Function
func (c *CompositionExecutor) resolveLocalReference(local *proto.Reference_Local) (*proto.Function, error) {
	if local == nil {
		return nil, fmt.Errorf("local reference is nil")
	}
	
	// This is a simplified implementation - in a real implementation, 
	// you would read and parse the proto file from disk
	
	// For now, we'll just create a simple echo function as a placeholder
	// In reality, this would load the Function from the specified file
	
	echoFunction := &proto.Function{
		Body: &proto.Body{
			BodyOneof: &proto.Body_Exec{
				Exec: &proto.Exec{
					Command: []string{"echo", "Function loaded from local reference"},
				},
			},
			EnvironmentOneof: &proto.Body_Environment{
				Environment: &proto.Environment{
					EnvVars:            make(map[string]*proto.Expression),
					CompositionOutputs: make(map[string]*proto.Environment_Map),
					Scopes:             make(map[string]*proto.Value),
					InputParameters:    make(map[string]*proto.Expression),
					ReturnParameters:   make(map[string]*proto.Value),
				},
			},
		},
	}
	
	return echoFunction, nil
}