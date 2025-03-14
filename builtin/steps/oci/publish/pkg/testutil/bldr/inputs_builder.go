package bldr

import "gitlab.com/gitlab-org/step-runner/builtin/steps/oci/publish/pkg"

type CLIInputsBuilder struct {
	registry   string
	repository string
	tag        string
	common     string
	platforms  string
	logLevel   string
}

func CLIInputs() *CLIInputsBuilder {
	return &CLIInputsBuilder{
		registry:   "registry.gitlab.com",
		repository: "my_group/my_project",
		tag:        "1.0.0",
		common:     `{"files": {"step.yml": "step.yml"}}`,
		platforms:  `{"linux/arm64": {"files": {"amd_run": "run"}}, "linux/amd64": {"files": {"arm_run": "run"}}}`,
		logLevel:   "info",
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

func (b *CLIInputsBuilder) Build() ([]string, pkg.GetEnv) {
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
	}

	envOpts := map[string]string{
		"CI_STEPS_LOG_LEVEL": b.logLevel,
	}

	getEnv := func(key string) string { return envOpts[key] }
	return cliOpts, getEnv
}
