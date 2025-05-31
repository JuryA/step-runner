package expression

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gitlab-org/step-runner/proto"
)

// Evaluator handles the evaluation of expressions with a given environment
type Evaluator struct {}

// NewEvaluator creates a new expression evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate resolves an expression to a concrete value using the provided environment
func (e *Evaluator) Evaluate(expr *proto.Expression, env *proto.Environment) (*proto.Value, error) {
	if expr == nil {
		return nil, errors.New("cannot evaluate nil expression")
	}

	// Handle different expression operations
	switch op := expr.GetOp().(type) {
	case *proto.Expression_Literal:
		// Literal values are already evaluated
		return op.Literal, nil

	case *proto.Expression_Environment:
		// Return a representation of the environment as a value
		// This is mostly used as a base for other operations
		return e.environmentToValue(env)

	case *proto.Expression_Read:
		// Read from a value using a path
		return e.evaluateRead(op.Read, env)

	case *proto.Expression_Write:
		// Write a value into another value at a specific path
		return e.evaluateWrite(op.Write, env)

	case *proto.Expression_Concat:
		// Concatenate compatible values
		return e.evaluateConcat(op.Concat, env)

	default:
		return nil, fmt.Errorf("unsupported expression type: %T", op)
	}
}

// environmentToValue converts an environment to a Value (typically as a map)
func (e *Evaluator) environmentToValue(env *proto.Environment) (*proto.Value, error) {
	if env == nil {
		return nil, errors.New("cannot convert nil environment to value")
	}

	// Create a map to represent the environment
	valueMap := &proto.Value_Map{
		Values: make(map[string]*proto.Value),
	}

	// Add environment variables
	envVarsMap := &proto.Value_Map{
		Values: make(map[string]*proto.Value),
	}
	for k, v := range env.EnvVars {
		val, err := e.Evaluate(v, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate env var %s: %w", k, err)
		}
		envVarsMap.Values[k] = val
	}
	valueMap.Values["env_vars"] = &proto.Value{
		Type: &proto.Value_Map_{
			Map: envVarsMap,
		},
	}

	// Add scopes
	scopesMap := &proto.Value_Map{
		Values: make(map[string]*proto.Value),
	}
	for k, v := range env.Scopes {
		scopesMap.Values[k] = v
	}
	valueMap.Values["scopes"] = &proto.Value{
		Type: &proto.Value_Map_{
			Map: scopesMap,
		},
	}

	// Add composition outputs
	compOutputsMap := &proto.Value_Map{
		Values: make(map[string]*proto.Value),
	}
	for k, v := range env.CompositionOutputs {
		outputMap := &proto.Value_Map{
			Values: make(map[string]*proto.Value),
		}
		for kk, vv := range v.Map {
			outputMap.Values[kk] = vv
		}
		compOutputsMap.Values[k] = &proto.Value{
			Type: &proto.Value_Map_{
				Map: outputMap,
			},
		}
	}
	valueMap.Values["composition_outputs"] = &proto.Value{
		Type: &proto.Value_Map_{
			Map: compOutputsMap,
		},
	}

	// Add input parameters
	inputParamsMap := &proto.Value_Map{
		Values: make(map[string]*proto.Value),
	}
	for k, v := range env.InputParameters {
		val, err := e.Evaluate(v, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate input parameter %s: %w", k, err)
		}
		inputParamsMap.Values[k] = val
	}
	valueMap.Values["input_parameters"] = &proto.Value{
		Type: &proto.Value_Map_{
			Map: inputParamsMap,
		},
	}

	// Add return parameters
	returnParamsMap := &proto.Value_Map{
		Values: make(map[string]*proto.Value),
	}
	for k, v := range env.ReturnParameters {
		returnParamsMap.Values[k] = v
	}
	valueMap.Values["return_parameters"] = &proto.Value{
		Type: &proto.Value_Map_{
			Map: returnParamsMap,
		},
	}

	// Add work_dir and func_dir if present
	if env.WorkDir != nil {
		valueMap.Values["work_dir"] = &proto.Value{
			Type: &proto.Value_String_{
				String_: *env.WorkDir,
			},
		}
	}

	if env.FuncDir != nil {
		valueMap.Values["func_dir"] = &proto.Value{
			Type: &proto.Value_String_{
				String_: *env.FuncDir,
			},
		}
	}

	return &proto.Value{
		Type: &proto.Value_Map_{
			Map: valueMap,
		},
	}, nil
}

