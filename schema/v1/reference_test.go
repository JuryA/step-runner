package schema

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

func TestReferenceCustomMethods(t *testing.T) {

	myStep := "my_step"

	cases := []struct {
		name          string
		json          string
		yaml          string
		wantRef       any
		wantSchemaErr bool
	}{{
		name:    "short reference",
		json:    `{"name":"my_step","step":"gitlab.com/components/script@v1"}`,
		yaml:    `{name: my_step, step: gitlab.com/components/script@v1}`,
		wantRef: "gitlab.com/components/script@v1",
	}, {
		name: "long simple git reference",
		json: `
{
  "name": "my_step",
  "step": {
    "git": {
      "url":"gitlab.com/components/script",
      "rev":"v1"
    }
  }
}
`,
		yaml: `
name: my_step
step:
  git:
    url:    gitlab.com/components/script
    rev: v1
`,
		wantRef: &Reference{
			Git: GitReference{
				Url: "gitlab.com/components/script",
				Rev: "v1",
			},
		},
	}, {
		name: "long git reference with dir",
		json: `
{
  "name": "my_step",
  "step": {
    "git": {
      "url":"gitlab.com/components/script",
      "dir":"bash",
      "rev":"v1"
    }
  }
}
`,
		yaml: `
name: my_step
step:
  git:
    url:    gitlab.com/components/script
    dir:    bash
    rev: v1
`,
		wantRef: &Reference{
			Git: GitReference{
				Url: "gitlab.com/components/script",
				Dir: stringRef("bash"),
				Rev: "v1",
			},
		},
	}, {
		name: "long one-line git reference with dir",
		json: `{"name":"my_step","step":{"git":{"url":"gitlab.com/components/script","dir":"bash","rev":"v1"}}}`,
		yaml: `{name: my_step, step: {git: {url: gitlab.com/components/script, dir: bash, rev: v1}}}`,
		wantRef: &Reference{
			Git: GitReference{
				Url: "gitlab.com/components/script",
				Dir: stringRef("bash"),
				Rev: "v1",
			},
		},
	}}

	data, err := os.ReadFile("step.json")
	if err != nil {
		panic(err)
	}
	stepsSchema := jsonschema.MustCompileString("step.json", string(data))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			step := Step{
				Name: &myStep,
				Step: tc.wantRef,
			}
			switch v := tc.wantRef.(type) {
			case string:
				step.Step = v
				check(t, json.Marshal, json.Unmarshal, []byte(tc.json), step, stepsSchema)
				check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), step, stepsSchema)
			case *Reference:
				step.Step = v
				check(t, json.Marshal, json.Unmarshal, []byte(tc.json), step, stepsSchema)
				check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), step, stepsSchema)
			}
		})
	}
}
