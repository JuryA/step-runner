package pkg

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"gitlab.com/gitlab-org/step-runner/pkg/cache/oci"
)

var semVerRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-.*)?`)

type Inputs struct {
	Registry         string
	Repository       string
	Tag              string
	Common           *oci.Artifacts
	PlatformSpecific *oci.Artifacts
}

func (i *Inputs) Validate() error {
	if i.Registry == "" {
		return errors.New("registry is required")
	}

	if i.Repository == "" {
		return errors.New("repository is required")
	}

	if i.Tag == "" {
		return errors.New("tag is required")
	}

	if matches := semVerRe.MatchString(i.Tag); !matches {
		return fmt.Errorf("tag input: %q does not conform to semantic versioning MAJOR.MINOR.PATCH[-release]", i.Tag)
	}

	return nil
}

func ParseInputs(args []string) (*Inputs, error) {
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

	common, err := parseFiles(oci.PlatformGeneric, commonJSON)
	if err != nil {
		return nil, fmt.Errorf("common input: %w", err)
	}

	platform, err := parsePlatforms(platformsJSON)
	if err != nil {
		return nil, fmt.Errorf("platforms input: %w", err)
	}

	inputs := &Inputs{
		Registry:         strings.TrimSpace(registry),
		Repository:       strings.TrimSpace(repository),
		Tag:              strings.TrimSpace(tag),
		Common:           common,
		PlatformSpecific: platform,
	}

	if err := inputs.Validate(); err != nil {
		return nil, err
	}

	return inputs, nil
}

func parsePlatforms(platformsJSON string) (*oci.Artifacts, error) {
	var parsed map[string]struct {
		Files map[string]string `json:"files"`
	}

	if err := unmarshal(platformsJSON, &parsed); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(parsed))
	for name := range parsed {
		names = append(names, name)
	}

	sort.Strings(names)

	allArtifacts := oci.NewArtifacts()

	for _, name := range names {
		nameParts := strings.Split(name, "/")

		if len(nameParts) != 2 {
			return nil, fmt.Errorf("invalid platform os/arch: %s", name)
		}

		platform := &v1.Platform{
			OS:           strings.TrimSpace(nameParts[0]),
			Architecture: strings.TrimSpace(nameParts[1]),
			OSVersion:    "",
			OSFeatures:   nil,
			Variant:      "",
			Features:     nil,
		}

		artifacts, err := buildArtifacts(platform, parsed[name].Files)
		if err != nil {
			return nil, fmt.Errorf(": %w", err)
		}

		allArtifacts = allArtifacts.Add(artifacts)
	}

	if allArtifacts.Len() == 0 {
		return nil, errors.New("must have at least one platform")
	}

	return allArtifacts, nil
}

func parseFiles(platform *v1.Platform, filesJSON string) (*oci.Artifacts, error) {
	var parsed struct {
		FromTo map[string]string `json:"files"`
	}

	if err := unmarshal(filesJSON, &parsed); err != nil {
		return nil, err
	}

	return buildArtifacts(platform, parsed.FromTo)
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

func buildArtifacts(platform *v1.Platform, parsed map[string]string) (*oci.Artifacts, error) {
	fromPaths := make([]string, 0, len(parsed))
	for fromPath := range parsed {
		fromPaths = append(fromPaths, fromPath)
	}

	sort.Strings(fromPaths)

	artifacts := make([]*oci.Artifact, 0, len(fromPaths))

	for _, fromPath := range fromPaths {
		from := strings.TrimSpace(fromPath)
		to := strings.TrimSpace(parsed[fromPath])

		if from == "" {
			return nil, fmt.Errorf("empty from path: %q: %q", fromPath, parsed[fromPath])
		}

		if to == "" {
			return nil, fmt.Errorf("empty to path: %q: %q", fromPath, parsed[fromPath])
		}

		artifacts = append(artifacts, oci.NewArtifact(platform, from, to))
	}

	return oci.NewArtifacts(artifacts...), nil
}
