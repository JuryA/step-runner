package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	outputFile = flag.String("output", "", "")
)

func main() {
	flag.Parse()

	if outputFile == nil || *outputFile == "" {
		panic("must specify an output file to write to")
	}

	err := os.WriteFile(*outputFile, []byte(`{"name":"message", "value":"publish step"}`), 0640)
	if err != nil {
		panic(fmt.Errorf("cannot write to output file: %w", err))
	}
}
