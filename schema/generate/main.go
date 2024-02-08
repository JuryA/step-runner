package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"

	"gitlab.com/gitlab-org/step-runner/schema"
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
		// Need to avoid letting jsonschema handle Value
		// 	name:  "spec",
		// 	value: &schema.Spec{},
		// }, {
		name:  "definition",
		value: &schema.Definition{},
	}} {
		s := r.Reflect(t.value)
		out, err := json.MarshalIndent(s, "", "    ")
		if err != nil {
			panic(err)
		}
		os.WriteFile(fmt.Sprintf("schema/%v.json", t.name), out, 0640)
	}
}
