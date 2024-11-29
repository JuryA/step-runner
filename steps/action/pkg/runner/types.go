package runner

import (
	"github.com/nektos/act/pkg/model"
)

func newWorkflow(
	action string,
	inputs map[string]string,
	expectedOutputs map[string]model.Output,
) *workflow {
	outputs := map[string]string{}
	for k := range expectedOutputs {
		v := "${{ steps.single-step.outputs." + k + " }}"
		outputs[k] = v
	}
	return &workflow{
		Jobs: map[string]job{
			"single-action": job{
				RunsOn: "act-image",
				Steps: []step{{
					Id:   "single-step",
					Uses: action,
					With: inputs,
				}},
				Outputs: outputs,
			},
		},
	}
}

type workflow struct {
	Jobs map[string]job `yaml:"jobs"`
}

type job struct {
	RunsOn  string            `yaml:"runs-on"`
	Steps   []step            `yaml:"steps"`
	Outputs map[string]string `yaml:"outputs"`
}

type step struct {
	Id   string            `yaml:"id"`
	Uses string            `yaml:"uses"`
	With map[string]string `yaml:"with"`
}
