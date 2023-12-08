# How to use Step Runner

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
