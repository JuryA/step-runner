package schema

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/step-runner/proto"
)

const distPrefix = "dist://"

type shortReference string

func (sr shortReference) compile() (*proto.Step_Reference, error) {
	if strings.HasPrefix(string(sr), ".") {
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
	path, filename := pathFilename(string(sr))
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
	path, filename := pathFilename(rest)
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

func pathFilename(pathStr string) (path []string, filename string) {
	filename = "step.yml"
	if pathStr == "" {
		return nil, filename
	}
	path = strings.Split(pathStr, "/")
	if strings.HasSuffix(path[len(path)-1], ".yml") {
		filename = path[len(path)-1]
		path = path[:len(path)-1]
	}
	return path, filename
}
