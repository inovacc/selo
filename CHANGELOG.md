# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project follows
[Semantic Versioning](https://semver.org/).

> **Project rename:** this project was renamed from `brdoc` to `selo`
> (module `github.com/inovacc/selo`) in the 1.1.0 line. The `v1.0.0` tag predates the rename and
> points at the original `brdoc` code; `github.com/inovacc/selo` is first installable at **v1.1.0**.

## [1.3.0] - 2026-06-19

### Added
- **Cross-language `generate()` parity.** Every code-generation target (TypeScript, JavaScript,
  Ruby, Java, C#) now emits a `generate<Kind>()` for all 13 kinds alongside validate/format/origin,
  each with a generate→validate round-trip test verified by the CI matrix on real toolchains.
- **Seedable / deterministic generation.** A `RandGenerator` capability interface
  (`GenerateRand(r *math/rand/v2.Rand) string`) implemented by every document type, a registry
  `GenerateRand(kind, r)` helper, and `GeneratePerson` `WithSeed(int64)` / `WithRand(*rand.Rand)`
  options — same seed produces identical output for deterministic fixtures. The default `Generate()`
  remains random and unchanged. (PIX's default `Generate()` still uses `crypto/rand`.)

### Changed
- **RG now supports SP only.** `RG.ValidateUF(value, UFRJ)` now returns `ErrUFNotImplemented` and
  `ImplementedUFs()` no longer lists RJ. Research found RJ uses a *different* check-digit algorithm
  than SP, so validating RJ with SP's algorithm was likely incorrect; rather than return wrong
  answers, RJ is unimplemented until an authoritative RJ algorithm + verifiable samples are sourced
  (see `docs/ISSUES.md`). This is a behavior change for callers relying on `--uf RJ`.

## [1.2.0] - 2026-06-19

Multi-language code generation, a bulk-generate CLI flag, and CLI/quality polish.

### Added
- **`selo gen` — multi-language code generation.** Emits **validate / format / origin** code for
  all 13 document kinds in **TypeScript, JavaScript, Ruby, Java, and C#**, each with Go-produced
  golden test vectors and a runnable test suite. Includes the `internal/codegen` framework (spec +
  vectors + data tables + per-language emitters), the `selo gen` CLI, the MCP `generate_code` tool,
  `Taskfile` `gen:*` / `gen:verify:*` targets, and a CI matrix (`codegen.yml`) verifying every
  target on real toolchains (node / ruby / JDK+Maven / .NET). See `docs/CODEGEN.md`.
- **CLI `--bulk N` / `-b N`** — bulk-generate N documents (implies `--generate`).
- `.gitattributes` enforcing LF line endings repo-wide.

### Changed
- CLI `--uf` for a UF-scoped kind (RG/IE) now surfaces the real reason ("UF not implemented")
  instead of a bare "invalid".
- The MCP server now exposes **7 tools** (added `generate_code`).

### Fixed
- `scanBuf` shared package-level scanner buffer made call-local in `cmd/selo/iohelper.go`
  (removed a latent data race).
- Repaired CI (red since CI was first enabled): `setup-go go-version: latest` → `stable`;
  satisfied the strict `golangci-lint default: all` config (84 pre-existing issues); resolved the
  CRLF/LF divergence that broke local-vs-CI parity.
- Raised MCP error-path test coverage 85% → 93%.

## [1.1.0] - 2026-06-19

First release under the `github.com/inovacc/selo` module path: the rebrand, the complete
Brazilian-document toolkit, and post-build hardening.

### Added
- **Complete document toolkit** behind a `Document` interface + self-registering registry,
  exposed identically via library, CLI, and MCP server. Kinds: CPF, CNPJ (alphanumeric + legacy),
  CNH, PIS/PASEP/NIS, RENAVAM, Título Eleitoral, CEP, phone, license plate, CNS, RG (SP/RJ), and
  PIX keys.
- **Inscrição Estadual (IE)** — UF-scoped like RG; first state **SP** implemented and verified
  against the SEFAZ-SP / Sintegra algorithm, with externally-sourced regression samples. CLI
  (`selo ie`) and MCP auto-derive it. Remaining UFs tracked in `docs/IE-NOTES.md`.
- `OriginResolver` (geolocation: CPF region, CEP/phone/voter-ID UF) and `UFScoped` optional
  capabilities, discovered by type assertion.
- **Cobra CLI** (`cmd/selo`) deriving one subcommand per registered kind; `detect`, `person`
  (synthetic identities), `mcp`, and `version` subcommands; bulk `--from FILE|-`; scriptable
  exit codes.
- **stdio MCP server** (`selo mcp`) with 6 registry-backed tools.
- **`compat` subpackage** mirroring `paemuri/brdoc` v3's `Is*` API for one-line-import migration,
  with a compile-time signature-parity guard.
- **`GeneratePerson`** — one coherent synthetic identity carrying every document type, all valid
  and UF-consistent (`WithUF`, `WithVehicle`, `WithCompany`).
- Project documentation set under `docs/` (ROADMAP, MILESTONES, ARCHITECTURE with Mermaid
  diagrams, ISSUES, BUGS, FEATURES, CONTRIBUTORS, IMPLEMENTATION_TASKS, ADR-0001/0002, IE-NOTES).

### Changed
- **Renamed `brdoc` → `selo`** (module `github.com/inovacc/selo`).
- Minimum Go version raised to **1.25** (required by the MCP `go-sdk`).
- Split the `brdoc.go` god-file into `cpf.go` + `cnpj.go` (behavior-preserving), matching the
  one-type-per-file convention.
- CI now gates `main` — workflows run on every branch and pull request; removed unused GUI build
  dependencies from the Linux build job.

### Fixed
- CNPJ now rejects all-equal inputs (e.g. `00000000000000`), matching CPF.
- MCP transport error prefix corrected from `brdoc mcp:` to `selo mcp:`.
- Corrected remaining `brdoc` references in package and example doc comments after the rebrand.
- Verified the RG (SP/RJ) check-digit convention against four independent authoritative sources;
  pinned externally-sourced regression samples (no behavior change).

## [1.0.0] - 2025-11-19

Original `brdoc` release (module `github.com/inovacc/brdoc`); CPF and alphanumeric CNPJ only.
Superseded by the `selo` rebrand in 1.1.0. See the entries below for the early history.

- Migrated unit tests to `github.com/stretchr/testify` (`assert`/`require`).
- CLI: fixed help/usage text being printed twice on incorrect flag usage (removed redundant
  `cmd.Help()` calls; enabled `SilenceUsage`/`SilenceErrors` on the root command).

## [0.1.0] - 2024-11-19

### Added

- Initial release
- CPF validation with check digit verification
- CPF generation with valid check digits
- CPF formatting (XXX.XXX.XXX-XX)
- CPF state/region identification based on 9th digit
- Detection of invalid CPF patterns (all same digits)
- CNPJ alphanumeric validation per SERPRO specification
- CNPJ alphanumeric generation
- CNPJ formatting (XX.XXX.XXX/XXXX-XX)
- Modulo 11 check digit calculation for CNPJ
- Auto-detection of document type (CPF/CNPJ)
- Comprehensive test suite with 95%+ coverage
- Benchmark suite for performance testing
- Thread-safe random number generation
- Zero external dependencies
- Complete API documentation
- Usage examples
- CI/CD pipeline with GitHub Actions

### Technical Details

- Implements official SERPRO algorithm for alphanumeric CNPJ
- Character mapping: 0-9 → 0-9, A-Z → 17-42 (ASCII - 48)
- Weight distribution: 2-9, repeating from right to left
- Special modulo 11 rule: remainder 0 or 1 → check digit 0

## [0.0.1] - 2024-11-15

### Added

- Project initialization
- Basic project structure
- MIT License

---

## Types of Changes

- `Added` for new features
- `Changed` for changes in existing functionality
- `Deprecated` for soon-to-be removed features
- `Removed` for now removed features
- `Fixed` for any bug fixes
- `Security` in case of vulnerabilities

[1.3.0]: https://github.com/inovacc/selo/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/inovacc/selo/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/inovacc/selo/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/inovacc/selo/releases/tag/v1.0.0
[0.1.0]: https://github.com/inovacc/selo/releases/tag/v0.1.0
[0.0.1]: https://github.com/inovacc/selo/releases/tag/v0.0.1
