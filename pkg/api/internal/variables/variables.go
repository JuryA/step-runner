// package variables implements a type and functions to handle CI job variables as described in
// https://docs.gitlab.com/ee/ci/variables. This includes handling file-type variables and (eventually) masked
// variables. It is analogous to, but a subset/simplification of,
// https://gitlab.com/gitlab-org/gitlab-runner/-/blob/main/common/variables.go. This includes things like expanding
// variables and writing file-type variables to file.
package variables

import (
	"fmt"
	"os"
	"path"

	"gitlab.com/gitlab-org/step-runner/proto"
)

type Variable struct {
	v       *proto.Variable
	tmpPath string
}

func (v *Variable) File() bool   { return v.v.File }
func (v *Variable) Masked() bool { return v.v.Masked }
func (v *Variable) Key() string  { return v.v.Key }

// File type variables return the full path to the file instead of the value.
func (v *Variable) Value() string {
	if v.v.File {
		return path.Join(v.tmpPath, v.v.Key)
	} else {
		return v.v.Value
	}
}

func (v *Variable) Write() error {
	if !v.v.File {
		return fmt.Errorf("variable %q is not a file variable", v.v.Key)
	}
	return os.WriteFile(v.Value(), []byte(v.v.Value), 0600)
}

type Variables []Variable

func New(vars []*proto.Variable, tmpPath string) Variables {
	result := make(Variables, 0, len(vars))
	for _, v := range vars {
		result = append(result, Variable{v: v, tmpPath: tmpPath})
	}
	return result
}

func (vs *Variables) Write() error {
	for _, v := range *vs {
		if !v.File() {
			continue
		}
		if err := v.Write(); err != nil {
			return fmt.Errorf("writing file variable %q: %w", v.Key(), err)
		}
	}
	return nil
}

func Prepare(jobVariables []*proto.Variable, tmpPath string) (map[string]string, error) {
	if len(jobVariables) == 0 {
		return map[string]string{}, nil
	}
	outEnv := make(map[string]string, len(jobVariables))
	jobVars := New(jobVariables, tmpPath)
	for _, v := range jobVars {
		outEnv[v.Key()] = v.Value()
	}
	if err := jobVars.Write(); err != nil {
		return nil, fmt.Errorf("preparing variables: %w", err)
	}
	return outEnv, nil
}

func Expand(env map[string]string) map[string]string {
	expanded := make(map[string]string, len(env))
	for k, v := range env {
		expanded[k] = os.Expand(v, func(k string) string {
			return env[k]
		})
	}
	return expanded
}
