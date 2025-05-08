package schema

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"gitlab.com/gitlab-org/step-runner/proto"
)

const distPrefix = "dist://"

var containsExpressionRe = regexp.MustCompile(`\${{.*}}`)

func CompileShortRef(value string) (*proto.Step_Reference, error) {
	return shortReference(value).compile()
}

type shortReference string

func (sr shortReference) compile() (*proto.Step_Reference, error) {
	value := strings.TrimSpace(string(sr))

	if containsExpressionRe.MatchString(string(sr)) {
		return &proto.Step_Reference{
			Protocol: proto.StepReferenceProtocol_dynamic,
			Url:      string(sr),
		}, nil
	}

	if strings.HasPrefix(value, ".") || strings.HasPrefix(value, "/") {
		return sr.compileLocal()
	}

	if strings.HasPrefix(string(sr), distPrefix) {
		return sr.compileDist()
	}

	return sr.compileRemote()
}

func (sr shortReference) compileDist() (*proto.Step_Reference, error) {
	stepDir := strings.TrimPrefix(string(sr), distPrefix)

	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_dist,
		Path:     strings.Split(stepDir, "/"),
		Filename: "step.yml",
	}, nil
}

func (sr shortReference) compileLocal() (*proto.Step_Reference, error) {
	path, filename := pathFilename(true, string(sr))
	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_local,
		Path:     path,
		Filename: filename,
	}, nil
}

func (sr shortReference) compileRemote() (*proto.Step_Reference, error) {
	parts := strings.Split(string(sr), "@")
	if len(parts) < 2 {
		return nil, fmt.Errorf("expecting url@rev. got %q", sr)
	}
	rest := strings.Join(parts[0:len(parts)-1], "@")
	rev := parts[len(parts)-1]
	url, rest, _ := strings.Cut(rest, "/-/")
	url = defaultHTTPS(url)
	path, filename := pathFilename(false, rest)

	// Check if the path contains "internal" segment
	if hasInternalPathSegment(path) {
		return nil, fmt.Errorf("steps inside folders named 'internal' cannot be accessed directly from external repositories")
	}

	path = append([]string{"steps"}, path...)
	return &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_git,
		Url:      url,
		Version:  rev,
		Path:     path,
		Filename: filename,
	}, nil
}

func defaultHTTPS(stepUrl string) string {
	if strings.HasPrefix(stepUrl, "http://") || strings.HasPrefix(stepUrl, "https://") {
		return stepUrl
	}
	return "https://" + stepUrl
}

func pathFilename(allowAbsolute bool, pathStr string) ([]string, string) {
	filename := "step.yml"
	if pathStr == "" {
		return nil, filename
	}

	path := strings.Split(pathStr, "/")
	lastItemIndex := len(path) - 1

	if strings.HasSuffix(path[lastItemIndex], ".yml") {
		filename = path[lastItemIndex]
		path = path[:lastItemIndex]
	}

	if allowAbsolute && len(path) > 0 && strings.HasPrefix(pathStr, "/") {
		// for absolute paths, strings.Split results in the first element of path being empty string
		path[0] = "/"
	}

	path = slices.DeleteFunc(path, func(value string) bool { return value == "" })
	return path, filename
}
