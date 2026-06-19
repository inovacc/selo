# Plan 004: Verify the RG (SP/RJ) check-digit convention against authoritative sources

> **Executor instructions**: This is an **investigation** plan, not a blind fix.
> Do the research, run the verification, and then take the branch the evidence
> points to. Run every verification command. If a "STOP conditions" item occurs,
> stop and report. When done, update the status row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat 15c0c91..HEAD -- rg.go rg_test.go`
> If either changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, STOP.

## Status
- **Priority**: P2
- **Effort**: M
- **Risk**: MED (a fix would change validation behavior + tests)
- **Depends on**: none (but verify before any "multi-state RG" work is built on top)
- **Category**: bug (correctness — confidence MED; this is a verify-then-maybe-fix)
- **Planned at**: commit `15c0c91`, 2026-06-18

## Why this matters
The RG (Registro Geral) validator for SP/RJ was built without external review, and the
check-digit **convention is genuinely ambiguous in public sources**. The implementation
uses `DV = 11 − (sum mod 11)` (encoding `10 → 'X'`, `11 → '0'`). Several public references
instead describe SP-RG as `DV = sum mod 11` (encoding `10 → 'X'`, with the remainder used
directly). The current tests only prove **internal** consistency (Generate uses the same
function Validate checks), which would pass under *either* convention — so they cannot tell
us whether the validator accepts **real** SP/RJ RG numbers. If the convention is wrong,
every real RG fails validation while every fake one this tool generates passes. This plan
resolves the ambiguity with authoritative evidence before anyone builds multi-state RG on top.

## Current state
File: `rg.go` (package `selo`). The convention lives in three places that must stay consistent:
```go
// rg.go:79-85 — the check-digit computation (the convention under question)
func (r *RG) checkDigit(base [RGBaseLength]int) int {
	sum := 0
	for i := 0; i < RGBaseLength; i++ {
		sum += base[i] * rgWeights[i]   // rgWeights = {2,3,4,5,6,7,8,9}
	}
	return 11 - (sum % 11)              // <-- convention: 11 - (sum % 11), range 1..11
}

// rg.go:41-51 — parse() decodes the stored check char into the same 1..11 space
//   'X'/'x' -> 10 ; '0' -> 11 ; '1'..'9' -> that digit

