# ADR-0002: paemuri/brdoc compatibility subpackage

- **Status:** Accepted
- **Date:** 2026-06 (documented 2026-06-19, retroactive for v1.0.0)
- **Context tags:** migration, public API

## Context
`paemuri/brdoc` is the established Brazilian-document validation library in the Go ecosystem. It
exposes a flat, validation-only API (`IsCPF`, `IsCNPJ`, `IsCEP`, `IsRG`, …). `selo` offers a broader,
differently-shaped API (interface + registry, plus generation/formatting/geolocation). For `selo` to
be a credible replacement, existing `paemuri/brdoc` users need a migration path that does not require
rewriting call sites — otherwise the switching cost blocks adoption.

## Decision
Ship a **`compat` subpackage** that mirrors `paemuri/brdoc` v3's public `Is*` surface with identical
signatures, so migration is a one-line import swap:

```go
import "github.com/inovacc/selo/compat" // was: github.com/paemuri/brdoc/v3
```

- `compat` wrappers delegate to the core `selo` types; they do **not** reimplement algorithms.
- A **compile-time signature-parity guard** keeps the wrapper signatures aligned with the upstream
  API, so accidental drift fails the build.
- `compat` deliberately exposes **only** what paemuri/brdoc exposes. New `selo`-only capabilities
  (e.g. Inscrição Estadual) are **not** added to `compat` — adding them would break the drop-in
  contract and the parity guarantee.

## Consequences
**Positive**
- Zero-rewrite migration for `paemuri/brdoc` users (import swap only).
- The parity guard prevents silent divergence from the upstream shape.
- Core `selo` stays free to evolve its richer API without constraining the compat surface.

**Negative / trade-offs**
- The compat surface is intentionally frozen to paemuri's shape, so it cannot grow with selo
  (callers wanting IE, generation, etc. must use the core API). This is by design and documented.
- Maintaining parity requires watching upstream for API changes; the guard catches our side, not
  upstream additions.

## Alternatives considered
- **No compat layer (docs-only migration guide)** — rejected: higher switching cost, slower adoption.
- **Make the core API match paemuri exactly** — rejected: would forfeit the interface/registry design
  (ADR-0001) and the generate/format/geolocate capabilities that differentiate `selo`.
