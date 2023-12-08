# How to use Step Runner

Step Runner reads `steps` and executes them one at a time.
Each step takes `inputs` and environment variables (`env`) and produces `outputs` and additional environment variables (`exports`).
Steps have access to previous step outputs and exports as well as job-level parameters via the `context`.
The context is accessed via `expressions`.

Example step invocation:

```yaml
- name: hello-world-step
  step: https+git://gitlab.com/gitlab-org/ci-cd/runner-tools/echo-step
  inputs:
    echo: hello world
```

Steps are defined in a `step.yml` file.
Step definitions consist of a `spec` which lists inputs and ouputs, and an implementation of either a single `exec` or a list of `steps`.
Inputs can have a `type` and a `default`.
An `exec` consists of a `command` to run and a directory in which to run it (`workDir`).
A `steps` implementation is just another list of step invocations.

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

Inputs and other context values are provided to the command through expressions within `${{ }}` delimiters.
Outputs are written in a `key=value` format a file at `$STEP_RUNNER_OUTPUT`.
And exports are written in the same format to `$STEP_RUNNER_ENV`.

## Use in CI

In order to use steps in GitLab CI the `step-runner` binary must be available in the execution environment.
During the experimental phase, Step Runner is provided as a Docker image tagged as `v0`.
The `ci` command looks for an environment variable `STEPS` with the list of steps to execute.
The results are written to a file `step-results.json`.

Example GitLab CI job:

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