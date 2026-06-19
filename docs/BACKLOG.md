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
    RG         string   // SP/RJ only until multi-state RG ships (see v2)
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

**Remaining enhancement — `WithSeed` (reproducible output):** the per-document
generators use the goroutine-safe global `math/rand/v2`, which cannot be seeded
per-call. Deterministic fixtures need the generators to accept a `*rand.Rand` source
(or a parallel seeded construction path). Deferred. **Value: M, Effort: M.**

**Depends on:** nothing blocking; multi-state RG (v2) would let `Person.RG` cover all
states instead of SP/RJ.

---

## v2 Document Types (deferred from v1 scope)

- **Inscrição Estadual** — per-UF state tax registration; 27 distinct algorithms.
  Biggest open gap in the Go ecosystem (paemuri/brdoc issue #7, never shipped). Land
  incrementally behind the same `Document`/`UFScoped` pattern used by RG.
  **First batch shipped 2026-06-19 (plan 006): SP only** — verified algorithm + 2 sourced
  samples (`ie.go`, `TestIE_AuthoritativeSamples`); CLI/MCP auto-derive it. **26 UFs
  remaining** (MG/RJ/RS/PR researched but deferred for lack of ≥2 verifiable samples; the
  rest unstarted). Architecture, SP sources, and the per-UF roadmap are in
  `docs/IE-NOTES.md`. **Value: H, Effort: L (remaining).**
- **Multi-state RG** — extend `rg.go` beyond SP/RJ wherever per-UF check-digit rules are
  documented; explicit `ErrUFNotImplemented` elsewhere (paemuri issue #22 ships only
  SP/RJ). Unblocks `Person.RG` for all states. **Value: M, Effort: L.**
  - RG SP/RJ convention **verified (A)** 2026-06-19 (plan 004): the implemented
    `DV = 11 - (sum mod 11)` with ascending weights 2..9 (10→'X', 11→'0') matches four
    independent sources — NG Matemática and "Tudo em AdvPL"/siga0984 state it verbatim;
    Bóson Treinamentos and dev.to/shadowlik state the algebraically-equivalent descending
    form (weights 9..2, `DV = sum mod 11`). Real samples pinned in
    `TestRG_AuthoritativeSamples` (`24.678.131-2`, `29.465.327-2`). Caveat: the **SP**
    algorithm is what's verified; the code applies the *same* algorithm to **RJ**, but no
    independent RJ-specific source was found — confirm RJ before relying on it or building
    multi-state RG on top.

---

## Hardening / Tech Debt

- **CNPJ accepts all-zeros.** `CNPJ.Validate("00000000000000")` returns `true` (all-zero
  input is mathematically check-digit-valid and CNPJ has no all-equal guard, unlike CPF
  which rejects repeated-digit inputs via `notAcceptedCPF`). Add a symmetric all-equal /
  all-zeros rejection to `CNPJ.Validate` for parity with CPF. **Value: M, Effort: S.**
  (Note: this is a behavior change — gate it deliberately and update the regression tests.)
- **`scanBuf` shared package-level buffer** in `cmd/brdoc/iohelper.go` is passed to every
  `streamValidate` call; safe today (CLI invokes it serially) but a latent data race if it
  were ever called concurrently. Make the buffer call-local. **Value: L, Effort: S.**
- **`golangci-lint` gate not enforced locally** — the tool is not installed in the dev
  environment, so M2C-4/M5-4 lint gates were satisfied by `go vet` only. Ensure CI runs
  `golangci-lint run --fix ./... --timeout=5m`. **Value: M, Effort: S.**
- **Per-call `SilenceUsage` inconsistency** — `runFormat`/`runOrigin` set
  `cmd.SilenceUsage = true` per-call while `runValidate`/`runFrom` rely on the root-level
  flag. Cosmetic; pick one approach. **Value: L, Effort: S.**
- **Go 1.25 requirement (release note, not debt).** Adding the MCP `go-sdk` (v1.6.1)
  bumped `go.mod`'s go directive from 1.24.0 → 1.25.0 (the sdk requires it). Document this
  minimum-Go bump in the README/release notes; it is a consumer-visible requirement.

---

## DEPRECATION

- **`ValidateDocument(doc string) (string, bool)`** — superseded by `Detect` + `Validate`.
  Marked `// Deprecated:` in `brdoc.go`. **Removal: after 2026-07-18** (≥30 days). Remove
  in a dedicated cleanup commit (not mixed with features) once the date passes.
