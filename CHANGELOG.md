# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project follows
[Semantic Versioning](https://semver.org/).

> **Project rename:** this project was renamed from `brdoc` to `selo`
> (module `github.com/inovacc/selo`) in the 1.1.0 line. The `v1.0.0` tag predates the rename and
> points at the original `brdoc` code; `github.com/inovacc/selo` is first installable at **v1.1.0**.

## [1.6.0] - 2026-06-20

Shell completions, prebuilt cross-platform binaries, expanded Inscrição Estadual coverage, richer
synthetic people, an eighth code-generation target (Rust), and a benchmark/fuzz/example quality pass.

### Added
- **Shell completions.** A documented `selo completion [bash|zsh|fish|powershell]` command (Cobra's
  default completion command was previously disabled) emits a completion script with per-shell
  install instructions.
- **Inscrição Estadual — MG, RS, PR (now SP, MG, RS, PR).** Three new UFs with authoritative,
  adversarially-verified algorithms (official SINTEGRA-MG / SINTEGRA-RS and SEFA-PR worked examples,
  corroborated by independent reference implementations). `GeneratePerson` now carries a
  UF-consistent `IE` for MG/RS/PR people too. **RJ was re-researched and remains blocked** — its
  official page omits the weight vector. See [IE-NOTES.md](docs/IE-NOTES.md).
- **Prebuilt cross-platform binaries (GoReleaser).** Pushing a `v*` tag now auto-publishes
  linux/darwin/windows × amd64/arm64 archives plus checksums and source via the existing
  `inovacc/workflows` reusable release (a `.goreleaser.yaml` config was added). `selo version` now
  reports the ldflags-injected version/commit/date in release builds. New `Taskfile`
  `release:check` / `release:snapshot` / `release` targets.
- **`Person.Address`.** `GeneratePerson` gains an optional UF-consistent `Address` (Street, Number,
  Neighborhood, City, UF, CEP); `City` is a real municipality in the person's UF, backed by expanded
  pt-BR name lists. Seeded determinism is preserved — `Address` is the last rand draw, so
  pre-existing seeded output is byte-identical.
- **Rust code-generation target (8th language).** `selo gen --lang rust` emits
  validate / format / origin / generate for all 13 kinds as a Cargo library crate with Go-produced
  golden-vector tests and a CI cargo lane. The MCP `generate_code` tool now offers `rust`. Languages
  now: TypeScript, JavaScript, Ruby, Java, C#, Python, PHP, Rust. See [CODEGEN.md](docs/CODEGEN.md).
- **Quality:** a Go 1.25 `b.Loop` benchmark suite (with alloc reporting) for hot paths; fuzz
  coverage extended to all 13 kinds and the `Detect` entry points; runnable godoc `Example`
  functions (`Detect`, registry `Validate`, `GeneratePerson`).
- **ADR-0004** documenting binary distribution via GoReleaser.

### Changed
- ADR-0003 updated to list Rust (8 code-generation targets).
- Docs reconciled with the above (README / ROADMAP / MILESTONES / FEATURES / BACKLOG / ISSUES).

### Known issues
- The multi-language codegen IE emitter is still **SP-only** — a documented parity gap; emitting
  MG/RS/PR in generated targets needs a digit-sum DV rule (tracked in BACKLOG).

## [1.5.0] - 2026-06-20

Seedable generation across every generate surface, Inscrição Estadual in synthetic people, and a
seventh code-generation target (PHP).

### Added
- **PHP code-generation target (7th language).** `selo gen --lang php` emits
  validate / format / origin / generate for all 13 kinds as a PSR-4 Composer package with a
  Go-produced golden-vector PHPUnit suite (678 tests), verified by a CI-matrix lane on a real PHP
  toolchain. The MCP `generate_code` tool now offers `php`. See [CODEGEN.md](docs/CODEGEN.md).
- **`--seed` on per-kind generation.** `selo <kind> --generate` / `--bulk --seed N` and the MCP
  `generate_document` `seed` field now produce deterministic, reproducible output (a batch shares one
  seeded source — reproducible yet still distinct), completing the seedable surface begun in v1.4.0.
- **Inscrição Estadual in `GeneratePerson`.** A person whose UF has a verified IE algorithm
  (currently SP) now carries a UF-consistent `IE` field (seedable like the rest of the identity).
- **ADR-0003** documenting the multi-language code-generation architecture.

### Changed
- Docs reconciled with the above (ROADMAP / MILESTONES / FEATURES / BACKLOG / ISSUES); the resolved
  `SilenceUsage` item moved to BACKLOG "Resolved".

## [1.4.0] - 2026-06-19

Seed exposed at the CLI/MCP surfaces, a sixth code-generation target (Python), and CLI/doc polish.

### Added
- **`selo person --seed N` and MCP `generate_person` `seed`.** The v1.3.0 seedable generation is now
  reachable from both surfaces: a `--seed` flag (CLI) and a `seed` field (MCP) produce deterministic,
  reproducible output. A batch (`--count` / `count`) shares one seeded source, so it stays
  reproducible while still yielding distinct people. New exported helper
  `NewSeededRand(seed int64) *rand.Rand`.
- **Python code-generation target (6th language).** `selo gen --lang python` emits
  validate / format / origin / generate for all 13 kinds as an installable Python package
  (`pip install -e .`) with a Go-produced golden-vector pytest suite (686 tests). Includes
  `internal/codegen/emit_python*.go` + `templates/python/`, `golden_python_test.go`, the committed
  `generated/python/` reference, and a Python CI-matrix lane (`codegen.yml`) verifying it on a real
  toolchain. The MCP `generate_code` tool now offers `python`. See [CODEGEN.md](docs/CODEGEN.md).

### Changed
- CLI: unified `SilenceUsage` on the root command (removed redundant per-call assignments in
  `detect` / `format` / `origin`).
- Documentation reconciled with the shipped v1.1.0–v1.3.0 state (ROADMAP / MILESTONES / ISSUES /
  BACKLOG): resolved tech-debt moved to a "Resolved" section; the inaccurate "generation is not
  reproducible" note corrected; RG documented as SP-only.

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

[1.6.0]: https://github.com/inovacc/selo/compare/v1.5.0...v1.6.0
[1.5.0]: https://github.com/inovacc/selo/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/inovacc/selo/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/inovacc/selo/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/inovacc/selo/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/inovacc/selo/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/inovacc/selo/releases/tag/v1.0.0
[0.1.0]: https://github.com/inovacc/selo/releases/tag/v0.1.0
[0.0.1]: https://github.com/inovacc/selo/releases/tag/v0.0.1
