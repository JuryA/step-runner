package schema

// Spec is a document describing the interface of a step.
type Spec struct {
	// Spec corresponds to the JSON schema field "spec".
	Spec *Signature `json:"spec,omitempty" yaml:"spec,omitempty" mapstructure:"spec,omitempty"`
}