// evaluateRead evaluates a read operation (access a value from a structure)
func (e *Evaluator) evaluateRead(readOp *proto.OpRead, env *proto.Environment) (*proto.Value, error) {
	if readOp == nil {
		return nil, errors.New("cannot evaluate nil read operation")
	}

	// Evaluate the base value
	baseValue, err := e.Evaluate(readOp.Value, env)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate base value for read: %w", err)
	}

	// If no path, return the base value
	if len(readOp.Path) == 0 {
		return baseValue, nil
	}

	// Apply the path to navigate the value
	current := baseValue
	for i, pathExpr := range readOp.Path {
		// Evaluate the path expression to get the key/index
		pathValue, err := e.Evaluate(pathExpr, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate path element %d: %w", i, err)
		}

		// Handle different types of current value
		switch cv := current.GetType().(type) {
		case *proto.Value_Map_:
			// For map, the path element must be a string
			key, ok := pathValue.GetType().(*proto.Value_String_)
			if !ok {
				return nil, fmt.Errorf("path element %d must be a string for map access, got %T", i, pathValue.GetType())
			}

			// Get the value for the key
			value, ok := cv.Map.Values[key.String_]
			if !ok {
				return nil, fmt.Errorf("key %s not found in map at path element %d", key.String_, i)
			}
			current = value

		case *proto.Value_Array_:
			// For array, the path element must be a number
			var index int
			switch indexValue := pathValue.GetType().(type) {
			case *proto.Value_Number:
				index = int(indexValue.Number)
			case *proto.Value_String_:
				idx, err := strconv.Atoi(indexValue.String_)
				if err != nil {
					return nil, fmt.Errorf("path element %d must be a valid number for array access, got %s", i, indexValue.String_)
				}
				index = idx
			default:
				return nil, fmt.Errorf("path element %d must be a number for array access, got %T", i, pathValue.GetType())
			}

			// Check index bounds
			if index < 0 || index >= len(cv.Array.Values) {
				return nil, fmt.Errorf("index %d out of bounds for array of length %d at path element %d", index, len(cv.Array.Values), i)
			}
			current = cv.Array.Values[index]

		default:
			return nil, fmt.Errorf("cannot access path element %d in value of type %T", i, cv)
		}
	}

	return current, nil
}

// evaluateWrite evaluates a write operation (modify a value at a specific path)
func (e *Evaluator) evaluateWrite(writeOp *proto.OpWrite, env *proto.Environment) (*proto.Value, error) {
	if writeOp == nil {
		return nil, errors.New("cannot evaluate nil write operation")
	}

	// Evaluate the value to write
	value, err := e.Evaluate(writeOp.Value, env)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate value for write: %w", err)
	}

	// Evaluate the base structure
	into, err := e.Evaluate(writeOp.Into, env)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate target for write: %w", err)
	}

	// Make a deep copy of the 'into' value since we'll be modifying it
	result, err := e.deepCopyValue(into)
	if err != nil {
		return nil, fmt.Errorf("failed to copy target value: %w", err)
	}

	// If no path, just return the value to write
	if len(writeOp.Path) == 0 {
		return value, nil
	}

	// Evaluate all path elements first
	pathElements := make([]string, len(writeOp.Path))
	for i, pathExpr := range writeOp.Path {
		pathValue, err := e.Evaluate(pathExpr, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate path element %d: %w", i, err)
		}

		// Convert path element to string
		switch pv := pathValue.GetType().(type) {
		case *proto.Value_String_:
			pathElements[i] = pv.String_
		case *proto.Value_Number:
			pathElements[i] = fmt.Sprintf("%d", int(pv.Number))
		default:
			return nil, fmt.Errorf("path element %d must be a string or number, got %T", i, pathValue.GetType())
		}
	}

	// Apply the write operation
	return e.applyWrite(result, value, pathElements, 0)
}

