package runner_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	stepdist "gitlab.com/gitlab-org/step-runner/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/cache/dist"
	"gitlab.com/gitlab-org/step-runner/pkg/runner"
)

func TestDistStepResource_Fetch(t *testing.T) {
	fetcher := dist.NewFetcher(stepdist.FindDistributedStep)
	res := runner.NewDistStepResource(fetcher, "step/oci/build", "step.yml")
	specDef, err := res.Fetch(context.Background(), nil)
	require.NoError(t, err)
	require.Contains(t, strings.Join(specDef.ToProto().Definition.Exec.Command, " "), "run")
}
