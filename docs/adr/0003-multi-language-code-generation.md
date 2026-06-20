# ADR-0003: Multi-language code generation from a single Go source of truth

- **Status:** Accepted (2026-06-20)
- **Supersedes:** —
- **Related:** [ADR-0001 (interface + registry architecture)](0001-interface-registry-architecture.md),
  [docs/CODEGEN.md](../CODEGEN.md)

## Context

selo's check-digit, format, origin, and generation algorithms for Brazilian documents are verified
against authoritative sources and pinned by tests. Those algorithms are valuable beyond Go —
TypeScript, JavaScript, Ruby, Java, C#, Python, PHP, and Rust projects all need the same validators —
but hand-porting them to each language invites **drift**: a subtle weight or modulo difference
produces a validator that silently accepts invalid (or rejects valid) documents, and there is no
single place that guarantees all ports agree.

We wanted other-language validators that stay correct as the Go algorithms evolve, without turning
selo into eight hand-maintained codebases.

## Decision

Generate the other-language code from the Go implementation, with the Go library as the single
source of truth:

- **`internal/codegen`** holds a declarative per-kind spec (`spec.go`: `KindPlan`, `CheckDigit`,
  `DVRule`), shared data tables (`data.go`: CEP ranges, DDD→UF), and **golden vectors**
  (`vectors.go`) produced by running the *live* selo library.
- **Per-language emitters** (`emit_<lang>*.go` + `templates/<lang>/`) render an installable module
  plus a runnable test suite for each target. A new language is one new emitter set plus a one-line
  registration in `supportedLangs`.
- **`selo gen`** (CLI) and the **`generate_code`** MCP tool expose generation; both derive their
  language list from `supportedLangs`, so adding a language widens both surfaces with no extra edits.
- **Golden snapshot tests** (`golden_<lang>_test.go`) assert that re-emitting reproduces the
  committed `generated/<lang>/` reference byte-for-byte (deterministic files) and that every committed
  vector still agrees with the live library.
- **A CI matrix** (`.github/workflows/codegen.yml`) runs each target's emitted test suite on that
  language's real toolchain (node / ruby / JDK+Maven / .NET / Python / PHP / Cargo). A wrong port
  fails its own vector tests there.

The current targets are TypeScript, JavaScript, Ruby, Java, C#, Python, PHP, and **Rust** (8) —
Rust emits a Cargo library crate with golden-vector tests and a CI cargo lane (added in v1.6.0).

## Consequences

**Positive**
- One verified source of truth; ports cannot silently diverge — the golden vectors are the contract.
- Adding a kind propagates to every language via its `KindPlan`; adding a language is isolated to one
  emitter set.
- The committed `generated/<lang>/` trees are usable references and are continuously proven correct.

**Negative / trade-offs**
- Each language needs an emitter and templates — real per-language maintenance, and idiomatic output
  requires care (naming, error idioms, package layout).
- Generated code is a **snapshot**: changing a Go algorithm requires regenerating and re-committing
  the references (the golden test enforces this).
- The emitters must be deterministic (sorted map iteration) or the golden snapshot becomes flaky.

## Alternatives considered

- **Hand-written ports per language** — rejected: maximal drift risk, no single contract, N× the
  review burden.
- **A runtime/WASM core shared across languages** — rejected: heavyweight to consume, poor idiomatic
  fit, and it hides the algorithm behind a binary instead of producing readable, auditable code.
- **Publish only a spec document** — rejected: a prose spec is not executable and cannot be
  CI-verified against the Go source.
