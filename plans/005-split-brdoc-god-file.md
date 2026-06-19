# Plan 005: Split the `brdoc.go` god-file into `cpf.go` + `cnpj.go` (shed the stale brand name)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in "STOP conditions" occurs, stop and report — do not
> improvise. When done, update the status row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat 15c0c91..HEAD -- brdoc.go brdoc_test.go`
> If either changed since this plan was written, compare the "Current state"
> map against the live `brdoc.go` (`grep -nE '^(func|type|var|const|// ===)' brdoc.go`)
> before proceeding; on a mismatch, treat it as a STOP condition.

## Status
- **Priority**: P3
- **Effort**: M
- **Risk**: MED (moves the core CPF/CNPJ validators; the build + existing tests are the safety net)
- **Depends on**: none (land after 001 so CI validates the move)
- **Category**: tech-debt
- **Planned at**: commit `15c0c91`, 2026-06-18

## Why this matters
The project was renamed `brdoc` → `selo`, but the single largest source file is still
`brdoc.go` (517 lines) — a god-file holding the **CPF** type, the **CNPJ** type, shared
constants, and the deprecated `ValidateDocument` dispatcher, under the *old brand name*.
Every other document type already lives in its own file (`cpf` logic is the exception only
because it predates the pattern: `cnh.go`, `pis.go`, `renavam.go`, `voterid.go`, `cep.go`,
`phone.go`, `plate.go`, `cns.go`, `rg.go`, `pix.go`). This plan brings CPF/CNPJ in line:
`cpf.go` + `cnpj.go`, matching the repo's one-type-per-file convention and removing the
jarring `brdoc.go`/`brdoc_test.go` names. It is a **pure move** — no behavior changes — so
the existing test suite proves correctness.

## Current state
`brdoc.go` (package `selo`) is cleanly separable. Map (line numbers at commit `15c0c91`):

- **Imports** (lines 3–9): `fmt`, `math/rand/v2`, `slices`, `strconv`, `strings`.
- **Const block** (11–25): `CpfLength = 11`, `CnpjLength = 14`, and `IsDigit0`–`IsDigit9`
  (CPF region strings).
- `var notAcceptedCPF []string` (27) — **CPF only**.
- `var charToValue = map[rune]int{…}` (30–35) — **CNPJ only** (alphanumeric value map).
- `func init()` (37–49): populates `notAcceptedCPF` (uses `strings.Repeat`, `strconv.Itoa`)
  **and** calls `Register(&CPF{})` **and** `Register(&CNPJ{})`.
- **CPF section** (51–286): `type CPF` + all its methods/helpers.
- **CNPJ section** (288–491): banner `// === … CNPJ … ===`, `type CNPJ struct{}` + all its
  methods (`Generate`, `GenerateLegacy`, `Validate`, `Format`, `Kind`, `generateDigits`,
  `calculateDV`, `normalizeChar`, `processRemainingChars`, `digits`).
- **`ValidateDocument`** (493–517): deprecated dispatcher; references `CpfLength`,
  `CnpjLength`, `(&CNPJ{}).digits`, `onlyDigits`, `Validate`, `KindCPF`, `KindCNPJ`.

Cross-references are all **package-level** (everything is `package selo`), so moving a type
to a sibling file does not break callers — Go does not care which file a symbol lives in.
Verified separation: the CPF section does not call CNPJ code and vice-versa; `charToValue`
is CNPJ-only; `notAcceptedCPF` is CPF-only.

**Exact import sets** (computed from per-section usage — use these verbatim):
- Code that stays in `cpf.go` uses `fmt`, `strings`, `strconv`, `slices`, `rand` →
  imports: `fmt`, `math/rand/v2`, `slices`, `strconv`, `strings` (all five current imports).
- Code moving to `cnpj.go` uses `fmt`, `strconv`, `rand` only →
  imports: `fmt`, `math/rand/v2`, `strconv` (no `strings`, no `slices`).

