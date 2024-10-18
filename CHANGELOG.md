## v0 - 2024-10-17

### Breaking Changes:

- **Names must be alphanumeric**. All input and environment variable
  names must be alphanumeric. Specifically they must match the regular
  expression `^[a-zA-Z_][a-zA-Z0-9_]*$`. This makes expression easier
  to parse. This limitation was intended to be in place since Jan 4,
  2024
  (https://gitlab.com/gitlab-org/step-runner/-/commit/5a964c4ab52b71fc87a5e63e528d7024fc6043c7).
  However it mistakenly wasn't applied to the `ci` command.

## v0 - 2024-10-15

### Added:

- **CHANGELOG**. Step Runner is currently in a pre-release state
  where all changes to the `main` branch are published under the
  `gitlab.com/gitlab-org/step-runner:v0` image tag. This makes changes
  immediately available in production. In order to communicate
  breaking changes clearly, we are establishing a CHANGELOG indexed by
  date. Currently **only** breaking changes will be included in the
  CHANGELOG. For a complete list of changes view the Git history or
  Merge Requests. Step Runner will print the commit at which it was
  built to the job logs. When we create a release process the
  CHANGELOG will be indexed by semantic version **and** date.
