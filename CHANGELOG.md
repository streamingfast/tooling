## Unreleased

### Added

- `go_bump --pr`: new flag that creates a dedicated `bump/<name>` branch, commits the dependency upgrade, and opens a pull request. Uses `gh pr create` when the `gh` CLI is available; otherwise pushes the branch and prints a GitHub compare URL for manual PR creation.
