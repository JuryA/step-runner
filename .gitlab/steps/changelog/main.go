package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"gitlab.com/gitlab-org/step-runner/.gitlab/steps/changelog/pkg/changelog"
)

var changelogPath = flag.String("changelog", "", "")
var outputFilePath = flag.String("output_file", "", "")
var echoLatestVersion = flag.Bool("echo_latest", false, "")

func main() {
	flag.Parse()

	if changelogPath == nil || *changelogPath == "" {
		log.Fatalln("changelog is required, aborting")
	}

	if outputFilePath == nil || *outputFilePath == "" {
		log.Fatalln("output_file is required, aborting")
	}

	contents, err := os.ReadFile(*changelogPath)
	if err != nil {
		log.Fatalf("failed to read changelog %s: %s\n", *changelogPath, err.Error())
	}

	version, err := changelog.New(contents).LatestVersion()
	if err != nil {
		log.Fatalf("failed to determine version: %s\n", err.Error())
	}

	err = os.WriteFile(*outputFilePath, []byte(outputs(version)), 0660)
	if err != nil {
		log.Fatalf("failed to write outputs to output_file %s: %s\n", *outputFilePath, err.Error())
	}

	if echoLatestVersion != nil && *echoLatestVersion {
		log.Printf("Latest changelog version: v%s (%s)\n", version.Tag())

		for _, change := range version.Changes() {
			log.Printf("%s\n", change)
		}
	}
}

func outputs(version *changelog.Version) string {
	template := `
version="%s"
major="%s"
major_minor="%s"
major_minor_patch="%s"
changes="%s"
`
	changes := strings.Trim(strings.Join(version.Changes(), `\n`), `\n`)
	return fmt.Sprintf(template, version.Tag(), version.Major(), version.MajorMinor(), version.MajorMinorPatch(), changes)
}
