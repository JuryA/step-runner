package internal

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type GetEnv func(key string) string

type Inputs struct {
	ImageRef         name.Reference
	Common           Artifacts
	PlatformSpecific Artifacts
	LogLevel         slog.Level
	OutputFile       string
}

func ParseInputs(args []string, getenv GetEnv) (*Inputs, error) {
	var registry, repository, tag, commonJSON, platformsJSON, outputFile string

	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.StringVar(&registry, "registry", "", "")
	flags.StringVar(&repository, "repository", "", "")
	flags.StringVar(&tag, "tag", "", "")
	flags.StringVar(&commonJSON, "common", "", "")
	flags.StringVar(&platformsJSON, "platforms", "", "")
	flags.StringVar(&outputFile, "output_file", "", "")

	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}

	imageRef, err := parseImageRef(registry, repository, tag)
	if err != nil {
		return nil, fmt.Errorf("image ref: %w", err)
	}

	common, err := parseCommon(commonJSON)
	if err != nil {
		return nil, fmt.Errorf("common input: %w", err)
	}

	platform, err := parsePlatforms(platformsJSON)
	if err != nil {
		return nil, fmt.Errorf("platforms input: %w", err)
	}

	logLevel, err := parseLogLevel(getenv("CI_STEPS_LOG_LEVEL"))
	if err != nil {
		return nil, fmt.Errorf("log level: %w", err)
	}

	if _, err := os.Stat(outputFile); err != nil {
		return nil, fmt.Errorf("output file is required: %w", err)
	}

	inputs := &Inputs{
		ImageRef:         imageRef,
		Common:           common,
		PlatformSpecific: platform,
		LogLevel:         logLevel,
		OutputFile:       outputFile,
	}

	return inputs, nil
}

func parseImageRef(registry, repository, tag string) (name.Reference, error) {
	registry = strings.TrimSpace(registry)
	repository = strings.TrimSpace(repository)
	tag = strings.TrimSpace(tag)

	if registry == "" {
		return nil, errors.New("registry is required")
	}

	if repository == "" {
		return nil, errors.New("repository is required")
	}

	if tag == "" {
		return nil, errors.New("tag is required")
	}

	imageRef, err := name.ParseReference(fmt.Sprintf("%s:%s", path.Join(registry, repository), tag))
	if err != nil {
		return nil, fmt.Errorf("parsing image reference: %w", err)
	}

	return imageRef, nil
}

func parsePlatforms(platformsJSON string) (Artifacts, error) {
	var parsed map[string]struct {
		OSVersion  string            `json:"os.version"`
		OSFeatures []string          `json:"os.features"`
		Variant    string            `json:"variant"`
		Features   []string          `json:"features"`
		Files      map[string]string `json:"files"`
	}

	if err := unmarshal(platformsJSON, &parsed); err != nil {
		return nil, err
	}

	if len(parsed) == 0 {
		return nil, errors.New("must have at least one platform")
	}

	allArtifacts := NewArtifacts()

	for _, platformName := range slices.Sorted(maps.Keys(parsed)) {
		nameParts := strings.Split(platformName, "/")

		if len(nameParts) != 2 {
			return nil, fmt.Errorf("invalid platform os/arch: %s", platformName)
		}

		platform := &v1.Platform{
			OS:           strings.TrimSpace(nameParts[0]),
			Architecture: strings.TrimSpace(nameParts[1]),
			OSVersion:    strings.TrimSpace(parsed[platformName].OSVersion),
			OSFeatures:   trimSpaceInStrings(parsed[platformName].OSFeatures),
			Variant:      strings.TrimSpace(parsed[platformName].Variant),
			Features:     trimSpaceInStrings(parsed[platformName].Features),
		}

		if len(allArtifacts.ForPlatform(platform)) > 0 {
			return nil, fmt.Errorf(`platform "%s/%s" defined more than once`, platform.OS, platform.Architecture)
		}

		artifacts, err := buildArtifacts(platform, parsed[platformName].Files)
		if err != nil {
			return nil, fmt.Errorf(": %w", err)
		}

		allArtifacts = allArtifacts.Add(artifacts)
	}

	return allArtifacts, nil
}

func parseCommon(filesJSON string) (Artifacts, error) {
	var parsed struct {
		Files map[string]string `json:"files"`
	}

	if err := unmarshal(filesJSON, &parsed); err != nil {
		return nil, err
	}

	return buildArtifacts(PlatformGeneric, parsed.Files)
}

func unmarshal(jsonInput string, into any) error {
	decoder := json.NewDecoder(strings.NewReader(jsonInput))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(into)

	if errors.Is(err, io.ErrUnexpectedEOF) {
		return errors.New("unexpected end of JSON input")
	}

	return err
}

func buildArtifacts(platform *v1.Platform, srcDst map[string]string) (Artifacts, error) {
	artifacts := make([]*Artifact, 0, len(srcDst))

	for _, srcPath := range slices.Sorted(maps.Keys(srcDst)) {
		src := strings.TrimSpace(srcPath)
		dst := strings.TrimSpace(srcDst[srcPath])

		if src == "" {
			return nil, fmt.Errorf("empty source path: %q: %q", srcPath, srcDst[srcPath])
		}

		if dst == "" {
			return nil, fmt.Errorf("empty destination path: %q: %q", srcPath, srcDst[srcPath])
		}

		artifacts = append(artifacts, NewArtifact(platform, src, dst))
	}

	return NewArtifacts(artifacts...), nil
}

func parseLogLevel(level string) (slog.Level, error) {
	var logLevel slog.Level
	err := logLevel.UnmarshalText([]byte(level))
	return logLevel, err
}

func trimSpaceInStrings(values []string) []string {
	trimmed := make([]string, 0, len(values))

	for _, value := range values {
		trimmed = append(trimmed, strings.TrimSpace(value))
	}

	return trimmed
}
