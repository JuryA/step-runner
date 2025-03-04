package main

import (
	"log"
	"os"

	"gitlab.com/gitlab-org/step-runner/builtin/steps/oci/publish/pkg"
)

func main() {
	inputs, err := pkg.ParseInputs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	_, err = inputs.ImgRef()
	if err != nil {
		log.Fatal(err)
	}
}
