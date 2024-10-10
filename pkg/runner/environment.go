package runner

import (
	"fmt"
	"maps"
	"os"
	"strings"
	"sync"

	"golang.org/x/exp/slices"
)

// Environment represents environment variables. Environment is immutable.
// Environment variables are used in places such as the step export_file, step definitions with ENV, and OS environment.
// An environment can be added as "lexical scope", these values have higher precedence when looking up a variable.
type Environment struct {
	vars   map[string]string // Variables of this lexical scoped environment
	parent *Environment      // Variables of the parent lexical scoped environment

	// used to optimize retrieving values from the environment
	_getValuesOnce sync.Once
	_values        map[string]string
}

// NewEnvironmentFromOS returns the environment variables found in the OS runtime.
// Variables can be filtered by name, passing no names will return all variables.
func NewEnvironmentFromOS(names ...string) (*Environment, error) {
	vars := map[string]string{}

	for _, nameValue := range os.Environ() {
		name, value, ok := strings.Cut(nameValue, "=")

		if !ok {
			return nil, fmt.Errorf("failed to parse environment variable: %s", nameValue)
		}

		if len(names) > 0 && !slices.Contains(names, name) {
			continue
		}

		vars[name] = value
	}

	return NewEnvironment(vars), nil
}

func NewEnvironmentFromOSWithKnownVars() (*Environment, error) {
	return NewEnvironmentFromOS(
		"HTTPS_PROXY",
		"HTTP_PROXY",
		"LANG",
		"LC_ALL",
		"LC_CTYPE",
		"LOGNAME",
		"NO_PROXY",
		"PATH",
		"SHELL",
		"TERM",
		"TMPDIR",
		"TZ",
		"USER",
		"all_proxy",
		"http_proxy",
		"https_proxy",
		"no_proxy",
	)
}

func NewEmptyEnvironment() *Environment {
	return NewEnvironment(map[string]string{})
}

func NewEnvironment(vars map[string]string) *Environment {
	return &Environment{vars: vars, parent: nil}
}

func (e *Environment) AddLexicalScope(vars map[string]string) *Environment {
	if len(vars) == 0 {
		return e
	}

	return &Environment{vars: vars, parent: e}
}

func (e *Environment) Values() map[string]string {
	e._getValuesOnce.Do(func() {
		e._values = map[string]string{}

		if e.parent != nil {
			maps.Copy(e._values, e.parent.Values())
		}

		maps.Copy(e._values, e.vars)
	})

	return e._values
}

func (e *Environment) ValueOf(key string) string {
	return e.Values()[key]
}
