package schema

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

func TestReferenceCustomMethods(t *testing.T) {
	cases := []struct {
		name          string
		json          string
		yaml          string
		wantRef       any
		wantSchemaErr bool
	}{{
		name:    "short reference",
		json:    `"gitlab.com/components/script@v1"`,
		yaml:    `gitlab.com/components/script@v1`,
		wantRef: "gitlab.com/components/script@v1",
	}, {
		name: "long simple git reference",
		json: `
{
  "git": {
    "url":"gitlab.com/components/script",
    "rev":"v1"
  }
}`,
		yaml: `
git:
  url:    gitlab.com/components/script
  rev: v1
`,
		wantRef: &Reference{
			Git: &GitReference{
				Url: stringRef("gitlab.com/components/script"),
				Rev: stringRef("v1"),
			},
		},
	}, {
		name: "long git reference with dir",
		json: `
{
  "git": {
    "url":"gitlab.com/components/script",
    "dir":"bash",
    "rev":"v1"
  }
}
`,
		yaml: `
git:
  url:    gitlab.com/components/script
  dir:    bash
  rev: v1
`,
		wantRef: &Reference{
			Git: &GitReference{
				Url: stringRef("gitlab.com/components/script"),
				Dir: stringRef("bash"),
				Rev: stringRef("v1"),
			},
		},
	}, {
		name: "long one-line git reference with dir",
		json: `{"git":{"url":"gitlab.com/components/script","dir":"bash","rev":"v1"}}`,
		yaml: `git: {url: gitlab.com/components/script, dir: bash, rev: v1}`,
		wantRef: &Reference{
			Git: &GitReference{
				Url: stringRef("gitlab.com/components/script"),
				Dir: stringRef("bash"),
				Rev: stringRef("v1"),
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
			step := Step{}
			switch v := tc.wantRef.(type) {
			case string:
				step.Step = v
				check(t, json.Marshal, json.Unmarshal, []byte(tc.json), v, stepsSchema, step)
				check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), v, stepsSchema, step)
			case *Reference:
				step.Step = v
				check(t, json.Marshal, json.Unmarshal, []byte(tc.json), v, stepsSchema, step)
				check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), v, stepsSchema, step)
			}
		})
	}
}
