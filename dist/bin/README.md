## What is the purpose of this README?

To explain to engineers the purpose of this folder, and to ensure that the `//go:embed bin`
directive has something to embed on a new machine when distributed steps have not been generated.

## What is the bin folder for?

The `dist/bin` directory hosts "distributed" steps. These are generated from the steps found in
`dist/steps` and embedded into the distributed step-runner binary.

Like any other steps, distributed steps run in a separate process to the step-runner. Each step
is written to the file system before being executed.

## How to generate distributed steps

From the step-runner source folder, run `make build` to build the step-runner and dependent distributed steps.
