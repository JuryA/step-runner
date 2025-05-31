package environment

import (
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/step-runner/proto"
	"gitlab.com/gitlab-org/step-runner/proto/reference/expression"
)

// Manager handles environment operations
type Manager struct {
	evaluator *expression.Evaluator
}

// NewManager creates a new environment manager
func NewManager(evaluator *expression.Evaluator) *Manager {
	return &Manager{
		evaluator: evaluator,
	}
}

// CreateInitialEnvironment creates an initial environment from the system
func (m *Manager) CreateInitialEnvironment() *proto.Environment {
	// Create base environment
	env := &proto.Environment{
		EnvVars:            make(map[string]*proto.Expression),
		CompositionOutputs: make(map[string]*proto.Environment_Map),
		Scopes:             make(map[string]*proto.Value),
		InputParameters:    make(map[string]*proto.Expression),
		ReturnParameters:   make(map[string]*proto.Value),
	}

	// Get current working directory
	workDir, err := os.Getwd()
	if err == nil {
		env.WorkDir = &workDir
	}

	// Add system environment variables
	for _, envVar := range os.Environ() {
		for i := 0; i < len(envVar); i++ {
			if envVar[i] == '=' {
				name := envVar[:i]
				value := envVar[i+1:]
				env.EnvVars[name] = &proto.Expression{
					Op: &proto.Expression_Literal{
						Literal: &proto.Value{
							Type: &proto.Value_String_{
								String_: value,
							},
						},
					},
				}
				break
			}
		}
	}

	return env
}

// OverlayEnvironment combines a base environment with an overlay
func (m *Manager) OverlayEnvironment(base, overlay *proto.Environment) (*proto.Environment, error) {
	if base == nil {
		return nil, fmt.Errorf("base environment cannot be nil")
	}

	// If overlay is nil, return a copy of the base
	if overlay == nil {
		return m.copyEnvironment(base), nil
	}

	// Create a new environment as a copy of the base
	result := m.copyEnvironment(base)

	// Overlay environment variables
	for k, v := range overlay.EnvVars {
		result.EnvVars[k] = v
	}

	// Overlay composition outputs
	for k, v := range overlay.CompositionOutputs {
		result.CompositionOutputs[k] = v
	}

	// Overlay scopes
	for k, v := range overlay.Scopes {
		result.Scopes[k] = v
	}

	// Overlay input parameters
	for k, v := range overlay.InputParameters {
		result.InputParameters[k] = v
	}

	// Overlay return parameters
	for k, v := range overlay.ReturnParameters {
		result.ReturnParameters[k] = v
	}

	// Overlay workdir if provided
	if overlay.WorkDir != nil {
		// If overlay workdir is relative, resolve it against the base workdir
		if !filepath.IsAbs(*overlay.WorkDir) && base.WorkDir != nil {
			absPath := filepath.Join(*base.WorkDir, *overlay.WorkDir)
			result.WorkDir = &absPath
		} else {
			result.WorkDir = overlay.WorkDir
		}
	}

	// Overlay funcdir if provided
	if overlay.FuncDir != nil {
		// If overlay funcdir is relative, resolve it against the base funcdir
		if !filepath.IsAbs(*overlay.FuncDir) && base.FuncDir != nil {
			absPath := filepath.Join(*base.FuncDir, *overlay.FuncDir)
			result.FuncDir = &absPath
		} else {
			result.FuncDir = overlay.FuncDir
		}
	}

	return result, nil
}

// ResolveEnvironment evaluates all expressions in an environment
func (m *Manager) ResolveEnvironment(env *proto.Environment) (*proto.Environment, error) {
	if env == nil {
		return nil, fmt.Errorf("cannot resolve nil environment")
	}

	// Create a new environment
	result := &proto.Environment{
		EnvVars:            make(map[string]*proto.Expression),
		CompositionOutputs: make(map[string]*proto.Environment_Map),
		Scopes:             make(map[string]*proto.Value),
		InputParameters:    make(map[string]*proto.Expression),
		ReturnParameters:   make(map[string]*proto.Value),
	}

	// Copy non-expression fields directly
	if env.WorkDir != nil {
		workDir := *env.WorkDir
		result.WorkDir = &workDir
	}

	if env.FuncDir != nil {
		funcDir := *env.FuncDir
		result.FuncDir = &funcDir
	}

	// Copy scopes
	for k, v := range env.Scopes {
		result.Scopes[k] = v
	}

	// Copy return parameters
	for k, v := range env.ReturnParameters {
		result.ReturnParameters[k] = v
	}

	// Copy composition outputs
	for k, v := range env.CompositionOutputs {
		outputs := &proto.Environment_Map{
			Map: make(map[string]*proto.Value),
		}
		for kk, vv := range v.Map {
			outputs.Map[kk] = vv
		}
		result.CompositionOutputs[k] = outputs
	}

	// Evaluate environment variables
	for k, expr := range env.EnvVars {
		// Create literal expressions for resolved values
		val, err := m.evaluator.Evaluate(expr, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate environment variable %s: %w", k, err)
		}
		
		result.EnvVars[k] = &proto.Expression{
			Op: &proto.Expression_Literal{
				Literal: val,
			},
		}
	}

	// Evaluate input parameters
	for k, expr := range env.InputParameters {
		// Create literal expressions for resolved values
		val, err := m.evaluator.Evaluate(expr, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate input parameter %s: %w", k, err)
		}
		
		result.InputParameters[k] = &proto.Expression{
			Op: &proto.Expression_Literal{
				Literal: val,
			},
		}
	}

	return result, nil
}

