package schema

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/require"
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

	data, err := os.ReadFile("../../schema/v1/steps.json")
	require.NoError(t, err)
	stepsSchema := jsonschema.MustCompileString("steps.json", string(data))

	check := func(
		t *testing.T,
		marshal func(any) ([]byte, error),
		unmarshal func([]byte, any) error,
		data []byte,
		wantRef Reference,
	) {
		// Unmarshal
		ref := Reference{}
		err := unmarshal(data, &ref)
		require.NoError(t, err)
		require.Equal(t, wantRef, ref)

		// Marshal
		roundTripData, err := marshal(ref)
		require.NoError(t, err)

		// Unmarshal
		roundTripRef := Reference{}
		err = unmarshal(roundTripData, &roundTripRef)
		require.NoError(t, err)
		require.Equal(t, wantRef, roundTripRef)

		// Validate reference with steps schema
		steps := []Step{{
			Step: wantRef,
		}}
		stepsData, err := marshal(steps)
		require.NoError(t, err)
		var untyped any
		err = unmarshal(stepsData, &untyped)
		require.NoError(t, err)
		err = stepsSchema.Validate(untyped)
		require.NoError(t, err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check(t, json.Marshal, json.Unmarshal, []byte(tc.json), tc.wantRef)
			check(t, yaml.Marshal, yaml.Unmarshal, []byte(tc.yaml), tc.wantRef)
		})
	}
}
