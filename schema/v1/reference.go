package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/invopop/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

// Reference is a reference to a step. References can contain either a
// short or a full form.
type Reference struct {
	// Short is a compact step reference which supports only a
	// subset of the possible configuration.
	Short string `json:"-" yaml:"-"`
	// Git a reference to a step in a Git repository.
	Git GitReference `json:"-" yaml:"-"`
}

var (
	_ yaml.Unmarshaler = &Reference{}
	_ yaml.Marshaler   = &Reference{}
	_ json.Unmarshaler = &Reference{}
	_ json.Marshaler   = &Reference{}
)

// GitReference is a reference to a step in a Git repository
// containing the full set of configuration options.
type GitReference struct {
	// Url is the location of the Git repo containing the step
	// definition.
	Url string `json:"url" yaml:"url"`
	// Dir is the relative path to the step definition within the
	// Git repo.
	Dir string `json:"dir" yaml:"dir"`
	// Rev is the step version to use.
	Rev string `json:"rev" yaml:"rev"`
}

func (r Reference) IsEmpty() bool {
	return r.Short == "" && r.Git.IsEmpty()
}

func (g GitReference) IsEmpty() bool {
	return strings.TrimSpace(g.Url) == "" && strings.TrimSpace(g.Dir) == "" && strings.TrimSpace(g.Rev) == ""
}

func (r *Reference) UnmarshalYAML(value *yaml.Node) error {
	switch value.Tag {
	case "!!str":
		return value.Decode(&r.Short)
	case "!!map":
		ref := map[string]GitReference{}
		err := value.Decode(ref)
		if err != nil {
			return err
		}
		if g, ok := ref["git"]; ok {
			r.Git = g
			return nil
		}
		return fmt.Errorf("missing keyword 'git': %v", maps.Keys(ref))
	default:
		return fmt.Errorf("unsupported reference type: %q", value.Tag)
	}
}

func (r Reference) MarshalYAML() (any, error) {
	switch {
	case r.Short != "":
		return r.Short, nil
	case !r.Git.IsEmpty():
		return map[string]GitReference{
			"git": r.Git,
		}, nil
	default:
		return nil, fmt.Errorf("unhandled reference type: %v", r)
	}
}

func (r *Reference) UnmarshalJSON(data []byte) error {
	var untyped any
	err := json.Unmarshal(data, &untyped)
	if err != nil {
		return err
	}
	switch t := untyped.(type) {
	case string:
		r.Short = t
		return nil
	case map[string]any:
		ref := map[string]GitReference{}
		err := json.Unmarshal(data, &ref)
		if err != nil {
			return err
		}
		if g, ok := ref["git"]; ok {
			r.Git = g
			return nil
		}
		return fmt.Errorf("missing keyword 'git': %v", maps.Keys(ref))
	default:
		return fmt.Errorf("unsupported type: %T", untyped)
	}
}

func (r Reference) MarshalJSON() ([]byte, error) {
	switch {
	case r.Short != "":
		return json.Marshal(r.Short)
	case !r.Git.IsEmpty():
		return json.Marshal(map[string]GitReference{
			"git": r.Git,
		})
	default:
		return nil, fmt.Errorf("unhandled reference type: %v", r)
	}
}

func (r Reference) JSONSchema() *jsonschema.Schema {
	jsr := jsonschema.Reflector{
		DoNotReference: true,
	}
	if err := jsr.AddGoComments("gitlab.com/gitlab-org/step-runner", "./"); err != nil {
		panic(err)
	}
	properties := orderedmap.New[string, *jsonschema.Schema](1)
	properties.Set("git", jsr.Reflect(GitReference{}))
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{{
			Type: "string",
		}, {
			Type:       "object",
			Properties: properties,
		}},
	}
}
