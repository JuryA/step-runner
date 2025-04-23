package bldr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/dist-steps/oci/publish/internal"
)

type CLIInputsBuilder struct {
	t          *testing.T
	registry   string
	repository string
	tag        string
	common     string
	platforms  string
	logLevel   string
	outputFile string
}

func CLIInputs(t *testing.T) *CLIInputsBuilder {
	return &CLIInputsBuilder{
		t:          t,
		registry:   "registry.gitlab.com",
		repository: "my_group/my_project",
		tag:        "1.0.0",
		common:     `{"files": {"step.yml": "step.yml"}}`,
		platforms:  `{"linux/arm64": {"files": {"amd_run": "run"}}, "linux/amd64": {"files": {"arm_run": "run"}}}`,
		logLevel:   "info",
		outputFile: filepath.Join(t.TempDir(), "output.txt"),
	}
}

func (b *CLIInputsBuilder) WithRegistry(registry string) *CLIInputsBuilder {
	b.registry = registry
	return b
}

func (b *CLIInputsBuilder) WithRepository(repository string) *CLIInputsBuilder {
	b.repository = repository
	return b
}

func (b *CLIInputsBuilder) WithTag(tag string) *CLIInputsBuilder {
	b.tag = tag
	return b
}

func (b *CLIInputsBuilder) WithCommon(common string) *CLIInputsBuilder {
	b.common = common
	return b
}

func (b *CLIInputsBuilder) WithPlatforms(platforms string) *CLIInputsBuilder {
	b.platforms = platforms
	return b
}

func (b *CLIInputsBuilder) WithLogLevel(logLevel string) *CLIInputsBuilder {
	b.logLevel = logLevel
	return b
}

func (b *CLIInputsBuilder) WithOutputFile(outputFile string) *CLIInputsBuilder {
	b.outputFile = outputFile
	return b
}

func (b *CLIInputsBuilder) Build() ([]string, internal.GetEnv) {
	err := os.WriteFile(b.outputFile, []byte(""), 0644)
	require.NoError(b.t, err)

	cliOpts := []string{
		"--registry",
		b.registry,
		"--repository",
		b.repository,
		"--tag",
		b.tag,
		"--common",
		b.common,
		"--platforms",
		b.platforms,
		"--output_file",
		b.outputFile,
	}

	envOpts := map[string]string{
		"CI_STEPS_LOG_LEVEL": b.logLevel,
	}

	getEnv := func(key string) string { return envOpts[key] }
	return cliOpts, getEnv
}
