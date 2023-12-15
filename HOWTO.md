# Step Runner Experiment

- Demo: https://youtu.be/TU_53vWVKeE
- Code: https://gitlab.com/josephburnett/hello-step-runner
- Feedback: https://gitlab.com/gitlab-org/step-runner/-/issues/10
- Blueprint: https://docs.gitlab.com/ee/architecture/blueprints/gitlab_steps/

## How Step Runner works

- Step Runner reads `steps` and executes them one at a time.
- Each step can take `inputs` and environment variables (`env`), and can produce `outputs` and additional environment variables (`exports`).
- Steps have access to previous step outputs and exports, and also job-level parameters through `context`.
- Context is accessed with `expressions`.

For example, using a step with one input:

```yaml
- name: hello-world-step
  step: https+git://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step
  inputs:
    echo: hello world
```

- Steps are defined in a `step.yml` file.
- Step definitions consist of a `spec` which lists inputs and outputs, and an implementation of either a single `exec` or a list of `steps`.
- Inputs can have a `type` and a `default`.
- An `exec` consists of a `command` to run and a directory in which to run it (`workDir`).
- A `steps` implementation is just another list of step invocations.

Example step definition:

```yaml
spec:
  inputs:
    echo:
      type: string
      default: yo
  outputs:
    echo:
---
type: exec
exec:
  command:
    - bash
    - -c
    - echo echo=${{inputs.echo}} | tee ${STEP_RUNNER_OUTPUT}
```

- Inputs and other context values are provided to the command through expressions within `${{ }}` delimiters.
- Outputs are written in a `key=value` format a file at `$STEP_RUNNER_OUTPUT`.
- Exports are written in the same format to `$STEP_RUNNER_ENV`.

## Use in GitLab CI/CD

During the experimental phase, Step Runner is provided as a Docker image tagged as `v0`.
This docker image contains the `step-runner` binary, which is required to use steps in GitLab CI/CD.

Define the list of steps to execute in a `STEPS` CI/CD variable, which the `step-runner ci` command retrieves and executes.
The results are written to a `step-results.json` file and saved as an artifact that can be viewed after the job completes.

Example test GitLab CI/CD job:

```yaml
hello-world-job:
  image: registry.gitlab.com/gitlab-org/step-runner:v0
  script:
    - /step-runner ci
  variables:
    STEPS: |
      - name: hello-world-step
        step: https+git://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step
        inputs:
          echo: hello world
  artifacts:
    paths:
      - step-results.json
```

## Data structures

- The structure of steps and their definitions are defined in the [`step.proto`](https://gitlab.com/gitlab-org/step-runner/-/blob/main/proto/step.proto) Protocol Buffer file.
- Additional [syntactic sugar](https://docs.gitlab.com/ee/architecture/blueprints/gitlab_steps/steps-syntactic-sugar.html) will be added to the syntax but it will always be folded down to the baseline proto structures.

The [Iteration 2 epic](https://gitlab.com/groups/gitlab-org/-/epics/12167) proposes a `script` keyword as syntactic sugar and the underlying data models.

## Expression syntax

- Expressions are a language for accessing and processing the context and step outputs.

The [Iteration 3 epic](https://gitlab.com/groups/gitlab-org/-/epics/12168) proposes expression support for `if` statements and more complex parsing of expressions.

Additionally:

- Environment variable values and output values are always of type `string`.
- Inputs can be of type `string`, `number`, `bool`, or `struct`.
- Nested fields of an input struct can be accessed with a `.` syntax path (for example: `foo.bar`).

Example valid expressions:

- `${{ inputs.echo }}`
- `${{ inputs.foo.bar }}`
- `${{ env.BAZ }}`
- `${{ steps.hello-world-step.outputs.echo }}`
- `${{ steps.hello-world-step.exports.BAM }}`

Valid locations for expressions:

- Input values (but not defaults)
- Environment variable values
- Exec commands
- Exec working directory
- Steps output values
