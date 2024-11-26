## v0.2.0 (2024-11-21)

- Log name of executing step

## v0.1.0 (2024-10-28)

- Version the Step Runner

## v0.0.0 (2024-10-28)

### Breaking Changes:

- **Input and output types must be declared in the spec**. Variables no longer
  default to `string`.

## v0.0.0 (2024-10-21)

### Breaking Changes:

- **Output type `raw_string` removed**. Output variables can no longer
  be of type `raw_string`. Recommend changing the output type to `string` and
  surrounding the value in quotes when writing to the output file.

- **Default Output type changed to string**. Due to the `raw_string` type being
  removed.

## v0.0.0 (2024-10-17)

### Breaking Changes:

- **Names must be alphanumeric**. All input and environment variable
  names must be alphanumeric. Specifically they must match the regular
  expression `^[a-zA-Z_][a-zA-Z0-9_]*$`. This makes expression easier
  to parse. This limitation was intended to be in place since Jan 4,
  2024
  (https://gitlab.com/gitlab-org/step-runner/-/commit/5a964c4ab52b71fc87a5e63e528d7024fc6043c7).
  However it mistakenly wasn't applied to the `ci` command.

## v0.0.0 (2024-10-15)

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
