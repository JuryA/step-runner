package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Project struct {
	projectID string
	jobToken  string
}

func NewProject(projectID string, jobToken string) *Project {
	return &Project{
		projectID: projectID,
		jobToken:  jobToken,
	}
}

// Release creates a tag if it is not present, and will fail if it has already been released
func (r Project) Release(ctx context.Context, commit, tagName, description string) error {
	data, err := json.Marshal(map[string]any{
		"name":        tagName,
		"tag_name":    tagName,
		"ref":         commit,
		"description": description,
	})

	if err != nil {
		return fmt.Errorf("creating payload: %w", err)
	}

	response, err := r.sendRequest(ctx, "POST", r.projectURL("/releases"), bytes.NewReader(data))

	if err != nil {
		return fmt.Errorf("releasing: %w", err)
	}

	if response.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(response.Body)
		return fmt.Errorf("request to release returned status code: %d, body: %s", response.StatusCode, string(responseBody))
	}

	return nil
}

func (r Project) projectURL(urlTemplate string, v ...any) string {
	baseURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s", r.projectID)
	return baseURL + fmt.Sprintf(urlTemplate, v...)
}

func (r Project) sendRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, url, body)

	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	request.Header.Set("JOB-TOKEN", r.jobToken)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)

	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return response, nil
}
