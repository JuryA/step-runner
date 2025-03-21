package pkg

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
)

var semVerRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(-.*)?$`)

type GetEnv func(key string) string

type Inputs struct {
	RemoteImageRef   *RemoteImageRef
	Common           Artifacts
	PlatformSpecific Artifacts
	LogLevel         slog.Level
}

func ParseInputs(args []string, getenv GetEnv) (*Inputs, error) {
	var registry, repository, tag, commonJSON, platformsJSON string

	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.StringVar(&registry, "registry", "", "")
	flags.StringVar(&repository, "repository", "", "")
	flags.StringVar(&tag, "tag", "", "")
	flags.StringVar(&commonJSON, "common", "", "")
	flags.StringVar(&platformsJSON, "platforms", "", "")

	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}

	remoteImgRef, err := parseRemoteImageRef(registry, repository, tag)
	if err != nil {
		return nil, fmt.Errorf("version: %w", err)
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

	inputs := &Inputs{
		RemoteImageRef:   remoteImgRef,
		Common:           common,
		PlatformSpecific: platform,
		LogLevel:         logLevel,
	}

	return inputs, nil
}

func parseRemoteImageRef(registry, repository, tag string) (*RemoteImageRef, error) {
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

	tagParts := semVerRe.FindStringSubmatch(tag)

	if len(tagParts) != 5 {
		return nil, fmt.Errorf("tag does not conform to semantic versioning major.minor.patch[-release]: %s", tag)
	}

	major, err := strconv.ParseUint(tagParts[1], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("major version %s: %w", tagParts[1], err)
	}

	minor, err := strconv.ParseUint(tagParts[2], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("minor version: %s: %w", tagParts[2], err)
	}

	patch, err := strconv.ParseUint(tagParts[3], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("patch version: %s: %w", tagParts[3], err)
	}

	release := tagParts[4]
	imgRef, err := name.ParseReference(fmt.Sprintf("%s:%d.%d.%d%s", path.Join(registry, repository), major, minor, patch, release))
	if err != nil {
		return nil, fmt.Errorf("parsing image reference: %w", err)
	}

	return NewRemoteImageRef(imgRef, major, minor, patch, release), nil
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
