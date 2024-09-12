package runner_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/step-runner/pkg/runner"
	"gitlab.com/gitlab-org/step-runner/pkg/testutil/bldr"
)

func TestExecutableStep_Describe(t *testing.T) {
	protoDef := bldr.ProtoDef().WithExecType("", []string{"go", "run", "."}).Build()
	specDef := bldr.ProtoSpecDef().WithDefinition(protoDef).Build()

	step := runner.NewExecutableStep(runner.StepDefinedInGitLabJob, &runner.Params{}, specDef)
	require.Equal(t, `executable step "go run ."`, step.Describe())
}
