package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"

	schema "gitlab.com/gitlab-org/step-runner/schema/v1"
)

func main() {
	r := jsonschema.Reflector{
		DoNotReference: true,
	}
	if err := r.AddGoComments("gitlab.com/gitlab-org/step-runner", "./"); err != nil {
		panic(err)
	}
	for _, t := range []struct {
		name  string
		value any
	}{{
		name:  "spec",
		value: &schema.Spec{},
	}, {
		name:  "definition",
		value: &schema.Definition{},
	}, {
		name:  "steps",
		value: schema.Steps{},
	}} {
		s := r.Reflect(t.value)
		out, err := json.MarshalIndent(s, "", "    ")
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(fmt.Sprintf("schema/v1/%v.json", t.name), out, 0640)
		if err != nil {
			panic(err)
		}
	}
}
