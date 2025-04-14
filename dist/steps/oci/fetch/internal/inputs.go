package internal

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

type GetEnv func(key string) string

type Inputs struct {
	RemoteImageRef name.Reference
	StepFilePath   string
	LogLevel       slog.Level
	OutputFile     string
}

func ParseInputs(args []string, getenv GetEnv) (*Inputs, error) {
	var registry, repository, tag, stepPath, stepFile, outputFile string

	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.StringVar(&registry, "registry", "", "")
	flags.StringVar(&repository, "repository", "", "")
	flags.StringVar(&stepPath, "step_path", "", "")
	flags.StringVar(&stepFile, "step_file", "", "")
	flags.StringVar(&tag, "tag", "", "")
	flags.StringVar(&outputFile, "output_file", "", "")

	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}

	registry = strings.TrimSpace(registry)
	repository = strings.TrimSpace(repository)
	tag = strings.TrimSpace(tag)

	if registry == "" {
		return nil, fmt.Errorf("registry is required")
	}

	if repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	if tag == "" {
		return nil, fmt.Errorf("tag is required")
	}

	if stepFile == "" {
		return nil, fmt.Errorf("step_file is required")
	}

	stepPath = strings.TrimSpace(stepPath)
	stepFile = strings.TrimSpace(stepFile)

	remoteImgRef, err := parseNamedReference(registry, repository, tag)
	if err != nil {
		return nil, fmt.Errorf("parsing image reference: %w", err)
	}

	logLevel, err := parseLogLevel(getenv("CI_STEPS_LOG_LEVEL"))
	if err != nil {
		return nil, fmt.Errorf("log level: %w", err)
	}

	if _, err := os.Stat(outputFile); err != nil {
		return nil, fmt.Errorf("output file is required: %w", err)
	}

	inputs := &Inputs{
		RemoteImageRef: remoteImgRef,
		StepFilePath:   filepath.Join(stepPath, stepFile),
		LogLevel:       logLevel,
		OutputFile:     outputFile,
	}

	return inputs, nil
}

func parseNamedReference(registry, repository, tag string) (name.Reference, error) {
	repository = path.Join(registry, repository)

	imgRefTag, tagErr := name.ParseReference(fmt.Sprintf("%s:%s", repository, tag))
	if tagErr == nil {
		return imgRefTag, nil
	}

	digestImgRef, digestErr := name.ParseReference(fmt.Sprintf("%s@%s", repository, tag))
	if digestErr == nil {
		return digestImgRef, nil
	}

	return nil, tagErr
}

func parseLogLevel(level string) (slog.Level, error) {
	var logLevel slog.Level
	err := logLevel.UnmarshalText([]byte(level))
	return logLevel, err
}
