package main

import (
	"log"
	"os"

	"gitlab.com/gitlab-org/step-runner/builtin/steps/oci/publish/pkg"
)

func main() {
	_, err := pkg.ParseInputs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}
