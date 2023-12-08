# How to use Step Runner

Step Runner reads `steps` and executes them one at a time.
Each step takes `inputs` and environment variables (`env`) and produces `outputs` and additional environment (`exports`).
Steps have access to previous step outputs and exports as well as job-level parameters via the `context`. The context is accessed via `expressions`.



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
