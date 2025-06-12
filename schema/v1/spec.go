package schema

import (
	"fmt"

	"gitlab.com/gitlab-org/step-runner/proto"
)

// Spec is a document describing the interface of a step.
type Spec struct {
	// Description is a human-readable description of the step.
	Description string `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description,omitempty"`

	// Spec corresponds to the JSON schema field "spec".
	Spec *Signature `json:"spec,omitempty" yaml:"spec,omitempty" mapstructure:"spec,omitempty"`
}

func (spec *Spec) Compile() (*proto.Spec, error) {
	protoSpec := &proto.Spec{Spec: &proto.Spec_Content{}}
	inputs := map[string]*proto.Spec_Content_Input{}
	if spec.Spec == nil {
		spec.Spec = &Signature{}
	}
	for k, v := range spec.Spec.Inputs {
		protoV, err := v.compile()
		if err != nil {
			return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
		}
		inputs[k] = protoV
	}
	protoSpec.Spec.Inputs = inputs
	outputs := map[string]*proto.Spec_Content_Output{}
	switch o := spec.Spec.Outputs.(type) {
	case string:
		protoSpec.Spec.OutputMethod = proto.OutputMethod_delegate
	case *Outputs:
		protoSpec.Spec.OutputMethod = proto.OutputMethod_outputs
		for k, v := range *o {
			protoV, err := v.compile()
			if err != nil {
				return nil, fmt.Errorf("compiling input[%q]: %v: %w", k, v, err)
			}
			outputs[k] = protoV
		}
	case nil:
		protoSpec.Spec.OutputMethod = proto.OutputMethod_outputs
	default:
		return nil, fmt.Errorf("unsupported type: %T", spec.Spec.Outputs)
	}
	protoSpec.Spec.Outputs = outputs
	return protoSpec, nil
}
