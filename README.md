# Step Runner

Step Runner is an RFC implementation for [GitLab Steps](https://docs.gitlab.com/ee/architecture/blueprints/gitlab_steps/), a CI/CD feature to define and use reusable components within a single job execution context.
See [HOWTO.md](HOWTO.md) for usage.

## Project status

Step Runner is currently an [experimental feature](https://docs.gitlab.com/ee/policy/experiment-beta-support.html#experiment).
It can be tested in GitLab CI/CD jobs but should not be used for production workloads yet.

- Demo: https://youtu.be/TU_53vWVKeE
- Code: https://gitlab.com/josephburnett/hello-step-runner
- Feedback: https://gitlab.com/gitlab-org/step-runner/-/issues/10
- Blueprint: https://docs.gitlab.com/ee/architecture/blueprints/gitlab_steps/

## Contributing
Contributions are welcome, see [CONTRIBUTING.md](./CONTRIBUTING.md) for more details.

## Project License

You can view this projects license in [LICENSE](./LICENSE).

## Release

During the experimental phase all changes to `main` are automatically built as a Docker image and tagged in the container repository as `v0`.
So any workflows referencing the image always use the latest version of the code, and behavior could change at any time.
See [HOWTO.md](HOWTO.md) for an example of how to test the Step Runner in a job by using the container image.