// applyWrite recursively applies a write operation at a specific path
func (e *Evaluator) applyWrite(target, value *proto.Value, path []string, index int) (*proto.Value, error) {
	// Base case: reached the end of the path
	if index >= len(path) {
		return value, nil
	}

	pathElement := path[index]

	// Handle different types of target
	switch tv := target.GetType().(type) {
	case *proto.Value_Map_:
		// Create a copy of the map
		resultMap := &proto.Value_Map{
			Values: make(map[string]*proto.Value),
		}

		// Copy all existing entries
		for k, v := range tv.Map.Values {
			resultMap.Values[k] = v
		}

		// If we're at the last path element, set the value directly
		if index == len(path)-1 {
			resultMap.Values[pathElement] = value
		} else {
			// Get the next target in the path
			nextTarget, ok := tv.Map.Values[pathElement]
			if !ok {
				// If path doesn't exist, create it as an empty map
				nextTarget = &proto.Value{
					Type: &proto.Value_Map_{
						Map: &proto.Value_Map{
							Values: make(map[string]*proto.Value),
						},
					},
				}
			}

			// Recursively apply write to the next target
			newValue, err := e.applyWrite(nextTarget, value, path, index+1)
			if err != nil {
				return nil, err
			}
			
			// Update the map with the new value
			resultMap.Values[pathElement] = newValue
		}

		return &proto.Value{
			Type: &proto.Value_Map_{
				Map: resultMap,
			},
			Sensitive: target.Sensitive,
		}, nil

	case *proto.Value_Array_:
		// Parse the path element as an array index
		idx, err := strconv.Atoi(pathElement)
		if err != nil {
			return nil, fmt.Errorf("invalid array index '%s': %w", pathElement, err)
		}

		// Create a copy of the array
		resultArray := &proto.Value_Array{
			Values: make([]*proto.Value, len(tv.Array.Values)),
		}

		// Copy all existing elements
		copy(resultArray.Values, tv.Array.Values)

		// Check bounds
		if idx < 0 || idx >= len(resultArray.Values) {
			return nil, fmt.Errorf("array index %d out of bounds for array of length %d", idx, len(resultArray.Values))
		}

		// If we're at the last path element, set the value directly
		if index == len(path)-1 {
			resultArray.Values[idx] = value
		} else {
			// Recursively apply write to the next target
			newValue, err := e.applyWrite(resultArray.Values[idx], value, path, index+1)
			if err != nil {
				return nil, err
			}
			
			// Update the array with the new value
			resultArray.Values[idx] = newValue
		}

		return &proto.Value{
			Type: &proto.Value_Array_{
				Array: resultArray,
			},
			Sensitive: target.Sensitive,
		}, nil

	default:
		return nil, fmt.Errorf("cannot write to value of type %T at path element %d", tv, index)
	}
}

