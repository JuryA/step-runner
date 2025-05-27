package internal

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

type GetEnv func(key string) string

type Inputs struct {
	FromImageRef name.Reference
	ToImageRef   *RemoteImageRef
	LogLevel     slog.Level
}

func ParseInputs(args []string, getenv GetEnv) (*Inputs, error) {
	var fromImage, toRegistry, toRepository, toVersion string

	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.StringVar(&fromImage, "from-image", "", "")
	flags.StringVar(&toRegistry, "to-registry", "", "")
	flags.StringVar(&toRepository, "to-repository", "", "")
	flags.StringVar(&toVersion, "to-version", "", "")

	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}

	fromImageRef, err := parseFromImageRef(fromImage)
	if err != nil {
		return nil, fmt.Errorf("from image: %w", err)
	}

	toImageRef, err := ParseRemoteImageRef(toRegistry, toRepository, toVersion)
	if err != nil {
		return nil, fmt.Errorf("to image: %w", err)
	}

	logLevel, err := parseLogLevel(getenv("CI_STEPS_LOG_LEVEL"))
	if err != nil {
		return nil, fmt.Errorf("log level: %w", err)
	}

	inputs := &Inputs{
		FromImageRef: fromImageRef,
		ToImageRef:   toImageRef,
		LogLevel:     logLevel,
	}

	return inputs, nil
}

func parseFromImageRef(image string) (name.Reference, error) {
	image = strings.TrimSpace(image)

	if image == "" {
		return nil, errors.New("from image is required")
	}

	imageRef, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}

	if imageRef.Identifier() == "latest" && !strings.Contains(image, "latest") {
		return nil, errors.New("must specify tag or digest")
	}

	return imageRef, nil
}

func parseLogLevel(level string) (slog.Level, error) {
	var logLevel slog.Level
	err := logLevel.UnmarshalText([]byte(level))
	return logLevel, err
}