// ApplyExports applies exports from a Result to an environment
func (m *Manager) ApplyExports(env *proto.Environment, result *proto.Result) (*proto.Environment, error) {
	if env == nil {
		return nil, fmt.Errorf("environment cannot be nil")
	}

	if result == nil || result.Return == nil || len(result.Return.Exports) == 0 {
		// No exports to apply
		return env, nil
	}

	// Create a new environment as a copy of the original
	newEnv := m.copyEnvironment(env)

	// Apply exports to environment variables
	for name, value := range result.Return.Exports {
		// Create a literal expression for the exported value
		newEnv.EnvVars[name] = &proto.Expression{
			Op: &proto.Expression_Literal{
				Literal: value,
			},
		}
	}

	return newEnv, nil
}

// GetOSEnvironment converts environment variables from an Environment to OS format
func (m *Manager) GetOSEnvironment(env *proto.Environment) ([]string, error) {
	if env == nil {
		return nil, fmt.Errorf("environment cannot be nil")
	}

	// Resolve the environment first
	resolvedEnv, err := m.ResolveEnvironment(env)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve environment: %w", err)
	}

	// Convert to OS environment format
	var osEnv []string
	for name, exprValue := range resolvedEnv.EnvVars {
		// Extract the literal value
		if literal, ok := exprValue.GetOp().(*proto.Expression_Literal); ok {
			// Convert value to string
			strValue, err := expression.GetValueAsString(literal.Literal)
			if err != nil {
				return nil, fmt.Errorf("failed to convert environment variable %s to string: %w", name, err)
			}
			osEnv = append(osEnv, fmt.Sprintf("%s=%s", name, strValue))
		} else {
			return nil, fmt.Errorf("environment variable %s is not a resolved literal", name)
		}
	}

	return osEnv, nil
}

// StoreInvocationResult stores a function invocation result in the environment
func (m *Manager) StoreInvocationResult(env *proto.Environment, name string, result *proto.Result) (*proto.Environment, error) {
	if env == nil {
		return nil, fmt.Errorf("environment cannot be nil")
	}

	if result == nil || result.Return == nil {
		// No outputs to store
		return env, nil
	}

	// Create a new environment as a copy of the original
	newEnv := m.copyEnvironment(env)

	// Store the outputs in composition_outputs
	outputs := &proto.Environment_Map{
		Map: make(map[string]*proto.Value),
	}

	// Copy all return outputs
	for k, v := range result.Return.Outputs {
		outputs.Map[k] = v
	}

	// Store in the environment
	newEnv.CompositionOutputs[name] = outputs

	return newEnv, nil
}

// copyEnvironment creates a deep copy of an environment
func (m *Manager) copyEnvironment(env *proto.Environment) *proto.Environment {
	if env == nil {
		return nil
	}

	// Create a new environment
	result := &proto.Environment{
		EnvVars:            make(map[string]*proto.Expression),
		CompositionOutputs: make(map[string]*proto.Environment_Map),
		Scopes:             make(map[string]*proto.Value),
		InputParameters:    make(map[string]*proto.Expression),
		ReturnParameters:   make(map[string]*proto.Value),
	}

	// Copy work_dir and func_dir if present
	if env.WorkDir != nil {
		workDir := *env.WorkDir
		result.WorkDir = &workDir
	}

	if env.FuncDir != nil {
		funcDir := *env.FuncDir
		result.FuncDir = &funcDir
	}

	// Copy env vars
	for k, v := range env.EnvVars {
		result.EnvVars[k] = v
	}

	// Copy composition outputs
	for k, v := range env.CompositionOutputs {
		outputs := &proto.Environment_Map{
			Map: make(map[string]*proto.Value),
		}
		for kk, vv := range v.Map {
			outputs.Map[kk] = vv
		}
		result.CompositionOutputs[k] = outputs
	}

	// Copy scopes
	for k, v := range env.Scopes {
		result.Scopes[k] = v
	}

	// Copy input parameters
	for k, v := range env.InputParameters {
		result.InputParameters[k] = v
	}

	// Copy return parameters
	for k, v := range env.ReturnParameters {
		result.ReturnParameters[k] = v
	}

	return result
}