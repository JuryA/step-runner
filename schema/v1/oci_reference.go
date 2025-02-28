package schema

import (
	"encoding/json"
	"fmt"
)

// OCIReference is a reference to a step in an OCI image that is hosted in an OCI repository.
type OCIReference struct {
	// Registry corresponds to the JSON schema field "registry".
	Registry string `json:"registry" yaml:"registry" mapstructure:"registry"`

	// Repository corresponds to the JSON schema field "repository".
	Repository string `json:"repository" yaml:"repository" mapstructure:"repository"`

	// Tag corresponds to the JSON schema field "tag".
	Tag string `json:"tag" yaml:"tag" mapstructure:"tag"`

	// Dir corresponds to the JSON schema field "dir".
	Dir *string `json:"dir,omitempty" yaml:"dir,omitempty" mapstructure:"dir,omitempty"`

	// File corresponds to the JSON schema field "file".
	File *string `json:"file,omitempty" yaml:"file,omitempty" mapstructure:"file,omitempty"`
}

func NewOCIReference(registry, repository, tag string) *OCIReference {
	return &OCIReference{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OCIReference) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	if _, ok := raw["tag"]; raw != nil && !ok {
		return fmt.Errorf("field tag in oci: required")
	}

	if _, ok := raw["registry"]; raw != nil && !ok {
		return fmt.Errorf("field registry in oci: required")
	}

	if _, ok := raw["repository"]; raw != nil && !ok {
		return fmt.Errorf("field repository in oci: required")
	}

	type Plain OCIReference
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}

	*j = OCIReference(plain)
	return nil
}
