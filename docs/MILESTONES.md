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
  Título Eleitoral, CEP, phone, license plate, CNS, RG (SP), Inscrição Estadual (SP), PIX keys.
- **Surfaces:** Cobra CLI (`cmd/selo`, registry-derived; scriptable exit codes), stdio MCP server
  (6 tools), `compat` subpackage mirroring `paemuri/brdoc` v3 with a signature-parity guard.
- **Synthetic data:** `GeneratePerson` (UF-consistent identities).
- **Hardening (plans 001–006):** CI now gates `main` + GUI build deps dropped; CNPJ rejects
  all-equal inputs; MCP error prefix corrected; `brdoc.go` split into `cpf.go`/`cnpj.go`; RG SP/RJ
  check-digit convention verified against authoritative sources; Inscrição Estadual SP shipped.
- **Test Coverage:** 92.2% total (core 94.2%, compat 95.3%, cmd/selo 87.0%, mcp 84.4%).

## v1.2.0 — Multi-language code generation + CLI polish — ✅ RELEASED (2026-06-19, tag `v1.2.0`)
- **Goal:** generate validators in other languages from the verified Go algorithms, plus CLI and
  quality polish.
- **Code generation (`selo gen`):** validate/format/origin for all 13 kinds in TypeScript,
  JavaScript, Ruby, Java, and C# — Go-produced golden vectors + per-language test suites; a CI
  matrix verifies all five on real toolchains. `internal/codegen` framework + `selo gen` CLI + MCP
  `generate_code` tool. See [CODEGEN.md](CODEGEN.md).
- **CLI:** `--bulk N` bulk document generation (implies `--generate`); `--uf` now surfaces the real
  reason ("UF not implemented") instead of a bare "invalid".
- **Hardening:** `.gitattributes` enforces LF repo-wide (fixed the CRLF/CI divergence + the
  `default: all` golangci-lint pass); `scanBuf` made call-local (latent data race removed); MCP
  error-path coverage 85% → 93%.
- **Coverage target:** ≥85% per package (met).

## v1.3.0 — Generate parity + seedable generation + RG correctness — ✅ RELEASED (2026-06-19, tag `v1.3.0`)
- **Goal:** cross-language generation parity, reproducible generation, and an RG correctness fix.
- **Cross-language `generate()` parity:** every codegen target (TypeScript, JavaScript, Ruby,
  Java, C#) now emits `generate<Kind>()` for all 13 kinds alongside validate/format/origin, each
  with a generate→validate round-trip test verified by the CI matrix on real toolchains.
- **Seedable / deterministic generation:** a `RandGenerator` capability interface
  (`GenerateRand(*math/rand/v2.Rand)`) on every document type, a registry `GenerateRand(kind, r)`
  helper, and `GeneratePerson` `WithSeed(int64)` / `WithRand(*rand.Rand)` options — the same seed
  produces identical output. Default `Generate()` stays random (PIX still uses `crypto/rand`).
- **RG correctness:** `UFRJ` demoted to `ErrUFNotImplemented` (research found RJ's algorithm
  differs from SP; validating RJ with SP's algorithm was likely wrong). `ImplementedUFs()` lists
  SP only. Behavior change for callers relying on `--uf RJ`.
- **Test Coverage:** ≥85% per package (maintained).

## v1.4.0 — Surface the seed + a 6th codegen language — ✅ RELEASED (2026-06-19, tag `v1.4.0`)
- **Goal:** expose seedable generation at the CLI/MCP surfaces and widen codegen reach.
- Shipped:
  - ✅ CLI `selo person --seed N` and MCP `generate_person` `seed` param (library already supported
    it via `WithSeed`); a `--count` batch shares one seeded source — reproducible yet distinct.
  - ✅ **Python** codegen target (6th language) — validate/format/origin/generate for all 13 kinds,
    Go-produced golden vectors + a pytest suite (686 tests) + a CI-matrix lane.
  - ✅ CLI consistency: unified per-command `SilenceUsage` on the root command.
- **Blocked (need verifiable sources):** Inscrição Estadual beyond SP (26 UFs), multi-state RG /
  re-adding RJ, and RNM — each gated on an authoritative algorithm plus ≥2 verifiable samples.
- **Test Coverage:** 94.2% total (core 92.5%, compat 94.6%, cmd/selo 89.3%, mcp 93.3%,
  internal/codegen 95.6%).

## v1.5.0 — Breadth — 🔜 PLANNED
- **Goal:** broaden UF coverage once authoritative sources are obtained.
- Candidate scope (each gated by an authoritative algorithm + ≥2 verifiable samples):
  - Inscrição Estadual next batch (MG, RJ, RS, PR), then the remaining UFs.
  - Multi-state RG (re-verify RJ independently, then add documented UFs).
  - RNM (Registro Nacional Migratório) — blocked on a public format + check-digit spec.
- **Coverage target:** maintain ≥85% per package.

## v2.0.0 — Cleanup (breaking) — 🔮 FUTURE
- **Goal:** remove deprecations once their windows pass.
- Remove `ValidateDocument` (deprecated; removal after 2026-07-18) in a dedicated commit.
- **Note:** a v2 release requires the `/v2` module-path suffix per Go module rules; removing an
  exported symbol is a breaking change, so it is grouped here rather than in a v1.x minor.
- **Coverage target:** ≥85%.
