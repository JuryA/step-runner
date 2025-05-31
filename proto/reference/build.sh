#!/bin/bash
set -e

# Build the step runner
go build -o step-runner main.go

# Run the simple hello world example
echo "Running hello world example:"
./step-runner ./examples/hello_world.textproto

# Run the composition example
echo "Running composition example:"
./step-runner ./examples/composition.textproto