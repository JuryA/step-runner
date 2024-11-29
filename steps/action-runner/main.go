package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"

	"gitlab.com/components/action-runner/pkg/runner"
)

var (
	action      = flag.String("action", "", "The action to run")
	actionImage = flag.String("action-image", "", "The image in which to run the action")
	inputsJson  = flag.String("inputs", "", "The action inputs as a JSON struct")
	outputFile  = flag.String("output-file", "", "The step-runner output file")

	inputs = map[string]string{}
)

func main() {
	readAndCheckInputs()
	stepResult, err := runner.Run(*action, *actionImage, inputs)
	if err != nil {
		errorAndExit("running action %q: %v", *action, err)
	}
	data, err := protojson.Marshal(stepResult)
	if err != nil {
		errorAndExit("marshaling step result: %v", err)
	}
	data = []byte(string(data))
	err = os.WriteFile(*outputFile, data, 0600)
	if err != nil {
		errorAndExit("writing output file: %v", err)
	}
}

func readAndCheckInputs() {
	flag.Parse()
	if *action == "" {
		errorAndExit("--action is required")
	}
	if *actionImage == "" {
		errorAndExit("--action-image is required")
	}
	if *inputsJson == "" {
		errorAndExit("--inputs is required")
	}
	if *outputFile == "" {
		errorAndExit("--output-file is required")
	}
	// Check types one layer at a time so we can give a nice error message
	var inputAny any
	err := json.Unmarshal([]byte(*inputsJson), &inputAny)
	if err != nil {
		errorAndExit("unmarshaling JSON inputs: %v", err)
	}
	inputMap, ok := inputAny.(map[string]interface{})
	if !ok {
		errorAndExit("inputs should be a JSON struct. got %T", inputAny)
	}
	for k, v := range inputMap {
		valueString, ok := v.(string)
		if !ok {
			errorAndExit("input values must be strings. %q was a %T", k, v)
		}
		inputs[k] = valueString
	}
}

func errorAndExit(msg string, params ...any) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf(msg, params...)+"\n")
	os.Exit(1)
}