// evaluateConcat evaluates a concatenation operation
func (e *Evaluator) evaluateConcat(concatOp *proto.OpConcat, env *proto.Environment) (*proto.Value, error) {
	if concatOp == nil {
		return nil, errors.New("cannot evaluate nil concat operation")
	}

	if len(concatOp.Expression) == 0 {
		return nil, errors.New("concat operation requires at least one expression")
	}

	// Evaluate the first expression to determine the type
	first, err := e.Evaluate(concatOp.Expression[0], env)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate first expression in concat: %w", err)
	}

	// For a single expression, just return its value
	if len(concatOp.Expression) == 1 {
		return first, nil
	}

	// Handle concatenation based on the type of the first value
	switch firstType := first.GetType().(type) {
	case *proto.Value_String_:
		// String concatenation
		result := firstType.String_
		sensitive := first.Sensitive != nil && *first.Sensitive

		for i, expr := range concatOp.Expression[1:] {
			val, err := e.Evaluate(expr, env)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate expression %d in concat: %w", i+1, err)
			}

			// Check if value is a string
			str, ok := val.GetType().(*proto.Value_String_)
			if !ok {
				return nil, fmt.Errorf("expression %d in string concat must be a string, got %T", i+1, val.GetType())
			}

			// Append the string
			result += str.String_

			// Propagate sensitivity
			if val.Sensitive != nil && *val.Sensitive {
				sensitive = true
			}
		}

		return &proto.Value{
			Type: &proto.Value_String_{
				String_: result,
			},
			Sensitive: &sensitive,
		}, nil

	case *proto.Value_Map_:
		// Map merging
		result := &proto.Value_Map{
			Values: make(map[string]*proto.Value),
		}
		sensitive := first.Sensitive != nil && *first.Sensitive

		// Copy the first map
		for k, v := range firstType.Map.Values {
			result.Values[k] = v
		}

		// Merge with subsequent maps
		for i, expr := range concatOp.Expression[1:] {
			val, err := e.Evaluate(expr, env)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate expression %d in concat: %w", i+1, err)
			}

			// Check if value is a map
			m, ok := val.GetType().(*proto.Value_Map_)
			if !ok {
				return nil, fmt.Errorf("expression %d in map concat must be a map, got %T", i+1, val.GetType())
			}

			// Merge the map (later values override earlier ones)
			for k, v := range m.Map.Values {
				result.Values[k] = v
			}

			// Propagate sensitivity
			if val.Sensitive != nil && *val.Sensitive {
				sensitive = true
			}
		}

		return &proto.Value{
			Type: &proto.Value_Map_{
				Map: result,
			},
			Sensitive: &sensitive,
		}, nil

	case *proto.Value_Array_:
		// Array concatenation
		values := make([]*proto.Value, len(firstType.Array.Values))
		sensitive := first.Sensitive != nil && *first.Sensitive

		// Copy the first array
		copy(values, firstType.Array.Values)

		// Append subsequent arrays
		for i, expr := range concatOp.Expression[1:] {
			val, err := e.Evaluate(expr, env)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate expression %d in concat: %w", i+1, err)
			}

			// Check if value is an array
			arr, ok := val.GetType().(*proto.Value_Array_)
			if !ok {
				return nil, fmt.Errorf("expression %d in array concat must be an array, got %T", i+1, val.GetType())
			}

			// Extend the array
			values = append(values, arr.Array.Values...)

			// Propagate sensitivity
			if val.Sensitive != nil && *val.Sensitive {
				sensitive = true
			}
		}

		return &proto.Value{
			Type: &proto.Value_Array_{
				Array: &proto.Value_Array{
					Values: values,
				},
			},
			Sensitive: &sensitive,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported type for concat: %T", firstType)
	}
}

// deepCopyValue creates a deep copy of a Value
func (e *Evaluator) deepCopyValue(v *proto.Value) (*proto.Value, error) {
	if v == nil {
		return nil, nil
	}

	result := &proto.Value{
		Sensitive: v.Sensitive,
	}

	switch vt := v.GetType().(type) {
	case *proto.Value_String_:
		result.Type = &proto.Value_String_{
			String_: vt.String_,
		}
	case *proto.Value_Number:
		result.Type = &proto.Value_Number{
			Number: vt.Number,
		}
	case *proto.Value_Boolean:
		result.Type = &proto.Value_Boolean{
			Boolean: vt.Boolean,
		}
	case *proto.Value_Array_:
		array := &proto.Value_Array{
			Values: make([]*proto.Value, len(vt.Array.Values)),
		}
		for i, val := range vt.Array.Values {
			copy, err := e.deepCopyValue(val)
			if err != nil {
				return nil, err
			}
			array.Values[i] = copy
		}
		result.Type = &proto.Value_Array_{
			Array: array,
		}
	case *proto.Value_Map_:
		m := &proto.Value_Map{
			Values: make(map[string]*proto.Value),
		}
		for k, val := range vt.Map.Values {
			copy, err := e.deepCopyValue(val)
			if err != nil {
				return nil, err
			}
			m.Values[k] = copy
		}
		result.Type = &proto.Value_Map_{
			Map: m,
		}
	default:
		return nil, fmt.Errorf("unsupported value type for deep copy: %T", vt)
	}

	return result, nil
}

