# BACKLOG

Future work and tech debt for the `brdoc` toolkit. Items are grouped by type;
`DEPRECATION` items carry a removal date per the project's deprecation policy.

---

## Planned Features

### ✅ SHIPPED: Fake Person Generator — `GenPerson` (synthetic Brazilian identity)

**Goal:** Generate a complete, internally **consistent** fake Brazilian person that
carries *every* document type this toolkit supports, so it can be used as
high-fidelity test data (QA, DB seeding, demos, form testing, fixtures). This is the
natural capstone of a "generate + validate + format" toolkit: instead of generating
isolated valid documents, it generates one coherent identity whose documents agree
with each other.

**Why it's distinctive:** any library can emit a random valid CPF. The value here is
**cross-document coherence** — the documents of one person must be mutually
consistent, not just individually valid:
- The **CPF region digit** (9th digit) must match the person's home region.
- The **Título Eleitoral** embedded UF code must match the person's state.
- The **phone DDD** must map to the person's state.
- The **CEP** must fall in the person's state's postal range.
- The **PIX keys** (CPF-key, phone-key, email-key) must reference the same person's
  CPF / phone / email.
- An optional **vehicle** (plate + RENAVAM) and **company** (CNPJ) are linked to the
  person but are separate entities.

That UF-consistency engine is the real work; the per-document generators already exist
(`Generate()` on every type) and the `UF` tables (CPF region, CEP ranges, DDD→UF,
voter UF codes) are already in the package.

**Proposed API (root package):**
```go
type Person struct {
    Name       string   // pt-BR given + surname (from embedded name lists)
    BirthDate  string   // ISO-8601; adult by default
    UF         UF       // home federative unit; drives all geo-consistent docs
    CPF        string
    RG         string   // SP only until multi-state RG ships (see v2)
    CNH        string
    PIS        string
    VoterID    string   // Título Eleitoral, UF code == UF
    CNS        string
    CEP        string   // within UF's range
    Phone      string   // DDD maps to UF
    Email      string
    PIXKeys    []string // CPF-key, phone-key (E.164), email-key, and an EVP
    Vehicle    *Vehicle // optional: Plate + RENAVAM
    Company    *Company // optional: CNPJ (+ company name)
}

func GeneratePerson(opts ...PersonOption) Person

// Options: WithUF(uf), WithSeed(n) (deterministic fixtures), WithVehicle(),
// WithCompany(), Formatted(bool) (masked vs raw documents).
```

**Surfaces:**
- **Library:** `brdoc.GeneratePerson(...)` returning the struct.
- **CLI:** `brdoc person [--count N] [--uf SP] [--seed 42] [--json] [--formatted] [--with-vehicle] [--with-company]` — JSON array is the default machine-friendly output; a human table for `--count 1`.
- **MCP tool:** `generate_person` (count, uf?, withVehicle?, withCompany?) → structured Person(s). Slots into the existing registry-derived tool set.

