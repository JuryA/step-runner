# Usage

```yaml
hello-world-job:
  image: registry.gitlab.com/gitlab-org/step-runner:v0
  script:
    - /step-runner ci
  variables:
    STEPS: |
      type: steps
      steps:
        - name: hello-world-step
          step: https+git://gitlab.com/gitlab-org/ci-cd/runner-tools/step-runner-e2e-test-project
          inputs:
            echo: hello world
  artifacts:
    paths:
      - step-results.json
```