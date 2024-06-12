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
		wantRef       Reference
		wantSchemaErr bool
	}{{
		name: "short reference",
		json: `"gitlab.com/components/script@v1"`,
		yaml: `gitlab.com/components/script@v1`,
		wantRef: Reference{
			Short: "gitlab.com/components/script@v1",
		},
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
		wantRef: Reference{
			Git: GitReference{
				Url: "gitlab.com/components/script",
				Rev: "v1",
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
		wantRef: Reference{
			Git: GitReference{
				Url: "gitlab.com/components/script",
				Dir: "bash",
				Rev: "v1",
			},
		},
	}, {
		name: "long one-line git reference with dir",
		json: `{"git":{"url":"gitlab.com/components/script","dir":"bash","rev":"v1"}}`,
		yaml: `git: {url: gitlab.com/components/script, dir: bash, rev: v1}`,
		wantRef: Reference{
			Git: GitReference{
				Url: "gitlab.com/components/script",
				Dir: "bash",
				Rev: "v1",
			},
		},
	}}

	data, err := os.ReadFile("steps.json")
	if err != nil {
		panic(err)
	}
	stepsSchema := jsonschema.MustCompileString("steps.json", string(data))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := []Step{{
				Step: tc.wantRef,
			}}
			check(t, json.Marshal, json.Unmarshal, []byte(tc.json), tc.wantRef, stepsSchema, steps)
			check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), tc.wantRef, stepsSchema, steps)
		})
	}
}
