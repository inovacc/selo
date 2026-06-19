# Plan 006 (design/spike): Inscrição Estadual — architecture + a verified first-batch of UFs

> **Executor instructions**: This is a **design/spike** plan, not a build-all-27 plan.
> Its deliverables are: (1) a small, verified, production-quality implementation for a
> *first batch* of states, behind a structure the rest can grow into, and (2) a written
> handoff (`docs/IE-NOTES.md`) capturing the per-UF research, open questions, and the
> remaining work. Run every verification command. **If you cannot find an authoritative
> algorithm AND ≥2 real, sourced sample numbers for a given UF, do NOT implement that UF**
> — drop it to the "needs research" list and move on (see STOP conditions). Update the
> status row in `plans/README.md` when done.
>
> **Drift check (run first)**: `git diff --stat 15c0c91..HEAD -- document.go rg.go registry.go`
> If `document.go`/`rg.go`/`registry.go` changed since this plan was written, compare the
> "Current state" excerpts to the live code before proceeding; on a mismatch, STOP.

## Status
- **Priority**: P2 (direction — maintainer's call; high product value)
- **Effort**: L (the *full* 27-UF feature is multi-day; this spike is M — first batch + scaffolding + notes)
- **Risk**: MED (new public API surface + per-UF algorithm correctness)
- **Depends on**: Plan 004 (verify the RG convention) is a useful precedent — read it first; the same "never invent samples" discipline applies here.
- **Category**: direction
- **Planned at**: commit `15c0c91`, 2026-06-19

## Why this matters
**Inscrição Estadual (IE)** — the state-level tax registration number — is the single
biggest gap in the Brazilian-document Go ecosystem: the reference library
`paemuri/brdoc` has had it as an open request (issue #7) since inception and **never
shipped it**, because there is no national standard — each of the 27 federative units has
its **own** algorithm (different lengths, weights, and check-digit rules; several UFs accept
multiple formats). Shipping even a credible subset makes `selo` strictly ahead of every
existing library. The hard part is not the code — it's **getting 27 per-UF algorithms right
and verified**. This spike de-risks that: it establishes the architecture, proves it on a
high-value first batch with *sourced real samples*, and hands off a structured TODO for the
rest. It deliberately does **not** attempt all 27 in one pass — that's how wrong check-digit
math ships at scale.

## Current state — the pattern to mirror
IE is structurally a **UF-scoped** document, exactly like RG (which already ships SP/RJ).
Mirror `rg.go`.

- The capability interface IE must implement, alongside `Document` (`document.go`):
  ```go
  // document.go (declared in the foundation)
  type UFScoped interface {
      ValidateUF(value string, uf UF) (bool, error)
      ImplementedUFs() []UF
  }
  type Document interface {
      Kind() Kind
      Validate(value string) bool
      Generate() string
      Format(value string) (string, error)
  }
  ```
- RG is the working exemplar (read `rg.go` in full before starting). Its shape:
  ```go
  // rg.go — the template to copy for ie.go
  type RG struct{}
  func init() { Register(&RG{}) }
  func (r *RG) Kind() Kind { return KindRG }
  var rgImplemented = map[UF]bool{UFSP: true, UFRJ: true}
  func (r *RG) ImplementedUFs() []UF { return []UF{UFSP, UFRJ} }
  func (r *RG) ValidateUF(value string, uf UF) (bool, error) {
      if !rgImplemented[uf] {
          return false, fmt.Errorf("%w: %s", ErrUFNotImplemented, uf)
      }
      // ... parse + check-digit ...
  }
  func (r *RG) Validate(value string) bool { // tries each implemented UF
      for _, uf := range r.ImplementedUFs() {
          if ok, err := r.ValidateUF(value, uf); err == nil && ok { return true }
      }
      return false
  }
  // Generate(), Format() also implemented; ErrUFNotImplemented from errors.go
  ```
- `Kind` constants live in `document.go` (`KindCPF`…`KindPIX`). **There is no `KindIE` yet** —
  this plan adds it.
- Registry: types self-register in `init()` via `Register(d Document)` (`registry.go`); the
  CLI (`cmd/selo/kindcmd.go`) and MCP (`mcp/server.go`) derive their surfaces from
  `Kinds()`, and the `--uf` CLI flag is already wired for `UFScoped` kinds (that's how RG
  gets `selo rg --uf SP`). **So once IE is registered, CLI + MCP pick it up for free.**
- `compat/` mirrors `paemuri/brdoc`, which has **no** IE — so **do NOT add IE to `compat/`**.
- UF constants + `UF.Valid()` live in `uf.go` (all 27 present).

## Recommended first batch
The five highest-population states (covers the large majority of real IE numbers), in priority order:
**SP, MG, RJ, RS, PR.** SP is the canonical, best-documented IE algorithm and the right one to prototype first. Implement only the UFs you can verify (see deliverable rules).

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Tests | `go test ./... -run IE` | all pass |
| Full tests | `go test ./...` | all `ok` |
| Format | `gofmt -w ie.go ie_test.go` then `gofmt -l ie.go ie_test.go` | second prints nothing |
| Web research | host web-search tool, or `ctx_fetch_and_index(url, source)` + `ctx_search(queries)` | (raw curl/wget/WebFetch are blocked in this repo) |

## Scope
**In scope** (create/modify only these):
- `document.go` — add the `KindIE` constant (one line in the existing `Kind` const block) and its `String()` coverage if there's a switch (there isn't — `String()` returns `string(k)`).
- `ie.go` (create) — the `IE` type, per-UF algorithm table, first-batch implementations.
- `ie_test.go` (create) — table tests with **sourced real samples** per implemented UF.
- `docs/IE-NOTES.md` (create) — research log, sources, per-UF status, open questions, remaining-UF TODO.
- `docs/BACKLOG.md` — update the Inscrição Estadual entry with what shipped vs. what remains.

**Out of scope** (do NOT touch):
- `compat/` — paemuri has no IE; adding it there breaks the drop-in contract.
- `rg.go` and every other document type.
- CLI/MCP files — they auto-derive from the registry; no edits needed (verify, don't edit).
- Attempting UFs you cannot verify — they go in the TODO, not the code.

## Git workflow
- Branch: `advisor/006-inscricao-estadual` off the current branch.
- Conventional commits, e.g. `feat(ie): scaffold Inscrição Estadual + SP/MG/RJ/RS/PR`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Research the first-batch algorithms (the real work)
For each of SP, MG, RJ, RS, PR, find an **authoritative** description (state Fazenda/Sefaz
docs, or a well-cited, source-citing implementation) of: the IE length(s)/format(s), the
weight vector(s), the mod (usually 11, sometimes 10), and the check-digit rule(s). Capture
**≥2 real, verifiable IE numbers per UF** with their check digits. Note where a UF has
**multiple valid formats** (SP, for instance, has a standard 12-digit form and a separate
rural-producer form — decide whether the spike covers one or both, and say so).

Write everything into `docs/IE-NOTES.md` as you go: per UF — source URL(s), the algorithm,
the sample numbers, and a `status: ready | needs-research`. A UF without an authoritative
source and ≥2 samples is `needs-research` and is **not** implemented (STOP condition).

**Verify**: `docs/IE-NOTES.md` exists with a per-UF section for all five, each marked
`ready` or `needs-research`, sources cited.

### Step 2: Add the `KindIE` constant
In `document.go`, add to the `Kind` const block:
```go
	KindIE Kind = "ie" // Inscrição Estadual (state tax registration)
```
**Verify**: `go build ./...` → exit 0; `grep -n 'KindIE' document.go` → one match.

### Step 3: Scaffold `ie.go` (mirror `rg.go`)
Create `ie.go` (`package selo`) with:
- `type IE struct{}`, `NewIE() *IE`, `func (e *IE) Kind() Kind { return KindIE }`.
- `func init() { Register(&IE{}) }`.
- A per-UF algorithm table — a `map[UF]ieAlgo` where `ieAlgo` is a small struct or
  closure that knows how to validate (and, where feasible, format) that UF's IE. Each
  entry is populated **only** for `ready` UFs from step 1.
- `ImplementedUFs()` returns the sorted keys of that table.
- `ValidateUF(value string, uf UF) (bool, error)`: `ErrUFNotImplemented` (wrapped, like RG)
  when `uf` isn't in the table; else run that UF's validator. Malformed input for a
  supported UF → `(false, ErrInvalidFormat)`.
- `Validate(value string) bool`: try each implemented UF (mirror RG; first match wins).
- `Generate() string`: pick a random implemented UF and construct a valid IE for it **only
  if** you implemented constructive generation for that UF; otherwise it is acceptable for
  this spike to generate for the subset that supports it and document the limitation in
  `IE-NOTES.md`. (Generation per arbitrary state format is genuinely hard — do not fake it.)
- `Format(value string, ...)`: per-UF mask where one exists; identity otherwise. (IE masks
  vary; identity is an acceptable spike default — document it.)

Reuse `onlyDigits` (helpers.go) and the sentinels in `errors.go`. Match `rg.go`'s style.

**Verify**: `go build ./...` → exit 0; `go vet ./...` → exit 0.

### Step 4: Tests with sourced real samples
Create `ie_test.go` (model on `rg_test.go`). For each implemented UF: a table asserting the
**sourced real samples** validate `true`, plus an off-by-one/wrong-check sample → `false`,
a wrong-length → `false`, and `ValidateUF(x, <unimplemented UF>)` → `errors.Is(err,
ErrUFNotImplemented)`. If `Generate` is implemented for a UF, add a Generate→Validate
round-trip for it.

**Verify**: `go test ./... -run IE` → all pass, including the real-sample cases.

### Step 5: Confirm the registry/CLI/MCP picked IE up automatically (no edits)
**Verify** (read-only — do not edit CLI/MCP):
- `go test ./...` → all `ok` (registry/CLI/MCP tests still green with the new kind present).
- `go run ./cmd/selo ie --uf SP --validate <a-sourced-SP-sample>` → prints `valid` and exits 0.
- `go run ./cmd/selo ie --uf AC --validate 123` → exits non-zero / reports unimplemented
  (whatever the existing `--uf`/error path does for an unimplemented UF). If the CLI does
  **not** expose `ie` or `--uf` for it, that's a real integration gap — STOP and report
  (it means `UFScoped` wiring needs a fix, which is out of this plan's scope).

### Step 6: Write up status + update backlog
Finish `docs/IE-NOTES.md`: implemented UFs, deferred UFs (with what's missing), generation/
format limitations, and a checklist of the remaining 22 UFs as the follow-up roadmap.
Update the Inscrição Estadual entry in `docs/BACKLOG.md` to "first batch shipped (SP/MG/RJ/
RS/PR or subset) — see docs/IE-NOTES.md; N UFs remaining."

**Verify**: both docs updated; `grep -n 'IE-NOTES' docs/BACKLOG.md` → match.

## Test plan
- `ie_test.go`: per implemented UF — real-sample valids, wrong-check invalid, wrong-length
  invalid, unimplemented-UF error; round-trip where Generate exists. Structure mirrors
  `rg_test.go`.
- Verification: `go test ./...` → all pass.

## Done criteria
ALL must hold:
- [ ] `docs/IE-NOTES.md` exists: per-UF algorithm + sources + ≥2 real samples for every UF marked `ready`; deferred UFs listed with what's missing.
- [ ] `KindIE` added to `document.go`; `ie.go` implements `Document` + `UFScoped` and self-registers.
- [ ] At least **SP** is implemented and validates its sourced real samples (`go test ./... -run IE`). (More UFs if verified; none you couldn't verify.)
- [ ] `ValidateUF(x, <unimplemented UF>)` returns an error matching `errors.Is(err, ErrUFNotImplemented)`.
- [ ] `go build ./... && go vet ./... && go test ./...` → all `ok`.
- [ ] `gofmt -l ie.go ie_test.go` prints nothing.
- [ ] `compat/` unchanged (`git status`); CLI/MCP source unchanged (auto-derived).
- [ ] `docs/BACKLOG.md` updated; `plans/README.md` status row updated.

## STOP conditions
Stop and report (do not improvise) if:
- You cannot find an authoritative algorithm **and** ≥2 real sourced samples for a UF — do
  **not** implement it from an unverified blog snippet, and **never invent a sample to make
  a test pass** (that re-creates the exact circularity plan 004 exists to prevent). Mark it
  `needs-research` and continue with the UFs you can verify. Shipping SP alone, verified, is
  a success; shipping five wrong ones is a failure.
- The CLI/MCP do **not** automatically expose `ie` once registered (step 5) — report the
  integration gap; fixing `UFScoped` wiring is out of scope here.
- Implementing IE requires changing `document.go` beyond adding `KindIE`, or touching any
  type other than IE.
- `go test ./...` regresses on an existing package after adding the kind.

## Maintenance notes
- This is a **spike**: the deliverable is a correct, verified *subset* plus a structured plan
  for the rest — not all 27 UFs. The remaining UFs are follow-up plans, one (or a few) per
  batch, each gated by the same "authoritative source + real samples" rule.
- Several UFs accept **multiple IE formats**; the `ieAlgo` table should be shaped to allow a
  UF to have more than one acceptable form. Note per-UF format coverage in `IE-NOTES.md`.
- Generation and formatting are genuinely hard for arbitrary state formats — partial coverage
  is fine and expected; document exactly what's covered so callers aren't surprised.
- Reviewer: scrutinize the **provenance of the sample numbers**, not just that tests pass —
  as with RG (plan 004), internally-consistent code can validate fake numbers while rejecting
  real ones. The samples are the ground truth.
- Once IE is mature, revisit whether `GenPerson` (`person.go`) should include an IE field for
  the person's UF (it currently does not) — a natural follow-up.
