# Milestones

Version milestones for `github.com/inovacc/selo`. Releases are tracked with git tags
(`git tag -l`). See [ROADMAP.md](ROADMAP.md) for phase-level status and [CHANGELOG.md](../CHANGELOG.md)
for the change log.

> **Versioning note:** the `v1.0.0` tag predates the rebrand and points at the original `brdoc`
> code (CPF/CNPJ only); at that commit the module path was `github.com/inovacc/brdoc`, so
> `github.com/inovacc/selo@v1.0.0` is **not** a valid module version. The first installable
> release of the `selo` module is **v1.1.0**. (Per Go module rules, the unsuffixed path
> `github.com/inovacc/selo` only supports v0/v1 tags; a future v2 would require a `/v2` path.)

## v1.0.0 — Original brdoc (CPF/CNPJ) — ✅ RELEASED (2025-11-19, tag `a5dfbaf`)
- Module: `github.com/inovacc/brdoc` (pre-rebrand).
- CPF and alphanumeric CNPJ: validate / generate / format; CPF region; auto-detect.
- Validation-focused, single-package library.

## v1.1.0 — Selo: complete toolkit + hardening — ✅ MERGED to `main` (2026-06-19)
First release under the `github.com/inovacc/selo` module path. Combines the rebrand, the complete
document toolkit, and the post-build hardening (advisor plans 001–006).

- **Goal:** a library + CLI + MCP server covering the standard Brazilian documents, with
  generate/format/geolocate (not validation-only), and a `paemuri/brdoc` drop-in compat layer.
- **Rebrand:** `brdoc` → `selo` (module `github.com/inovacc/selo`); minimum Go raised to 1.25
  (required by the MCP `go-sdk`).
- **Architecture:** `Document` interface + self-registering registry; optional `OriginResolver`
  and `UFScoped` capabilities.
- **13 document kinds:** CPF, CNPJ (alphanumeric + legacy), CNH, PIS/PASEP/NIS, RENAVAM,
  Título Eleitoral, CEP, phone, license plate, CNS, RG (SP/RJ), Inscrição Estadual (SP), PIX keys.
- **Surfaces:** Cobra CLI (`cmd/selo`, registry-derived; scriptable exit codes), stdio MCP server
  (6 tools), `compat` subpackage mirroring `paemuri/brdoc` v3 with a signature-parity guard.
- **Synthetic data:** `GeneratePerson` (UF-consistent identities).
- **Hardening (plans 001–006):** CI now gates `main` + GUI build deps dropped; CNPJ rejects
  all-equal inputs; MCP error prefix corrected; `brdoc.go` split into `cpf.go`/`cnpj.go`; RG SP/RJ
  check-digit convention verified against authoritative sources; Inscrição Estadual SP shipped.
- **Test Coverage:** 92.2% total (core 94.2%, compat 95.3%, cmd/selo 87.0%, mcp 84.4%).

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
- **Note:** a v2 release requires the `/v2` module-path suffix per Go module rules; removing an
  exported symbol is a breaking change, so it is grouped here rather than in a v1.x minor.
- **Coverage target:** ≥85%.