Repo convention to match — an existing one-type file, e.g. `pis.go`: `package selo`, a
short import block, `const`/`var` for that type, `func init() { Register(&PIS{}) }`, then
the type and methods. Mirror that shape.

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (full) | `go test ./...` | all packages `ok` |
| Format | `gofmt -w cpf.go cnpj.go` then `gofmt -l cpf.go cnpj.go` | second prints nothing |
| Unused-import check | `go build ./...` | the compiler errors on any unused import |
| No stale name | `ls brdoc*.go 2>/dev/null` | (after step 5) nothing |

## Suggested executor toolkit
- If `goimports` is installed, run `goimports -w cpf.go cnpj.go` after the move to fix import
  blocks automatically. If not, set the imports manually to the exact sets given above — the
  compiler will tell you if one is wrong.

## Scope
**In scope** (the only files you may create/modify/rename):
- `brdoc.go` → becomes `cpf.go` (CPF + shared consts + `ValidateDocument`)
- `cnpj.go` (create — CNPJ)
- `brdoc_test.go` → renamed `cpf_cnpj_test.go` (rename only, no content change)

**Out of scope** (do NOT touch):
- Any behavior: do not change a single statement inside any function. This is a move only.
- `cpf_registry_test.go`, `cnpj_registry_test.go` — leave as-is.
- All other `*.go` files — none import by filename; none need editing.
- Splitting `cpf_cnpj_test.go` further into `cpf_test.go`/`cnpj_test.go` — explicitly deferred (see Maintenance notes).

## Git workflow
- Branch: `advisor/005-split-brdoc` off the current branch.
- Use `git mv` for renames so history is preserved.
- Conventional commits, e.g. `refactor: split brdoc.go into cpf.go and cnpj.go`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Rename the source file
`git mv brdoc.go cpf.go`

**Verify**: `ls cpf.go` exists; `ls brdoc.go` → not found. `go build ./...` → exit 0 (rename alone is a no-op for the compiler).

### Step 2: Create `cnpj.go` and move the CNPJ section into it
Create `cnpj.go` with this skeleton, then move code into it:
```go
package selo

import (
	"fmt"
	"math/rand/v2"
	"strconv"
)

// CnpjLength is the number of characters in a CNPJ.
const CnpjLength = 14

// charToValue maps alphanumeric CNPJ characters to their numeric weights (SERPRO).
var charToValue = map[rune]int{ /* … moved verbatim from cpf.go … */ }

func init() { Register(&CNPJ{}) }

// … the entire CNPJ section moved verbatim from cpf.go …
```
Now move, **verbatim** (cut from `cpf.go`, paste into `cnpj.go`):
1. The `charToValue` `var` block (the `map[rune]int{…}` literal) → replace the placeholder above.
2. The whole **CNPJ section**: from the banner comment immediately above `type CNPJ struct{}`
   through the end of `func (c *CNPJ) digits(...)` — i.e. everything between the end of
   `func (c *CPF) length(...)` and the `// Utility Functions` banner that precedes
   `ValidateDocument`. Paste it after the `init()` in `cnpj.go`.

Do not alter any moved line.

