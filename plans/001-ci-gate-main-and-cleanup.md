# Plan 001: CI actually gates `main` (and stops failing / installing GUI cruft)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat 15c0c91..HEAD -- .github/workflows/`
> If `build.yml` or `test.yml` changed since this plan was written, compare the
> "Current state" excerpts against the live files before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status
- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dx
- **Planned at**: commit `15c0c91`, 2026-06-18

## Why this matters
The repository's whole flagship branch (`feat/complete-toolkit`, PR #3 into `main`)
has **no CI signal at all**. Three independent causes were confirmed:
1. **GitHub Actions is disabled** for the repo (`gh api repos/inovacc/selo/actions/permissions --jq .enabled` → `false`), so nothing runs.
2. Even with Actions on, **neither workflow gates `main`**: `build.yml` triggers on `develop` only; `test.yml` uses `branches-ignore: [main]`, which excludes PRs whose base is `main`. So a merge into `main` is never tested.
3. `build.yml` has **failed its last 3 runs** on `develop` (fast 14–19s failures, consistent with the GUI/audio apt-get step) and installs desktop libraries (`xorg-dev`, `libgl1-mesa-*`, `libasound2-dev`, `libpulse-dev`) that are **irrelevant to a CLI/library** — leftover template cruft.

`test.yml` (the reusable `inovacc/workflows` job: tests + lint + vulncheck) **passes** and is the one worth keeping. After this plan, pushes and PRs — including PRs targeting `main` — run that check, and `build.yml` either works or is gone.

## Current state
- `.github/workflows/test.yml` — the **working** check (last runs: success). Triggers exclude `main`:
  ```yaml
  # .github/workflows/test.yml:1-14
  name: Test
  on:
    push:
      branches-ignore: [ "main" ]
    pull_request:
      branches-ignore: [ "main" ]
  jobs:
    quality-check:
      uses: inovacc/workflows/.github/workflows/reusable-go-check.yml@main
      with:
        run-tests: true
        run-lint: true
        run-vulncheck: true
  ```
  The reusable workflow lives in the **public** repo `inovacc/workflows` (confirmed accessible) and already runs tests + lint + govulncheck.
- `.github/workflows/build.yml` — **failing**, `develop`-only, GUI cruft. Full file:
  ```yaml
  # .github/workflows/build.yml:1-37 (build-linux job; build-windows is identical minus the deps step)
  name: Build and Test
  on:
    push:
      branches: [ develop ]
    pull_request:
      branches: [ develop ]
  jobs:
    build-linux:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - name: Set up Go
          uses: actions/setup-go@v5
          with:
            go-version: latest
            check-latest: true
        - name: Install dependencies
          run: |
            sudo apt-get update
            sudo apt-get install -y xorg-dev libgl1-mesa-dev libegl1-mesa-dev libgles2-mesa-dev
            sudo apt-get install -y libasound2-dev libpulse-dev
        - name: Lint
          uses: golangci/golangci-lint-action@v8
          with:
            version: v2.4.0
        - name: Build
          run: go build -v ./...
        - name: Test
          run: go test -race -p=1 ./... -v
  ```
- This is a **pure-Go CLI + library** (`go.mod` module `github.com/inovacc/selo`, no Fyne/Ebiten/CGO GUI deps). The GUI system packages are never needed.
- Lint config exists: `.golangci.yml` (repo root).

## Commands you will need
| Purpose | Command | Expected on success |
|---|---|---|
| Confirm Actions state | `gh api repos/inovacc/selo/actions/permissions --jq .enabled` | prints `true` after step 1 |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (as CI) | `go test -race -p=1 ./...` | all packages `ok` |
| List workflows | `gh workflow list` | the workflows are listed |

## Scope
**In scope** (only files you may modify):
- `.github/workflows/test.yml`
- `.github/workflows/build.yml`

**Out of scope** (do NOT touch):
- The reusable workflow in `inovacc/workflows` (external repo) — not yours to edit.
- Any `Release` workflow / release process.
- Any `.go` source, `.golangci.yml`, or `go.mod`.

## Git workflow
- Branch: `advisor/001-ci-gate-main` off the current branch.
- Conventional-commit messages (repo style), e.g. `ci: gate main and drop GUI-dep cruft from build`.
- Do NOT push or open a PR unless the operator instructs it.

## Steps

### Step 1: Re-enable GitHub Actions (operator prerequisite)
Actions is disabled, so no workflow change can take effect until it is on. Try:
`gh api -X PUT repos/inovacc/selo/actions/permissions -f enabled=true -f allowed_actions=all`
Then confirm: `gh api repos/inovacc/selo/actions/permissions --jq .enabled` → `true`.

