# Add --pr Mode to go_bump

mode: feature
state: review
root_git: .
worktree: .worktrees/feature/go-bump-pr-mode
branch: feature/go-bump-pr-mode
target_branch: master

> **Resume protocol:** read **Dev Feedback** and the **State Tracker** below first, then jump to the
> step marked `Current`. Ensure that you are in the correct worktree and branch according to preamble here. Update current with Developer feedback and update the tracker after every meaningful change.
> Do not mutate completed steps; append a new entry instead.

---

## Initial Description

Add a `--pr` mode to `go_bump` that would:
- Checkout a branch preparing a PR for the bump
    - Keep current branch to be able to return to it once ready
    - Ensure this is done in some defer step to ensure we return to normal branch
- Bump like we do right now
- Git add go.mod/go.sum file
- Generate a correct commit with message + details about the bump (from_version => to_version)
- Using `gh` if available, open a PR on GitHub on behalf of the user
    - Use kept "current branch" as target
- If `gh` is not available, git push and present user's the URL to open a PR for the branch (based on git remote)
    - Use kept "current branch" as target

## Dev Feedback

## Spec & Implementation

### Overview

Added `--pr` boolean flag to `go_bump`.  When set, the command:

1. Reads the current branch via `git rev-parse --abbrev-ref HEAD`.
2. Creates + checks out a `bump/<name>` branch (or `bump/dependencies` for multiple packages).
3. Installs a `defer` to restore the original branch on exit.
4. Runs `go get` via `bumpWithOutput` (captures raw output for parsing).
5. Parses upgrade lines (`go: upgraded <module> <from> => <to>`) from the output.
6. Aborts early (with branch restore) when nothing was upgraded.
7. Optionally runs `go mod tidy` if `config.AfterBump.GoModTidy` is true.
8. Stages `go.mod` and `go.sum` with `git add`.
9. Commits with a message generated from the parsed upgrade entries.
10. If `gh` is in `$PATH`: runs `gh pr create --base <original> --title ... --body ...`.
11. Otherwise: runs `git push --set-upstream origin <bump-branch>` and prints a GitHub compare URL derived from the `origin` remote URL.

### Files changed

- `cmd/go_bump/main.go` — added `--pr` flag, route to `runPRMode` when set.
- `cmd/go_bump/pr.go` — new file with all PR-mode logic.
- `cmd/go_bump/pr_test.go` — unit tests covering parsing, branch naming, commit message generation, remote URL normalisation, and compare URL construction.
- `CHANGELOG.md` — new file documenting the addition.

## State Tracker

**Last Updated:** 2026-05-20
**Current Step:** Step 1 — Implementation complete, ready for review
**Status:** Review