// CreateStringValue creates a string value
func CreateStringValue(s string) *proto.Value {
	return &proto.Value{
		Type: &proto.Value_String_{
			String_: s,
		},
	}
}

// CreateNumberValue creates a number value
func CreateNumberValue(n float64) *proto.Value {
	return &proto.Value{
		Type: &proto.Value_Number{
			Number: n,
		},
	}
}

// CreateBooleanValue creates a boolean value
func CreateBooleanValue(b bool) *proto.Value {
	return &proto.Value{
		Type: &proto.Value_Boolean{
			Boolean: b,
		},
	}
}

// CreateArrayValue creates an array value
func CreateArrayValue(values []*proto.Value) *proto.Value {
	return &proto.Value{
		Type: &proto.Value_Array_{
			Array: &proto.Value_Array{
				Values: values,
			},
		},
	}
}

// CreateMapValue creates a map value
func CreateMapValue(values map[string]*proto.Value) *proto.Value {
	return &proto.Value{
		Type: &proto.Value_Map_{
			Map: &proto.Value_Map{
				Values: values,
			},
		},
	}
}

// GetValueAsString extracts a string from a Value, with type conversion if needed
func GetValueAsString(v *proto.Value) (string, error) {
	if v == nil {
		return "", errors.New("cannot extract string from nil value")
	}

	switch vt := v.GetType().(type) {
	case *proto.Value_String_:
		return vt.String_, nil
	case *proto.Value_Number:
		return fmt.Sprintf("%g", vt.Number), nil
	case *proto.Value_Boolean:
		return strconv.FormatBool(vt.Boolean), nil
	default:
		return "", fmt.Errorf("cannot convert value of type %T to string", vt)
	}
}

// GetValueAsNumber extracts a number from a Value, with type conversion if needed
func GetValueAsNumber(v *proto.Value) (float64, error) {
	if v == nil {
		return 0, errors.New("cannot extract number from nil value")
	}

	switch vt := v.GetType().(type) {
	case *proto.Value_Number:
		return vt.Number, nil
	case *proto.Value_String_:
		return strconv.ParseFloat(vt.String_, 64)
	case *proto.Value_Boolean:
		if vt.Boolean {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert value of type %T to number", vt)
	}
}

// GetValueAsBoolean extracts a boolean from a Value, with type conversion if needed
func GetValueAsBoolean(v *proto.Value) (bool, error) {
	if v == nil {
		return false, errors.New("cannot extract boolean from nil value")
	}

	switch vt := v.GetType().(type) {
	case *proto.Value_Boolean:
		return vt.Boolean, nil
	case *proto.Value_String_:
		return strconv.ParseBool(vt.String_)
	case *proto.Value_Number:
		return vt.Number != 0, nil
	default:
		return false, fmt.Errorf("cannot convert value of type %T to boolean", vt)
	}
}

// GetValueAsMap extracts a map from a Value
func GetValueAsMap(v *proto.Value) (map[string]*proto.Value, error) {
	if v == nil {
		return nil, errors.New("cannot extract map from nil value")
	}

	m, ok := v.GetType().(*proto.Value_Map_)
	if !ok {
		return nil, fmt.Errorf("value is not a map, got %T", v.GetType())
	}

	return m.Map.Values, nil
}

// GetValueAsArray extracts an array from a Value
func GetValueAsArray(v *proto.Value) ([]*proto.Value, error) {
	if v == nil {
		return nil, errors.New("cannot extract array from nil value")
	}

	a, ok := v.GetType().(*proto.Value_Array_)
	if !ok {
		return nil, fmt.Errorf("value is not an array, got %T", v.GetType())
	}

	return a.Array.Values, nil
}