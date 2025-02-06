package schema

import (
	"encoding/json"
	"fmt"
)

// Reference is a reference to a step in either a Git repository or an OCI image
type Reference struct {
	// Git corresponds to the JSON schema field "git".
	Git *GitReference `json:"git,omitempty" yaml:"git,omitempty" mapstructure:"git,omitempty"`

	// OCI corresponds to the JSON schema field "oci".
	OCI *OCIReference `json:"oci,omitempty" yaml:"oci,omitempty" mapstructure:"oci,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Reference) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	_, gitDefined := raw["git"]
	_, ociDefined := raw["oci"]

	if gitDefined && ociDefined {
		return fmt.Errorf("cannot use both git: and oci: fields, please specify only one step location")
	}

	if !gitDefined && !ociDefined {
		return fmt.Errorf("field git: or oci: required")
	}

	type Plain Reference
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Reference(plain)
	return nil
}