**If this returns a 403 / permission error**, you (the executor) lack admin rights — this is an operator action. Record it in your report as "Actions must be re-enabled by a repo admin in Settings → Actions → General → Allow all actions" and continue with the YAML edits (steps 2–3); they take effect once an admin re-enables Actions.

**Verify**: `gh api repos/inovacc/selo/actions/permissions --jq .enabled` → `true` (or, if 403, note the operator dependency and proceed).

### Step 2: Make `test.yml` run on `main` too
Edit `.github/workflows/test.yml`. Replace the `on:` block so the check runs on every push and on PRs to any base branch (including `main`):
```yaml
on:
  push:
    branches: [ "**" ]
  pull_request:
```
(`pull_request:` with no `branches`/`branches-ignore` key fires for PRs into any base branch, which is what gates `main`. `push: branches: ["**"]` keeps running on every branch as before, now including `main`.)

**Verify**: `grep -A4 '^on:' .github/workflows/test.yml` shows no `branches-ignore` and a bare `pull_request:` trigger.

### Step 3: Fix `build.yml` — drop the GUI cruft and gate `main`
Edit `.github/workflows/build.yml`:
1. **Delete the entire `Install dependencies` step** (the `- name: Install dependencies` block and its `run:` apt-get lines) from the `build-linux` job. The `build-windows` job has no such step — leave it.
2. Change the `on:` block to match `test.yml`'s new triggers:
   ```yaml
   on:
     push:
       branches: [ "**" ]
     pull_request:
   ```

**Verify**:
- `grep -c 'apt-get' .github/workflows/build.yml` → `0`
- `grep -c 'libgl1-mesa\|xorg-dev\|libasound2\|libpulse' .github/workflows/build.yml` → `0`
- `grep -A4 '^on:' .github/workflows/build.yml` shows the new triggers (no `develop`-only restriction).

### Step 4: Sanity-check the repo still builds and tests locally
This does not exercise CI, but proves the code the workflows will run is green:
```
go build ./... && go vet ./... && go test -race -p=1 ./...
```
**Verify**: all packages report `ok`; exit 0. (If `-race` is unavailable in your environment — it needs a C toolchain — fall back to `go test ./...` and note it.)

## Test plan
No application tests change in this plan (CI config only). Validation is:
- Config greps in steps 2–3 (above).
- The real end-to-end proof is the **next push once Actions is enabled**: a workflow run should appear via `gh run list --limit 3`. If you (or the operator) can push to a throwaway branch, confirm a run is created. Do not push to `main` or `feat/complete-toolkit` to test this.

## Done criteria
ALL must hold:
- [ ] `gh api repos/inovacc/selo/actions/permissions --jq .enabled` → `true` (or report records the operator dependency if 403).
- [ ] `grep -c 'branches-ignore' .github/workflows/test.yml` → `0`.
- [ ] `.github/workflows/test.yml` has a bare `pull_request:` trigger (PRs to `main` fire it).
- [ ] `grep -c 'apt-get' .github/workflows/build.yml` → `0`.
- [ ] `build.yml` triggers are no longer `develop`-only.
- [ ] `go build ./... && go test ./...` → all `ok`.
- [ ] Only the two in-scope workflow files modified (`git status`).
- [ ] `plans/README.md` status row updated.

## STOP conditions
Stop and report (do not improvise) if:
- The `on:` blocks in the live workflow files don't match the "Current state" excerpts (CI was changed since this plan).
- Removing the GUI deps step still leaves `build.yml` structurally broken (e.g. it referenced an artifact those libs produced) — it should not, but if `go build ./...` needs a system lib, STOP and report which.
- You cannot determine whether the `inovacc/workflows` reusable workflow already performs a build (it does test+lint+vulncheck) — if so and you were tempted to **delete** `build.yml`, do **not**; keeping a fixed `build.yml` is the conservative choice for this plan.

## Maintenance notes
- For the reviewer: confirm the new triggers don't double-run CI wastefully (push + PR on the same branch is normal for GitHub and acceptable).
- If the team later standardizes entirely on the `inovacc/workflows` reusable check, `build.yml` may become redundant and can be deleted in a follow-up — out of scope here because it provides the explicit linux+windows build matrix the reusable workflow's coverage isn't confirmed to include.
- Re-enabling Actions is a repo **setting**; if it gets disabled again (e.g. by an org policy or another rename), CI silently goes dark — worth a note in `CONTRIBUTING`/`CLAUDE.md`.
