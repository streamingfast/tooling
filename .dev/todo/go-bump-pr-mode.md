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

All feedback items addressed in commit `08ab1e3`:

1. **Merged bump/bumpWithOutput** — replaced both with a single `goGet(ctx, w, packageIDs...)` helper that writes output to any `io.Writer` (or nil to suppress).
2. **Single-package branch name** — implemented Option A: create `bump/<short>` initially, run `go get`, parse upgraded version, rename to `bump/<short>-to-<version>` via `git branch -m`.
3. **Branch name for 2-3 packages** — `preBumpBranchName` joins short names: 2 pkgs → `bump/a-b`, 3 pkgs → `bump/a-b-c`, 4+ → `bump/dependencies`.
4. **Branch already exists** — uses `cli.AskConfirmation` to prompt before `git branch -D`; aborts if user declines or is non-interactive.
5. **gh pr create fix** — branch is now pushed to remote (via `git push --set-upstream origin <branch>`) before `gh pr create`; also passes `--head <branch>` explicitly.

## Spec & Implementation

### Overview

Added `--pr` boolean flag to `go_bump`.  When set, the command:

1. Reads the current branch via `git rev-parse --abbrev-ref HEAD`.
2. Computes the initial branch name from package IDs (`preBumpBranchName`):
   - 1 package: `bump/<short>`
   - 2 packages: `bump/<short1>-<short2>`
   - 3 packages: `bump/<short1>-<short2>-<short3>`
   - 4+ packages: `bump/dependencies`
3. If the computed branch exists, asks the user for confirmation before deleting it (via `cli.AskConfirmation`).
4. Creates + checks out the initial branch.
5. Installs a `defer` to restore the original branch on exit.
6. Runs `go get` via `goGet` (the unified helper that both PR-mode and normal mode use).
7. Parses upgrade lines from output; aborts when nothing was upgraded.
8. For a single package, renames the branch to `bump/<short>-to-<new-version>` via `git branch -m`.
9. Optionally runs `go mod tidy` if `config.AfterBump.GoModTidy` is true.
10. Stages `go.mod` and `go.sum` with `git add`.
11. Commits with a message generated from the parsed upgrade entries.
12. Pushes the branch to `origin` with `git push --set-upstream origin <branch>`.
13. If `gh` is in `$PATH`: runs `gh pr create --base <original> --head <branch> --title ... --body ...`.
14. Otherwise: prints a GitHub compare URL derived from the `origin` remote URL.

### Files changed

- `cmd/go_bump/main.go` — replaced `bump`/`bumpWithOutput` with `goGet(ctx, w, ...)`.
- `cmd/go_bump/pr.go` — full PR-mode logic with updated branch naming, `AskConfirmation`, push-before-PR, and `--head` flag.
- `cmd/go_bump/pr_test.go` — tests updated to cover `preBumpBranchName`, `finalBranchName`, `moduleShortName`, and removed old `bumpBranchFromPackageIDs`/`prBranchName` tests.
- `CHANGELOG.md` — updated entry with full feature description.

## State Tracker

**Last Updated:** 2026-05-20
**Current Step:** Step 2 — Review feedback addressed
**Status:** Review

- Step 1 (2026-05-20): Initial implementation complete.
- Step 2 (2026-05-20): Addressed all dev feedback: merged goGet helper, improved branch naming (1 pkg gets `-to-<version>`, 2-3 join short names, 4+ generic), branch-exists confirmation via `cli.AskConfirmation`, push before `gh pr create`, explicit `--head` flag.