// rg.go (Generate + Format) encode an int check value back to a char:
//   10 -> 'X' ; 11 -> '0' ; else '0'+dv
```
Tests in `rg_test.go` use synthetic samples derived from this same convention
(e.g. `24.678.131-2`, `10.000.006-X`) — they do **not** include any independently-sourced
real RG number.

The two candidate conventions:
- **A (current)**: `DV = 11 − (sum % 11)`; `10→X`, `11→'0'`.
- **B (alternative)**: `DV = sum % 11`; `10→X` (and `'0'` means remainder 0, not 11).

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Build | `go build ./...` | exit 0 |
| RG tests | `go test ./... -run RG` | all pass |
| Full tests | `go test ./...` | all `ok` |
| Web research | (use the host's web search / fetch tool, or the `ctx_fetch_and_index` + `ctx_search` MCP tools if available) | — |

> NOTE: this repo blocks raw `curl`/`wget`/`WebFetch` in agent environments. Use a
> web-search tool or `ctx_fetch_and_index(url, source)` then `ctx_search(queries)`.

## Scope
**In scope**:
- `rg.go` (only if the evidence shows convention A is wrong)
- `rg_test.go` (add the sourced real-sample regression test in all cases)
- `docs/BACKLOG.md` (record the verification outcome)

**Out of scope**:
- All other document types (their algorithms were verified during the build).
- Adding new UFs to RG (that is a separate "multi-state RG" effort; do not start it here).

## Git workflow
- Branch: `advisor/004-rg-convention` off the current branch.
- Conventional commits: `test(rg): pin authoritative RG sample` and, if needed, `fix(rg): correct SP/RJ check-digit convention`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Gather authoritative evidence
Research the SP-SSP / RJ-Detran RG check-digit algorithm. Acceptable sources, best first:
a government/official spec; a widely-cited, well-reviewed implementation (e.g. a popular
validation library in any language) that states its source; or **at least two
independently-published, real-format SP RG numbers with their check digit** that you can
test. Capture: which convention (A or B) the sources describe, and 2+ concrete
`base(8 digits)+checkchar` examples. Write what you found (with URLs) into your report.

**Verify**: you have, in writing, (i) the convention the sources agree on, and (ii) ≥2
concrete RG samples to test. If sources **conflict irreconcilably**, that is itself the
finding — go to the STOP condition for "sources conflict".

### Step 2: Test the real samples against the current implementation
For each sourced sample, run it through the current validator. Add a temporary test or use
a tiny throwaway `go test` case calling `selo.NewRG().Validate("<sample>")`. (You may also
call `ValidateUF(sample, selo.UFSP)`.)

**Verify**: record, per sample, whether the current code returns `true` or `false`.

### Step 3a: If the real samples PASS → convention A is correct
Keep `rg.go` unchanged. Add the sourced samples as a permanent regression test in
`rg_test.go` (model it on the existing `TestRG_Validate` table). This converts the
ambiguity into a pinned, externally-grounded guarantee.

**Verify**: `go test ./... -run RG` → all pass, including the new real-sample cases.

### Step 3b: If the real samples FAIL → convention is wrong (likely B)
This is a real bug. Change `rg.go:checkDigit` to the evidence-backed convention and make
`parse`, `Generate`, and `Format` consistent with it (they currently encode `'0'→11`,
which only makes sense for convention A — under B, `'0'` means remainder 0 and there is no
`11` value). Specifically, if B is correct:
- `checkDigit` returns `sum % 11`.
- The `'0' → 11` branch in `parse` (rg.go:45-46) becomes `'0' → 0`.
- `Generate`/`Format` drop the `11 → '0'` case; `10 → 'X'`, else `'0'+dv` (dv 0..9).
Then update the synthetic samples in `rg_test.go` to the new convention **and** keep the
sourced real samples as the authority.

**Verify**: `go test ./...` → all `ok`; the sourced real samples validate `true`; the
Generate→Validate round-trip test still passes.

### Step 4: Record the outcome
Append a one-line result to `docs/BACKLOG.md` near any RG note: either "RG SP/RJ convention
verified (A) against <source> — real samples pinned in tests" or "RG SP/RJ convention
corrected (A→B) per <source>".

**Verify**: `grep -n 'RG' docs/BACKLOG.md` shows your note.

## Test plan
- Add `TestRG_AuthoritativeSamples` to `rg_test.go` with the ≥2 externally-sourced real RG
  numbers, asserting `Validate(sample) == true`. Model structure on `TestRG_Validate`.
- If a fix was needed, the existing synthetic-sample tests are updated to the corrected
  convention; the round-trip and `ErrUFNotImplemented` tests stay.
- Verification: `go test ./... -run RG` → all pass.

## Done criteria
ALL must hold:
- [ ] Report states the convention the sources support, with URLs and ≥2 real samples.
- [ ] `rg_test.go` contains ≥2 externally-sourced real-RG regression cases that pass.
- [ ] `go test ./...` → all `ok`.
- [ ] If `rg.go` was changed, `checkDigit`/`parse`/`Generate`/`Format` are mutually
      consistent and the round-trip test passes.
- [ ] `docs/BACKLOG.md` records the outcome.
- [ ] Only in-scope files modified (`git status`).
- [ ] `plans/README.md` status row updated.

## STOP conditions
Stop and report (do not improvise) if:
- You **cannot find** any authoritative source or any real, verifiable RG sample. Do NOT
  invent samples or pick a convention by coin-flip — report "unverifiable; left as convention A"
  so a human can decide. (Inventing a sample would re-introduce the exact circularity this plan exists to break.)
- Sources **conflict** (some say A, some say B) with no tiebreaker. Report both with their
  evidence; leave the code unchanged; recommend the maintainer decide.
- A fix to `rg.go` would require changing more than `checkDigit`/`parse`/`Generate`/`Format`.
- The live `rg.go` doesn't match the "Current state" excerpt.

## Maintenance notes
- This must be settled **before** any "multi-state RG" feature (audit direction item D2) —
  building 20+ more per-UF algorithms on an unverified convention would multiply the error.
- Reviewer: the key question is not "do the tests pass" (they pass under either convention)
  but "do the *externally-sourced real* samples validate". Scrutinize the provenance of
  those samples.
- RG check digits are not nationally standardized; some UFs beyond SP/RJ use entirely
  different schemes — keep the `ErrUFNotImplemented` guard for those.
