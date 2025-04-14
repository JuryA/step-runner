package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"gitlab.com/gitlab-org/step-runner/proto"
)

// Reference is a reference to a step in either a Git repository or an OCI image
type Reference struct {
	// Git corresponds to the JSON schema field "git".
	Git *GitReference `json:"git,omitempty" yaml:"git,omitempty" mapstructure:"git,omitempty"`

	// OCI corresponds to the JSON schema field "oci".
	OCI *OCIReference `json:"oci,omitempty" yaml:"oci,omitempty" mapstructure:"oci,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Reference) UnmarshalJSON(b []byte) error {
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

	*r = Reference(plain)
	return nil
}

func (r *Reference) compile(stepName string, inputs map[string]*structpb.Value, env map[string]string) (*proto.Step_Reference, error) {
	if r.Git == nil && r.OCI == nil {
		return nil, fmt.Errorf("compiling reference: git or oci not specified")
	}

	if r.Git != nil && r.OCI != nil {
		return nil, fmt.Errorf("compiling reference: git and oci specified")
	}

	if r.Git != nil {
		return r.compileGit()
	}

	return r.compileOCI(stepName, inputs, env)
}

func (r *Reference) compileGit() (*proto.Step_Reference, error) {
	url := defaultHTTPS(r.Git.Url)
	s := &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_git,
		Url:      url,
		Filename: "step.yml",
		Version:  r.Git.Rev,
	}
	if r.Git.Dir != nil {
		s.Path = strings.Split(*r.Git.Dir, "/")
	}
	if r.Git.File != nil {
		s.Filename = *r.Git.File
	}
	return s, nil
}

func (r *Reference) compileOCI(stepName string, inputs map[string]*structpb.Value, env map[string]string) (*proto.Step_Reference, error) {
	tag := r.OCI.Tag
	if tag == "" {
		tag = "latest"
	}

	fetchStepName := "fetch_step_" + stepName

	stepRef := &proto.Step_Reference{
		Protocol: proto.StepReferenceProtocol_spec_def,
		SpecDef: &proto.SpecDefinition{
			Spec: &proto.Spec{
				Spec: &proto.Spec_Content{
					OutputMethod: proto.OutputMethod_delegate,
				}},
			Definition: &proto.Definition{
				Type: proto.DefinitionType_steps,
				Steps: []*proto.Step{
					{
						Name: fetchStepName,
						Step: &proto.Step_Reference{
							Protocol: proto.StepReferenceProtocol_dist,
							Path:     []string{"oci", "fetch"},
							Filename: "step.yml",
						},
						Inputs: map[string]*structpb.Value{ // inline the inputs
							"registry":   structpb.NewStringValue(r.OCI.Registry),
							"repository": structpb.NewStringValue(r.OCI.Repository),
							"tag":        structpb.NewStringValue(tag),
						},
						Env: env,
					},
					{
						Name: stepName,
						Step: &proto.Step_Reference{
							Protocol: proto.StepReferenceProtocol_dynamic,
							Url:      fmt.Sprintf("${{steps.%s.outputs.download_dir}}", fetchStepName),
						},
						Inputs: inputs,
						Env:    env,
					},
				},
				Delegate: stepName,
			},
		},
	}

	return stepRef, nil
}
