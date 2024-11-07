package runner

import (
	"fmt"
	"maps"
	"os"
	"strings"
	"sync"

	"golang.org/x/exp/slices"
)

// Environment represents environment variables.
// Environment variables are used in places such as the step export_file, step definitions with ENV, and OS environment.
// Environment does not merge maps, and so does not lose information. Merging of environments occurs when values are retrieved.
// An environment can be added as "lexical scope", these values have higher precedence when looking up a variable.
// Mutations to the environment take precedence over initialized variables, most recent mutations have the highest precedence
type Environment struct {
	vars        map[string]string // Variables of this lexical scoped environment
	parent      *Environment      // Variables of the parent lexical scoped environment
	mutationsMu sync.Mutex
	mutations   []*Environment // Mutations to the environment
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
	return &Environment{vars: vars, parent: nil, mutations: nil}
}

// NewKVEnvironment creates an environment from a list of key values
func NewKVEnvironment(keyValues ...string) *Environment {
	if len(keyValues)%2 == 1 {
		panic("NewEnvironmentFromValues: odd argument count")
	}

	vars := make(map[string]string)

	for i := 0; i < len(keyValues); i += 2 {
		vars[keyValues[i]] = keyValues[i+1]
	}

	return NewEnvironment(vars)
}

func (e *Environment) AddLexicalScope(vars map[string]string) *Environment {
	if len(vars) == 0 {
		return e
	}

	return &Environment{vars: vars, parent: e, mutations: nil}
}

func (e *Environment) Len() int {
	return len(e.Values())
}

func (e *Environment) Values() map[string]string {
	e.mutationsMu.Lock()
	defer e.mutationsMu.Unlock()

	values := map[string]string{}

	if e.parent != nil {
		maps.Copy(values, e.parent.Values())
	}

	maps.Copy(values, e.vars)

	for _, mutation := range e.mutations {
		maps.Copy(values, mutation.Values())
	}

	return values
}

func (e *Environment) ValueOf(key string) string {
	return e.Values()[key]
}

func (e *Environment) Mutate(env *Environment) {
	if env == nil || env.Len() == 0 {
		return
	}

	e.mutationsMu.Lock()
	defer e.mutationsMu.Unlock()

	e.mutations = append(e.mutations, env)
}
