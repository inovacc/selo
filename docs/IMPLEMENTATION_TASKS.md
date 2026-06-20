# Implementation Tasks

Granular tasks for the planned/proposed items in [ROADMAP.md](ROADMAP.md), [BACKLOG.md](BACKLOG.md),
and [FEATURES.md](FEATURES.md), grouped by domain. Effort: **S** ≤½ day, **M** ≈1–2 days,
**L** multi-day. (Advisor plans 001–006 plus the v1.2.0–v1.4.0 work — multi-language codegen,
seedable generation, hygiene/DX — are done; Domains 3, 5, 6 below are ✅ complete and kept for
reference. The open work is IE breadth (Domain 1), multi-state RG (Domain 2), and the
calendar-gated deprecation (Domain 4).)

## Domain 1 — Inscrição Estadual breadth
Each UF is gated by an authoritative algorithm + ≥2 independently-sourced samples (see
[IE-NOTES.md](IE-NOTES.md)); never invent a sample.

| ID | What | Files | Env | Depends on | Effort |
|----|------|-------|-----|------------|--------|
| 1.1 | Research + verify MG IE (algorithm + ≥2 samples) | `docs/IE-NOTES.md` | research | — | M |
| 1.2 | Implement MG in `ieTable` + tests | `ie.go`, `ie_test.go` | Go | 1.1 | S |
| 1.3 | Research + verify RJ IE | `docs/IE-NOTES.md` | research | — | M |
| 1.4 | Implement RJ + tests | `ie.go`, `ie_test.go` | Go | 1.3 | S |
| 1.5 | Repeat 1.1–1.4 for RS, PR (finish first batch) | same | Go/research | — | M |
| 1.6 | Batch the remaining 22 UFs (one plan per few UFs) | same | Go/research | 1.1–1.5 | L |
| 1.7 | Per-UF mask + constructive `generate` where feasible | `ie.go` | Go | per-UF impl | M |

## Domain 2 — Multi-state RG
| ID | What | Files | Env | Depends on | Effort |
|----|------|-------|-----|------------|--------|
| 2.1 | ✅ **Done (v1.3.0):** verified RJ differs from SP → RJ demoted to `ErrUFNotImplemented` | `rg.go`, `rg_test.go` | research | — | done |
| 2.2 | Generalize `rgImplemented`/`checkDigit` to a per-UF table (mirror `ieTable`) | `rg.go` | Go | 2.1 | M |
| 2.3 | Add documented UFs with sourced samples | `rg.go`, `rg_test.go` | Go/research | 2.2 | L |
| 2.4 | Let `Person.RG` cover the person's UF | `person.go`, `person_test.go` | Go | 2.2 | S |

## Domain 3 — Reproducible generation — ✅ COMPLETE (v1.3.0–v1.4.0)
Shipped: every type implements `RandGenerator` (`GenerateRand(*rand.Rand)`) and the registry exposes
`GenerateRand(kind, r)` (3.1–3.2); `GeneratePerson` has `WithSeed`/`WithRand` (3.3); determinism
tests pin same-seed→same-output (3.4); and `--seed` reaches the CLI/MCP surfaces (v1.4.0). Rows kept
for reference.

| ID | What | Files | Env | Depends on | Effort |
|----|------|-------|-----|------------|--------|
| 3.1 | Define a seedable source abstraction (accept `*rand.Rand`) | `document.go` / new `rand.go` | Go | — | M |
| 3.2 | Thread the source through each type's `Generate` | all `*.go` with `Generate` | Go | 3.1 | L |
| 3.3 | `GeneratePerson` `WithSeed(*rand.Rand)` option | `person.go`, `person_test.go` | Go | 3.1 | S |
| 3.4 | Deterministic-output tests (same seed → same output) | `*_test.go` | Go | 3.2, 3.3 | S |

## Domain 4 — Deprecation cleanup
| ID | What | Files | Env | Depends on | Effort |
|----|------|-------|-----|------------|--------|
| 4.1 | After 2026-07-18, remove `ValidateDocument` in a dedicated commit | `cpf.go`, `cpf_cnpj_test.go`, `doc.go` | Go | date passed | S |

## Domain 5 — Repo hygiene / DX — ✅ COMPLETE (v1.2.0–v1.4.0)
All shipped. Rows kept for reference.

| ID | What | Files | Env | Depends on | Effort |
|----|------|-------|-----|------------|--------|
| 5.1 | ✅ `.gitattributes` LF enforcement (v1.2.0) | `.gitattributes` | repo | — | done |
| 5.2 | ✅ `scanBuf` made call-local (v1.2.0) | `cmd/selo/iohelper.go` | Go | — | done |
| 5.3 | ✅ Distinct "UF not implemented" CLI message (v1.2.0) | `cmd/selo/kindcmd.go` | Go | — | done |
| 5.4 | ✅ Go 1.25 minimum documented (README/CHANGELOG) | `README.md` | docs | — | done |
| 5.5 | ✅ `mcp` coverage raised to 93.3% | `mcp/server_test.go` | Go | — | done |

## Domain 6 — Multi-language code generation — ✅ COMPLETE (v1.2.0–v1.4.0)
Shipped: `selo gen` + the `generate_code` MCP tool emit validate/format/origin/generate for all 13
kinds in TypeScript, JavaScript, Ruby, Java, C#, and Python; `internal/codegen` (spec + golden
vectors + per-language emitters), golden snapshot tests, committed `generated/<lang>/` references, and
a CI matrix verifying every target on its real toolchain. Open follow-up: more target languages
(tracked in FEATURES.md "Proposed").

## Suggested order (remaining)
1. **Domain 1** (IE breadth) — highest product value; each UF gated on an authoritative algorithm +
   ≥2 sourced samples (1.1→1.2, then 1.3→1.4, …).
2. **Domain 2** (multi-state RG) — 2.1 done (RJ demoted); 2.2–2.4 blocked on per-UF algorithms +
   verifiable samples.
3. **4.1** — calendar-gated deprecation removal (after 2026-07-18).

Domains 3, 5, 6 are complete. Cross-reference these IDs from ROADMAP.md / BACKLOG.md when scheduling.