**Design notes / decisions to make during planning:**
- **Determinism:** support `WithSeed` (use `math/rand/v2.NewPCG`/`NewChaCha8` with a fixed seed) so test fixtures are reproducible — the default registry generators use the global goroutine-safe RNG; the seeded path needs an explicit `*rand.Rand` threaded through generation.
- **UF-consistency engine:** generate the `UF` first, then drive each geo-document from it (CPF region digit, voter UF code, a DDD from that UF, a CEP in that UF's range). Some documents (CNH, PIS, CNS, RENAVAM) have no geo component — generate freely.
- **Names/addresses:** start with embedded pt-BR given-name + surname lists (small, vendored, no external dep). Street-level address realism is a stretch goal; the core deliverable is the *documents* + UF consistency.
- **LGPD / ethics:** output MUST be clearly synthetic. Document prominently that this generates **fake** data for testing only — never real PII. Consider a deterministic "fake marker" or keeping CPFs in test-reserved ranges if any such convention exists.
- **Scope guard (YAGNI):** v1 of GenPerson should cover the document set + UF consistency + seed. Photorealistic names/addresses, age-distribution modeling, and locale variants are follow-ups.

**Effort:** M (the per-doc generators and UF tables already exist; the work is the
consistency engine, the `Person`/options API, and the 3 surfaces + tests).
**Value:** H (flagship test-data feature; no Go BR-doc library offers a coherent
multi-document fake-person generator).

**Status:** SHIPPED across library (`GeneratePerson`), CLI (`selo person`), and MCP
(`generate_person`), with UF-consistency verified across all 27 UFs.

**Reproducible output — `WithSeed`: ✅ SHIPPED (v1.3.0).** Every document type now implements
`RandGenerator` (`GenerateRand(*math/rand/v2.Rand)`), the registry exposes `GenerateRand(kind, r)`,
and `GeneratePerson` accepts `WithSeed(int64)` / `WithRand(*rand.Rand)` for deterministic fixtures.
**Remaining:** expose `--seed` at the CLI (`selo person`) and MCP (`generate_person`) surfaces
(v1.4.0).

**Depends on:** nothing blocking; multi-state RG (v2) would let `Person.RG` cover all
states instead of SP/RJ.

---

## v2 Document Types (deferred from v1 scope)

- **Inscrição Estadual** — per-UF state tax registration; 27 distinct algorithms.
  Biggest open gap in the Go ecosystem (paemuri/brdoc issue #7, never shipped). Land
  incrementally behind the same `Document`/`UFScoped` pattern used by RG.
  **First batch shipped 2026-06-19 (plan 006): SP only** — verified algorithm + 2 sourced
  samples (`ie.go`, `TestIE_AuthoritativeSamples`); CLI/MCP auto-derive it.
  **Second batch shipped 2026-06-20 (v1.6.0): MG, RS, PR** — authoritative,
  adversarially-verified algorithms (official SINTEGRA-MG/RS and SEFA-PR worked examples +
  independent reference-impl corroboration); `GeneratePerson` carries a UF-consistent IE for them
  too. **RJ re-researched and kept BLOCKED** — its official page omits the weight vector and no ≥2
  verifiable samples were obtainable. **22 UFs remaining** (RJ blocked; the rest unstarted).
  Architecture, sources, and the per-UF roadmap are in `docs/IE-NOTES.md`. **Value: H, Effort: L
  (remaining).**
- **Codegen IE parity** — the multi-language codegen IE emitter is still **SP-only**, while the Go
  library now validates SP/MG/RS/PR. Emitting MG/RS/PR in the generated targets needs a digit-sum DV
  rule in the codegen spec (their algorithms use cross-digit/over-9 sums the current `DVRule` doesn't
  model). Documented follow-up. **Value: M, Effort: M.**
- **Multi-state RG** — extend `rg.go` beyond SP wherever per-UF check-digit rules are documented;
  explicit `ErrUFNotImplemented` elsewhere. **RJ was removed in v1.3.0** (its algorithm differs
  from SP — see the update below); re-adding RJ or any other UF is blocked on an authoritative
  algorithm plus ≥2 verifiable samples. Unblocks `Person.RG` for more states.
  **Value: M, Effort: L (blocked on sources).**
  - RG SP/RJ convention **verified (A)** 2026-06-19 (plan 004): the implemented
    `DV = 11 - (sum mod 11)` with ascending weights 2..9 (10→'X', 11→'0') matches four
    independent sources — NG Matemática and "Tudo em AdvPL"/siga0984 state it verbatim;
    Bóson Treinamentos and dev.to/shadowlik state the algebraically-equivalent descending
    form (weights 9..2, `DV = sum mod 11`). Real samples pinned in
    `TestRG_AuthoritativeSamples` (`24.678.131-2`, `29.465.327-2`). Caveat: the **SP**
    algorithm is what's verified; the code applies the *same* algorithm to **RJ**, but no
    independent RJ-specific source was found — confirm RJ before relying on it or building
    multi-state RG on top.
  - **Update (item 8, 2026-06-19):** research indicates **RJ uses a different RG algorithm than
    SP** (an SP-valid RG can be invalid under RJ rules), so the current RJ reuse of the SP algorithm
    is **likely incorrect**. No authoritative SSP-RJ algorithm or verifiable samples were
    obtainable. Fix path: source the SSP-RJ spec + ≥2 real samples, or demote `UFRJ` to
    `ErrUFNotImplemented` until verified. See ISSUES.md. **IE update (v1.6.0):** MG/RS/PR shipped
    (authoritative algorithms + verified samples). **RJ IE stays blocked** — its DV rule was found
    (mod 11, remainder ≤1→0 else 11−remainder) but its official page omits the weight vector and ≥2
    verifiable samples were not obtainable.
- **RNM (Registro Nacional Migratório)** — the foreigner ID on the CRNM card. **Researched
  2026-06-19, deferred** (owner's call) for lack of a verifiable spec: the RNM is an *opaque*
  alphanumeric sequence "derived from personal data + fingerprints" with **no public format or
  check-digit algorithm** ([Polícia Federal](https://www.gov.br/pf/pt-br/assuntos/imigracao/duvidas-frequentes/autorizacao-de-residencia-e-registro-nacional-migratorio-rnm/o-que-e-registro-nacional)).
  The predecessor **RNE** is reportedly "8 digits + 1 check digit", but no authoritative
  weights/algorithm and no ≥2 verifiable samples were found. Per the project's verify-or-don't-ship
  discipline (cf. plans 004/006), not implemented — would require inventing a checksum. **To ship:**
  obtain an authoritative format + check-digit algorithm (or a trusted reference impl + real
  samples), then add `rnm.go` behind the `Document` interface; the new kind auto-propagates to
  CLI/MCP and needs a `KindPlan` + the 5 codegen emitters + CI-matrix verification.
  **Value: M, Effort: M (blocked on spec).**

---

## Hardening / Tech Debt

- **Go 1.25 requirement (release note, not debt).** Adding the MCP `go-sdk` (v1.6.1)
  bumped `go.mod`'s go directive from 1.24.0 → 1.25.0 (the sdk requires it). Documented as a
  consumer-visible minimum in the README and CHANGELOG.

## Resolved (shipped)

- **CNPJ all-zeros rejection — ✅ v1.1.0.** `CNPJ.Validate` rejects all-equal inputs
  (e.g. `00000000000000`) via `allEqualBytes` (`cnpj.go`), matching CPF.
- **`scanBuf` data race — ✅ v1.2.0.** The shared package-level scanner buffer was made
  call-local in `cmd/selo/iohelper.go`.
- **`.gitattributes` / CRLF — ✅ v1.2.0.** `* text=auto eol=lf` enforces LF repo-wide, settling
  the local-vs-CI `gofmt` divergence on Windows.
- **`golangci-lint` CI gate — ✅ v1.2.0.** CI runs the full `default: all` lint via the reusable
  workflow (84 pre-existing issues cleared). It may still be absent in some dev environments;
  install locally for parity (`go vet` is the local fallback).
- **CLI `--uf` "not implemented" message — ✅ v1.2.0.** UF-scoped kinds (RG/IE) now surface the
  real reason instead of a bare "invalid".
- **Per-call `SilenceUsage` unified — ✅ v1.4.0.** Removed the redundant per-call
  `cmd.SilenceUsage = true` assignments in `detect`/`format`/`origin`; the root command's flag
  governs (Cobra inherits it).

---

## DEPRECATION

- **`ValidateDocument(doc string) (string, bool)`** — superseded by `Detect` + `Validate`.
  Marked `// Deprecated:` in `cpf.go` (was `brdoc.go` before plan 005's split). **Removal:
  after 2026-07-18** (≥30 days). Remove in a dedicated cleanup commit (not mixed with
  features) once the date passes.
