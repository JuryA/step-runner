package schema

import (
	"encoding/json"
	"fmt"
)

// OCIReference is a reference to a step in an OCI image that is hosted in an OCI repository.
type OCIReference struct {
	// Url corresponds to the JSON schema field "url".
	Url string `json:"url" yaml:"url" mapstructure:"url"`

	// Tag corresponds to the JSON schema field "tag".
	Tag string `json:"tag" yaml:"tag" mapstructure:"tag"`

	// Dir corresponds to the JSON schema field "dir".
	Dir string `json:"dir" yaml:"dir" mapstructure:"dir"`

	// File corresponds to the JSON schema field "file".
	File string `json:"file,omitempty" yaml:"file,omitempty" mapstructure:"file,omitempty"`
}

func NewOCIReference(url, tag string) *OCIReference {
	return &OCIReference{
		Url: url,
		Tag: tag,
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

	if _, ok := raw["url"]; raw != nil && !ok {
		return fmt.Errorf("field url in oci: required")
	}

	type Plain OCIReference
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}

	*j = OCIReference(plain)
	return nil
}
