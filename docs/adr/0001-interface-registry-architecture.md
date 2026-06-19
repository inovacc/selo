# ADR-0001: Document interface + self-registering registry

- **Status:** Accepted
- **Date:** 2026-06 (documented 2026-06-19, retroactive for v1.0.0)
- **Context tags:** core architecture

## Context
`selo` must support many Brazilian document types (CPF, CNPJ, CNH, PIS, RENAVAM, voter ID, CEP,
phone, plate, CNS, RG, IE, PIX) across three surfaces — a Go library, a Cobra CLI, and an MCP
server — and keep them in lock-step. Some types are plain (CPF), some resolve a geographic origin
(CEP, phone, voter ID, CPF region), and some depend on a federative unit (RG, IE). A naïve design
would hard-code each type into every surface (a `switch` in the CLI, another in the MCP server,
another for detection), so adding a type would mean editing many files and risking drift between
what the library supports and what the CLI/MCP expose.

## Decision
1. Define a minimal **`Document` interface** — `Kind()`, `Validate()`, `Generate()`, `Format()` —
   that every type implements.
2. Express variability through **optional capability interfaces** discovered by type assertion:
   `OriginResolver` (`Origin`) and `UFScoped` (`ValidateUF`, `ImplementedUFs`). A type opts in by
   implementing the interface; consumers feature-detect at runtime.
3. Provide a **self-registering registry**: each type calls `Register(&T{})` in its `init()`, and
   the package exposes `Get`, `Kinds`, and dispatch helpers `Validate`/`Generate`/`Format` plus
   `Detect`.
4. The **CLI and MCP server derive their entire surface from `Kinds()`** — one CLI subcommand per
   kind, MCP tool enums sourced from the registry, the `--uf` flag wired for `UFScoped` kinds.

## Consequences
**Positive**
- Adding a document type is one new file (`>kind<.go`) + a `Kind` constant + `Register` in `init()`.
  The CLI and MCP pick it up automatically — no edits there (verified with the IE addition: zero
  CLI/MCP changes).
- One source of truth; the surfaces cannot drift from the library's capabilities.
- Capabilities stay optional and explicit; a plain type isn't forced to stub `Origin`/`ValidateUF`.

**Negative / trade-offs**
- Registration is a global side effect at `init()` time (import order independence required; the
  registry must be complete before `main`). Acceptable for a stateless validation library.
- Type assertion for capabilities is checked at runtime, not the compiler — mitigated by
  compile-time conformance vars (`var _ UFScoped = (*RG)(nil)`) in tests.
- `Generate()` returns masked output for UI consistency, diverging slightly from the interface
  doc's "unformatted" wording (documented in ISSUES.md).

## Alternatives considered
- **Per-surface switch statements** — rejected: O(types × surfaces) edits, guaranteed drift.
- **One fat interface with every capability** — rejected: forces meaningless stubs (`Origin` on CPF
  check-digit-only types) and couples unrelated concerns. Optional interfaces are cleaner.
