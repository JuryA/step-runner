package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/gitlab-org/step-runner/proto"
	"github.com/gitlab-org/step-runner/proto/reference/environment"
	"github.com/gitlab-org/step-runner/proto/reference/expression"
	"github.com/gitlab-org/step-runner/proto/reference/executor"
	"google.golang.org/protobuf/encoding/prototext"
)

// This is a reference implementation of a function execution.
// Main takes a single input which is the filename of a text
// protobuf Function. It proceeds with execution depth-first
// and then it outputs to STDOUT the Result as a text proto.

func main() {
	// Check arguments
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <function-file>\n", os.Args[0])
		os.Exit(1)
	}

	// Get the function filename from arguments
	functionFile := os.Args[1]

	// Read the function from file
	functionBytes, err := ioutil.ReadFile(functionFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading function file: %v\n", err)
		os.Exit(1)
	}

	// Parse the function from text proto
	function := &proto.Function{}
	err = prototext.Unmarshal(functionBytes, function)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing function proto: %v\n", err)
		os.Exit(1)
	}

	// Create the execution components
	evaluator := expression.NewEvaluator()
	envManager := environment.NewManager(evaluator)
	functionExecutor := executor.NewFunctionExecutor(envManager, evaluator)

	// Create initial environment from the system
	initialEnv := envManager.CreateInitialEnvironment()

	// Execute the function with a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	result, err := functionExecutor.Execute(ctx, function, initialEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing function: %v\n", err)
		os.Exit(1)
	}

	// Format the result as a text proto
	resultBytes, err := prototext.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting result: %v\n", err)
		os.Exit(1)
	}

	// Output the result to stdout
	fmt.Println(string(resultBytes))

	// Exit with the function's exit code if it was an exec
	if exitCode, ok := result.Results.(*proto.Result_ExitCode); ok {
		os.Exit(int(exitCode.ExitCode))
	}
}
