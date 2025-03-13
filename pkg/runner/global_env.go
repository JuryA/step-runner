package runner

import (
	"fmt"
	"maps"
	"strings"
)

var LogLevelEnvName = "CI_STEPS_LOG_LEVEL"

// GlobalEnvironment sets environment variables that are always available for any step
func GlobalEnvironment(parent *Environment, jobVars map[string]string) (*Environment, error) {
	globalEnv := make(map[string]string)

	level, err := logLevel(parent, jobVars)
	if err != nil {
		return nil, fmt.Errorf("init global environment: %w", err)
	}
	maps.Copy(globalEnv, level)

	return parent.AddLexicalScope(NewEnvironment(globalEnv).Values()), nil
}

func logLevel(parent *Environment, jobVars map[string]string) (map[string]string, error) {
	level := strings.ToLower(strings.TrimSpace(jobVars[LogLevelEnvName]))

	if level == "" {
		level = strings.ToLower(strings.TrimSpace(parent.ValueOf(LogLevelEnvName)))
	}

	switch level {
	case "", "info":
		return map[string]string{LogLevelEnvName: "info"}, nil
	case "debug":
		return map[string]string{LogLevelEnvName: "debug"}, nil
	case "warn":
		return map[string]string{LogLevelEnvName: "warn"}, nil
	case "error":
		return map[string]string{LogLevelEnvName: "error"}, nil
	default:
		return nil, fmt.Errorf("log level: %s not supported", level)
	}
}
