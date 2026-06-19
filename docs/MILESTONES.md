# Milestones

Version milestones for `github.com/inovacc/selo`. Releases are tracked with git tags
(`git tag -l`). See [ROADMAP.md](ROADMAP.md) for phase-level status and [CHANGELOG.md](../CHANGELOG.md)
for the change log.

## v1.0.0 — Complete toolkit (tagged) — ✅ RELEASED
The full rebrand from `brdoc` to `selo` plus the complete Brazilian-document toolkit.

- **Goal:** library + CLI + MCP server covering the standard Brazilian documents, with
  generate/format/geolocate (not validation-only), and a paemuri/brdoc drop-in compat layer.
- Delivered:
  - 13 document kinds behind a `Document` interface + self-registering registry.
  - Cobra CLI (`cmd/selo`) deriving subcommands from the registry; scriptable exit codes.
  - stdio MCP server (`selo mcp`) with 6 registry-backed tools.
  - `compat` subpackage mirroring `paemuri/brdoc` v3 with a signature-parity guard.
  - `GeneratePerson` synthetic-identity generator.
- **Toolchain:** Go 1.25+ (the MCP `go-sdk` requires it).
- **Test Coverage:** 92.2% total (per-package: core 94.2%, compat 95.3%, cmd/selo 87.0%, mcp 84.4%).

## v1.1.0 — Hardening & first-state IE — ✅ MERGED to `main` (post-tag)
Quality and correctness work landed after v1.0.0 (advisor plans 001–006).

- CI now gates `main` (all branches + PRs); GUI build deps dropped.
- CNPJ rejects all-equal inputs (parity with CPF).
- MCP error prefix corrected to `selo`.
- `brdoc.go` god-file split into `cpf.go` + `cnpj.go`.
- RG SP/RJ check-digit convention verified against authoritative sources; samples pinned.
- **Inscrição Estadual** scaffolded; **SP shipped & verified** (CLI/MCP auto-derive it).
- **Test Coverage:** 92.2% total (unchanged target met).

## v1.2.0 — Breadth — 🔜 PLANNED
- **Goal:** broaden UF coverage and reproducibility.
- Candidate scope:
  - Inscrição Estadual next batch (MG, RJ, RS, PR) — each gated by an authoritative algorithm
    plus ≥2 verifiable samples.
  - Multi-state RG (verify RJ independently, then add documented UFs).
  - `GenPerson` seedable generation for deterministic fixtures.
- **Coverage target:** maintain ≥85% per package.

## v2.0.0 — Cleanup (breaking) — 🔮 FUTURE
- **Goal:** remove deprecations once their windows pass.
- Remove `ValidateDocument` (deprecated; removal after 2026-07-18) in a dedicated commit.
- **Coverage target:** ≥85%.
