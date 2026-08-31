## Unreleased

### Added

- `dockerx`: new command grouping Docker helpers. First subcommand is `dockerx compose down-all`, which lists every running Docker Compose project and runs `docker compose down` on each of them. Volumes are deleted by default (`--keep-volumes` to keep them), `--except` leaves given project name(s) running, and the list of affected projects is shown for confirmation before anything is deleted (`--yes` to skip, and required when not attached to a terminal).
- `go_bump --pr`: new flag that creates a dedicated bump branch, commits the dependency upgrade, and opens a pull request. Branch naming: single package uses `bump/<short>-to-<version>`; 2–3 packages use `bump/<short1>-<short2>[-<short3>]`; 4+ packages use `bump/dependencies`. If the branch already exists, the user is prompted for confirmation before it is deleted. The branch is pushed to the remote before `gh pr create` or a manual compare URL is shown.

### Changed

- `go_bump --pr` now validates that the package(s) you actually requested were upgraded by `go get`. When a requested package was already at the desired version (so only unrelated transitive dependencies got bumped), the command aborts the PR flow, lists what was/wasn't upgraded, and interactively asks whether to revert the unrelated `go.mod`/`go.sum` changes before returning to the original branch.
