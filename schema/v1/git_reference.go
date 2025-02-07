package schema

import (
	"encoding/json"
	"fmt"
)

// GitReference is a reference to a step in a Git repository containing the full
// set of configuration options.
type GitReference struct {
	// Dir corresponds to the JSON schema field "dir".
	Dir *string `json:"dir,omitempty" yaml:"dir,omitempty" mapstructure:"dir,omitempty"`

	// Rev corresponds to the JSON schema field "rev".
	Rev string `json:"rev" yaml:"rev" mapstructure:"rev"`

	// Url corresponds to the JSON schema field "url".
	Url string `json:"url" yaml:"url" mapstructure:"url"`

	// File corresponds to the JSON schema field "file".
	File *string `json:"file,omitempty" yaml:"file,omitempty" mapstructure:"file,omitempty"`
}

func NewGitReference(url, rev string, options ...func(*GitReference)) *GitReference {
	gitReference := &GitReference{
		Url: url,
		Rev: rev,
	}

	for _, opt := range options {
		opt(gitReference)
	}

	return gitReference
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *GitReference) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	if _, ok := raw["rev"]; raw != nil && !ok {
		return fmt.Errorf("field rev in git: required")
	}

	if _, ok := raw["url"]; raw != nil && !ok {
		return fmt.Errorf("field url in git: required")
	}

	type Plain GitReference
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}

	*j = GitReference(plain)
	return nil
}

func GitRefDir(dir string) func(*GitReference) {
	return func(j *GitReference) {
		j.Dir = &dir
	}
}