**Verify** (will still fail to build until step 3 — that's expected): proceed to step 3.

### Step 3: Fix `cpf.go` after the removal
In `cpf.go`:
1. Remove `CnpjLength = 14` from the `const` block (it now lives in `cnpj.go`). Keep
   `CpfLength` and all `IsDigit*`.
2. Remove the now-deleted `charToValue` var (already cut in step 2 — confirm it's gone).
3. In `func init()`, delete the line `Register(&CNPJ{})`. Keep the `notAcceptedCPF`
   population and `Register(&CPF{})`.
4. Confirm `cpf.go` still contains: the CPF section, `notAcceptedCPF`, and `ValidateDocument`
   (it references `CnpjLength` and `(&CNPJ{}).digits` — both still resolve, package-level).
5. Imports: `cpf.go` keeps `fmt`, `math/rand/v2`, `slices`, `strconv`, `strings`. (If
   `goimports` isn't available and the compiler reports an unused import, remove exactly the
   one it names.)

**Verify**: `go build ./...` → exit 0. `go vet ./...` → exit 0.

### Step 4: Format
`gofmt -w cpf.go cnpj.go`
**Verify**: `gofmt -l cpf.go cnpj.go` → prints nothing.

### Step 5: Rename the test file
`git mv brdoc_test.go cpf_cnpj_test.go` (rename only — the tests are `package selo` and
reference symbols, not filenames; no content change).

**Verify**: `ls brdoc*.go 2>/dev/null` → nothing. `go test ./...` → all packages `ok`.

### Step 6: Final full verification
**Verify**:
- `go build ./... && go vet ./... && go test ./...` → all `ok`.
- `git diff 15c0c91 -- cpf.go cnpj.go | grep -E '^[+-]' | grep -vE '^[+-]{3} |^[+-]\s*(package|import|"|\)|//|const|var|func init)' | grep -vE '^\+' | head` — sanity that **no CPF/CNPJ logic line was deleted** (deletions should only be the moved CNPJ block, the `CnpjLength` const, `charToValue`, and the `Register(&CNPJ{})` line). If you see deleted *logic* lines that reappear nowhere, STOP.

## Test plan
No new tests — this is a behavior-preserving move; the **existing** suite is the proof:
- `brdoc_test.go` (now `cpf_cnpj_test.go`) already covers CPF generate/validate/format/origin,
  CNPJ generate/validate/format/legacy, `calculateDV`, and `ValidateDocument`, plus benchmarks.
- `cpf_registry_test.go` / `cnpj_registry_test.go` cover registry integration.
- Verification: `go test ./...` → all pass, **same count as before** (no tests added or removed).

## Done criteria
ALL must hold:
- [ ] `cpf.go` and `cnpj.go` exist; `brdoc.go` and `brdoc_test.go` do not (`ls brdoc*.go` empty).
- [ ] `cnpj.go` contains `type CNPJ`, `CnpjLength`, `charToValue`, `func init() { Register(&CNPJ{}) }`, and imports exactly `fmt`, `math/rand/v2`, `strconv`.
- [ ] `cpf.go` contains `type CPF`, `CpfLength`+`IsDigit*`, `notAcceptedCPF`, `ValidateDocument`, and a single `init()` that registers CPF (not CNPJ).
- [ ] `go build ./... && go vet ./... && go test ./...` → all `ok`.
- [ ] `gofmt -l cpf.go cnpj.go` prints nothing.
- [ ] `grep -rn 'Register(&CNPJ{})' .` returns exactly one match (in `cnpj.go`); `grep -rn 'Register(&CPF{})' .` exactly one (in `cpf.go`).
- [ ] Only the in-scope files changed/renamed (`git status`).
- [ ] `plans/README.md` status row updated.

## STOP conditions
Stop and report (do not improvise) if:
- The live `brdoc.go` structure doesn't match the "Current state" map (drift).
- After the move, the build reports a symbol used in `cpf.go` that you moved to `cnpj.go`
  (or vice-versa) — it means the CPF/CNPJ separation isn't as clean as mapped; report the
  symbol, don't duplicate it across files.
- `go test ./...` fails — a move should never change a result; a failure means something was
  altered, not just moved. Report the failing test.
- The test count changes (a test was accidentally dropped during the file rename).

## Maintenance notes
- Deferred on purpose: splitting `cpf_cnpj_test.go` into `cpf_test.go` + `cnpj_test.go`, and
  relocating `ValidateDocument` out of `cpf.go` into `registry.go` (it's the deprecated
  registry dispatcher; it currently lives with CPF only because that's where it started).
  Both are cosmetic follow-ups; not worth the churn in this plan.
- `ValidateDocument` is **deprecated** (removal after 2026-07-18 per its doc comment). When
  that date passes and it's deleted, `cpf.go` shrinks further — no action needed now.
- Reviewer: this PR's diff should be almost entirely a move. Scrutinize that no statement
  inside any function changed — `git diff --color-moved` makes moved blocks obvious.
