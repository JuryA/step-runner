# Step Runner

Step Runner is an RFC implementation for [GitLab Steps](https://docs.gitlab.com/ee/architecture/blueprints/gitlab_steps/), a CI feature to define and use reusable components within a single job execution context.
See [HOWTO.md](./HOWTO.md) for usage.

## Project status

Step Runner is currently in an experimental state.
It can be used in GitLab CI jobs but should not be used for production workloads yet.

### Experiment

- Demo: https://youtu.be/TU_53vWVKeE
- Code: https://gitlab.com/josephburnett/hello-step-runner
- Feedback: https://gitlab.com/gitlab-org/step-runner/-/issues/10
- Blueprint: https://docs.gitlab.com/ee/architecture/blueprints/gitlab_steps/

## Release

During the experimental phase all changes to `main` are automatically built and tagged in the container repository as `v0`.
So all workflows referencing the image will get continuous updates.
See [HOWTO.md](./HOWTO.md) for an example of how to use the Step Runner container in a job.