# Plan 003: Finish the rebrand — `brdoc mcp:` error prefix → `selo mcp:`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> "STOP conditions" item occurs, stop and report. When done, update the status
> row for this plan in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat 15c0c91..HEAD -- mcp/server.go`
> If `mcp/server.go` changed since this plan was written, compare the "Current
> state" excerpt against the live code before proceeding; on a mismatch, treat
> it as a STOP condition.

## Status
- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `15c0c91`, 2026-06-18

## Why this matters
The project was renamed `brdoc` → `selo` (module `github.com/inovacc/selo`, package
`selo`, error prefixes `brdoc:` → `selo:`). One error prefix was missed because the
rename swept the literal `"brdoc: "` but this one reads `"brdoc mcp: "`. A user who
hits an MCP transport error sees the old brand. Trivial, but it's a visible
inconsistency in the freshly rebranded tool.

## Current state
File: `mcp/server.go` (package `mcp`).
```go
// mcp/server.go:275 (inside func Serve)
	return fmt.Errorf("brdoc mcp: %w", err)
```
This is the **only** remaining `brdoc` error prefix in non-test source. For contrast,
there is one **intentional** `brdoc` mention that must NOT change:
```go
// meta.go:5 — a comment explaining the package name choice; leave as-is
// "brdoc" (a domain term, not a brand).
```

## Commands you will need
| Purpose | Command | Expected on success |
|---|---|---|
| Build | `go build ./...` | exit 0 |
| Test (mcp) | `go test ./mcp/` | `ok` |
| Test (full) | `go test ./...` | all `ok` |
| Leftover scan | `grep -rn '"brdoc' --include='*.go' .` | only `meta.go:5` (the comment), nothing else |

## Scope
**In scope**: `mcp/server.go` (one line).
**Out of scope**:
- `meta.go:5` — intentional comment; do not touch.
- The filenames `brdoc.go` / `brdoc_test.go` — out of scope for this plan (see Maintenance notes).
- Any other package.

## Git workflow
- Branch: `advisor/003-mcp-prefix` off the current branch.
- Conventional commit, e.g. `fix(mcp): correct error prefix to selo`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Rename the prefix
In `mcp/server.go`, change the `Serve` error wrap from `"brdoc mcp: %w"` to
`"selo mcp: %w"`. Change nothing else.

**Verify**: `grep -n '"selo mcp: %w"' mcp/server.go` → matches line ~275; and
`grep -rn '"brdoc' --include='*.go' . | grep -v meta.go` → prints nothing.

### Step 2: Build + test
**Verify**: `go build ./... && go test ./...` → all packages `ok`.

## Test plan
No new test required — this is a string literal in an error path with no behavioral
assertion. (The existing MCP in-memory transport tests in `mcp/server_test.go` already
exercise the server; they don't assert this message text.) If you want belt-and-suspenders,
it is acceptable to skip adding a test here; the grep in Done criteria is the gate.

## Done criteria
ALL must hold:
- [ ] `grep -rn '"brdoc' --include='*.go' .` returns **only** `meta.go:5`.
- [ ] `go build ./... && go test ./...` → all `ok`.
- [ ] Only `mcp/server.go` modified (`git status`).
- [ ] `plans/README.md` status row updated.

## STOP conditions
Stop and report if:
- `mcp/server.go:275` no longer matches the excerpt.
- The leftover scan finds `brdoc` literals beyond `meta.go:5` and `mcp/server.go` — there
  may be more rebrand misses than this plan accounts for; report the full list rather than
  fixing them ad hoc.

## Maintenance notes
- Deferred on purpose (out of scope here): the root files are still named `brdoc.go` /
  `brdoc_test.go` even though the package is `selo`. Renaming them (`git mv`) is cosmetic,
  touches the largest file in the repo, and is tracked separately — see finding #6 in the
  audit / a future plan. Don't bundle it into this one.
- Reviewer: confirm `meta.go:5` was left intact.
