## What is the purpose of this README?

To explain to engineers the purpose of this folder, and to ensure that the `//go:embed bin`
directive has something to embed on a new machine when built-in steps have not been generated.

## What is the bin folder for?

The `builtin/bin` directory hosts "built-in" steps. These are generated from the built-in steps found in
`builtin/steps`.

Like any other steps, built-in steps are run in a separate process to the step-runner. Each built-in step
is written to the file system before being executed.

## How to generate built-in steps

From the step-runner source folder, run `make build` to build the step-runner and dependent built-in steps.
