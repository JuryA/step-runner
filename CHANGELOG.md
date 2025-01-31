## v0.4.0 (2025-01-30)

- The `bootstrap` command is idempotent. See !170.
- Script falls back to use `sh` when `bash` is not available. See [script!2](https://gitlab.com/components/script/-/merge_requests/2).
- Bootstrap command does not print version info. See !173.

## v0.3.0 (2025-01-06)

- Accept dir and file after /-/. See !169.

### Breaking Changes:

- Steps must be in the `steps` folder in the repository unless the
  expanded step syntax is used. See !gitlab/177038 for documentation
  update.

## v0.2.1 (2024-12-13)

- Initial working dir CI_PROJECT_DIR in CI. See !165.
- Ignore NoErrAlreadyUpToDate error when cloning a steps repo. See !166.

## v0.2.0 (2024-11-21)

- Make the initial Run API call wait for the connection to be ready. See !161+.
- Log name of executing step. See !158+.

### Breaking Changes:

- **export_file format is the same as the output_file**. Key/values must be
  written in the form `name=JSON value`. See !157+.
- **output file format** (and export_file format) changed to be JSON on each line.
  Each line of JSON must have non-null `name` and `value` keys. See !159+.
- **Use `run:` instead of `steps:` to run a sequence of steps**. This is consistent with what is used
  when defining steps for a job in the `.gitlab-ci.yml` file. See !145+.

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
