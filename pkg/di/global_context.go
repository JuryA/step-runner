package di

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/exp/slices"

	"gitlab.com/gitlab-org/step-runner/pkg/context"
)

func InitializeGlobalContext() func(*Container) error {
	return func(c *Container) error {
		globalCtx, err := context.NewGlobal()

		if err != nil {
			return fmt.Errorf("creating global context: %w", err)
		}

		c.CleanUpFns = append(c.CleanUpFns, globalCtx.Cleanup)

		// Step runner should have no concept of "CI_BUILDS_DIR".
		// However entire `ci` command is a workaround hack because
		// steps are not yet plumbed through runner. Once we receive
		// steps from runner over gRPC we will receive "work_dir"
		// explicitly (set to CI_BUILDS_DIR by runner). Then we can
		// delete this whole command.
		workDir := os.Getenv("CI_BUILDS_DIR")
		if workDir == "" {
			workDir, _ = os.Getwd()
		}
		globalCtx.WorkDir = workDir

		// Add all CI_, GITLAB_ and DOCKER_ environment variables as a
		// workaround until we get an explicit list in the Run gRPC
		// call.
		globalCtx.Job = map[string]string{}
		prefixes := []string{"CI_", "GITLAB_", "DOCKER_"}
		for _, e := range os.Environ() {
			k, v, ok := strings.Cut(e, "=")
			if !ok || !slices.ContainsFunc(prefixes, func(prefix string) bool {
				return strings.HasPrefix(k, prefix)
			}) {
				continue
			}
			globalCtx.Job[k] = v
		}

		c.GlobalCtx = globalCtx
		return nil
	}
}
