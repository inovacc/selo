# Implementation Tasks

Granular tasks for the planned/proposed items in [ROADMAP.md](ROADMAP.md), [BACKLOG.md](BACKLOG.md),
and [FEATURES.md](FEATURES.md), grouped by domain. Effort: **S** ≤½ day, **M** ≈1–2 days,
**L** multi-day. (Advisor plans 001–006 are already done; these are the remaining items.)

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
| 2.1 | Independently verify the RJ RG algorithm (currently assumed = SP) | `docs/BACKLOG.md`, `rg_test.go` | research | — | M |
| 2.2 | Generalize `rgImplemented`/`checkDigit` to a per-UF table (mirror `ieTable`) | `rg.go` | Go | 2.1 | M |
| 2.3 | Add documented UFs with sourced samples | `rg.go`, `rg_test.go` | Go/research | 2.2 | L |
| 2.4 | Let `Person.RG` cover the person's UF | `person.go`, `person_test.go` | Go | 2.2 | S |

## Domain 3 — Reproducible generation
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

## Domain 5 — Repo hygiene / DX
| ID | What | Files | Env | Depends on | Effort |
|----|------|-------|-----|------------|--------|
| 5.1 | Add `.gitattributes` (`*.go text eol=lf`, docs eol=lf) + `git add --renormalize .` | `.gitattributes` | repo | — | S |
| 5.2 | Make `scanBuf` call-local (latent data race) | `cmd/selo/iohelper.go` | Go | — | S |
| 5.3 | Distinct CLI message for unimplemented `--uf` (vs `invalid`) | `cmd/selo/kindcmd.go` | Go | — | S |
| 5.4 | Document Go 1.25 minimum in README/release notes | `README.md` | docs | — | S |
| 5.5 | Raise `mcp` error-path coverage (84.4%) | `mcp/server_test.go` | Go | — | S |

## Suggested order
1. **5.1** (gitattributes) — cheap, removes CRLF noise for everyone.
2. **Domain 1** (IE breadth) — highest product value; 1.1→1.2 then 1.3→1.4, etc.
3. **2.1** (verify RJ RG) — unblocks Domain 2; do before any multi-state RG code.
4. **Domain 3** (seedable) — cross-cutting; schedule when generator churn is acceptable.
5. **4.1** — calendar-gated (after 2026-07-18).
6. **5.2/5.3/5.5** — opportunistic, bundle with nearby changes.

Cross-reference these IDs from ROADMAP.md / BACKLOG.md when scheduling.
