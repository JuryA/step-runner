package bldr

import (
	"gitlab.com/gitlab-org/step-runner/dist/steps/step/oci/promote/internal"
)

type CLIInputsBuilder struct {
	fromImage    string
	toRegistry   string
	toRepository string
	toVersion    string
	logLevel     string
}

func CLIInputs() *CLIInputsBuilder {
	return &CLIInputsBuilder{
		fromImage:    "registry.gitlab.com/my_group/my_project:1234",
		toRegistry:   "registry.gitlab.com",
		toRepository: "my_group/my_project",
		toVersion:    "1.0.0",
		logLevel:     "info",
	}
}

func (b *CLIInputsBuilder) WithFromImage(image string) *CLIInputsBuilder {
	b.fromImage = image
	return b
}

func (b *CLIInputsBuilder) WithToRegistry(registry string) *CLIInputsBuilder {
	b.toRegistry = registry
	return b
}

func (b *CLIInputsBuilder) WithToRepository(repository string) *CLIInputsBuilder {
	b.toRepository = repository
	return b
}

func (b *CLIInputsBuilder) WithToVersion(version string) *CLIInputsBuilder {
	b.toVersion = version
	return b
}

func (b *CLIInputsBuilder) WithLogLevel(logLevel string) *CLIInputsBuilder {
	b.logLevel = logLevel
	return b
}

func (b *CLIInputsBuilder) Build() ([]string, internal.GetEnv) {
	cliOpts := []string{
		"--from-image",
		b.fromImage,
		"--to-registry",
		b.toRegistry,
		"--to-repository",
		b.toRepository,
		"--to-version",
		b.toVersion,
	}

	envOpts := map[string]string{
		"CI_STEPS_LOG_LEVEL": b.logLevel,
	}

	getEnv := func(key string) string { return envOpts[key] }
	return cliOpts, getEnv
}
