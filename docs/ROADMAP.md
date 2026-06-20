# Roadmap

Status of `github.com/inovacc/selo` — a Go toolkit to validate, generate, format, and
geolocate Brazilian documents through a library, a CLI, and an MCP server.

**Overall: ~92% of the planned v1 surface is complete.** The core library, CLI (with shell
completions and prebuilt cross-platform binaries), MCP server, paemuri compat layer,
synthetic-person generator (now with a UF-consistent address), multi-language code generation
(8 targets with `generate()` parity), and seedable/deterministic generation are all shipped.
Inscrição Estadual now covers SP/MG/RS/PR. The main remaining work is breadth: the remaining
Inscrição Estadual UFs (RJ blocked — its official page omits the weight vector) and multi-state
RG — both blocked on authoritative per-UF algorithms plus ≥2 verifiable samples.

## Phase 1 — Core library & document types — ✅ COMPLETE
- [COMPLETE] `Document` interface (`Kind`/`Validate`/`Generate`/`Format`) + optional
  `OriginResolver` and `UFScoped` capabilities, discovered by type assertion.
- [COMPLETE] Self-registering type registry (`Register`/`Get`/`Kinds`/`Validate`/`Generate`/`Format`).
- [COMPLETE] 13 document kinds: CPF, CNPJ (alphanumeric + legacy), CNH, PIS/PASEP/NIS,
  RENAVAM, Título Eleitoral, CEP, phone, license plate, CNS, RG (SP), Inscrição Estadual
  (SP), PIX keys.
- [COMPLETE] `Detect` (auto-detect kind) and `DetectPIXKind`.
- [COMPLETE] Comparable error sentinels (`ErrInvalidLength`, `ErrInvalidFormat`, `ErrUnknownKind`,
  `ErrUnsupported`, `ErrUFNotImplemented`).

## Phase 2 — CLI (`cmd/selo`) — ✅ COMPLETE
- [COMPLETE] Cobra CLI deriving one subcommand per registered kind from the registry.
- [COMPLETE] Per-kind flags: `--validate`, `--generate`, `--format`, `--origin`, `--from FILE|-`
  (bulk + stdin), `--count`, `--uf` (UF-scoped kinds).
- [COMPLETE] `detect`, `person` (synthetic identities), `version`, `gen`, `mcp`, `completion`
  subcommands. `selo completion [bash|zsh|fish|powershell]` ships an explicit completion command
  with per-shell install instructions (v1.6.0).
- [COMPLETE] Exit code 1 on invalid input (scriptable).
- [COMPLETE] Prebuilt cross-platform binaries — a `v*` tag auto-publishes linux/darwin/windows ×
  amd64/arm64 archives + checksums + source via GoReleaser and the `inovacc/workflows` reusable
  release; `selo version` reports ldflags-injected version/commit/date in release builds (v1.6.0).

## Phase 3 — MCP server (`mcp`) — ✅ COMPLETE
- [COMPLETE] stdio MCP server (via `modelcontextprotocol/go-sdk`) launched by `selo mcp`.
- [COMPLETE] 6 tools, kind enums sourced from the registry: `validate_document`,
  `generate_document`, `format_document`, `detect_document`, `list_document_types`,
  `generate_person`.

## Phase 4 — Compatibility & migration — ✅ COMPLETE
- [COMPLETE] `compat` subpackage mirroring `paemuri/brdoc` v3's flat `Is*` API for one-line
  import-swap migration, with a compile-time signature-parity guard.

## Phase 5 — Synthetic data (GenPerson) — ✅ COMPLETE
- [COMPLETE] `GeneratePerson` — one coherent fake identity carrying every document type, all
  valid and sharing a UF (CPF region, voter-ID code, phone DDD, CEP agree; RG for SP; IE for
  SP/MG/RS/PR).
  Options: `WithUF`, `WithSeed`/`WithRand`, `WithVehicle`, `WithCompany`, `Formatted`.
- [COMPLETE] Optional UF-consistent `Address` (Street, Number, Neighborhood, City, UF, CEP) — City
  is a real municipality in the person's UF, backed by expanded pt-BR name lists. Seeded
  determinism preserved (Address is the last rand draw, so prior seeded output is byte-identical)
  (v1.6.0).

## Phase 6 — Breadth & polish — 🚧 IN PROGRESS
- [COMPLETE] Repo hygiene: `.gitattributes` enforces LF repo-wide (settled the CRLF/CI noise).
- [COMPLETE] CLI: `--bulk N` bulk document generation; `--uf` now surfaces "UF not implemented"
  instead of a bare "invalid"; `scanBuf` shared buffer made call-local (latent data race removed).
- [COMPLETE] Shell completions — explicit `selo completion [bash|zsh|fish|powershell]` command with
  per-shell install instructions (v1.6.0).
- [COMPLETE] Prebuilt cross-platform binaries via GoReleaser + the `inovacc/workflows` reusable
  release; ldflags-injected `selo version` in release builds (v1.6.0).
- [COMPLETE] **Inscrição Estadual** beyond SP — **MG, RS, PR shipped** (v1.6.0) with authoritative,
  adversarially-verified algorithms (SINTEGRA-MG/RS, SEFA-PR worked examples + independent
  reference-impl corroboration); `GeneratePerson` carries a UF-consistent IE for them too.
- [BLOCKED] **Inscrição Estadual RJ** — re-researched and still blocked: RJ's official page omits
  the weight vector, and no ≥2 verifiable samples were obtainable (see [IE-NOTES.md](IE-NOTES.md)).
  The remaining UFs are unstarted.
- [COMPLETE] Quality pass: a Go 1.25 `b.Loop` benchmark suite (alloc reporting) for hot paths; fuzz
  coverage extended to all 13 kinds + the `Detect` entry points; runnable godoc `Example` functions
  (`Detect`, registry `Validate`, `GeneratePerson`) (v1.6.0).
- [BLOCKED] **Multi-state RG** — RJ was demoted to `ErrUFNotImplemented` in v1.3.0 (its
  check-digit algorithm differs from SP); re-adding RJ or any other UF is blocked on an
  authoritative per-UF algorithm plus ≥2 verifiable samples.
- [COMPLETE] **Reproducible generation** — `WithSeed(int64)` / `WithRand(*rand.Rand)` and the
  `RandGenerator` interface shipped in v1.3.0; `--seed` reaches `selo person` + `generate_person`
  (v1.4.0) and the per-kind `selo <kind> --generate/--bulk` + the `generate_document` MCP tool
  (v1.5.0).
- [PLANNED] Remove the deprecated `ValidateDocument` after 2026-07-18 (see BACKLOG).

## Phase 7 — Multi-language code generation — ✅ COMPLETE
- [COMPLETE] `selo gen` emits validate/format/origin code for all 13 kinds in **TypeScript,
  JavaScript, Ruby, Java, and C#**, each with Go-produced golden vectors and a runnable test suite.
- [COMPLETE] `internal/codegen` framework (spec + golden vectors + data tables + per-language
  emitters), the `selo gen` CLI, and the MCP `generate_code` tool.
- [COMPLETE] CI matrix (`codegen.yml`) verifies every target on real toolchains (all 8 green).
- [COMPLETE] Cross-language `generate()` parity — all targets emit `generate<Kind>()` for every
  one of the 13 kinds with generate→validate round-trip tests, CI-matrix verified (v1.3.0).
- [COMPLETE] **Python** target (6th language) — validate/format/origin/generate for all 13 kinds,
  Go-produced golden vectors + a pytest suite (686 tests) + a CI-matrix lane (v1.4.0).
- [COMPLETE] **PHP** target (7th language) — full parity for all 13 kinds, golden vectors + a
  PHPUnit suite (678 tests) + a CI-matrix lane (v1.5.0).
- [COMPLETE] **Rust** target (8th language) — full parity for all 13 kinds as a Cargo library crate,
  Go-produced golden-vector tests + a CI cargo lane (v1.6.0). **Languages now (8): TypeScript,
  JavaScript, Ruby, Java, C#, Python, PHP, Rust.**
- [NOTE] Parity gap: the codegen IE emitter is still **SP-only** — emitting MG/RS/PR in generated
  targets needs a digit-sum DV rule (tracked in BACKLOG).
- See [CODEGEN.md](CODEGEN.md).

## Test Coverage
**Current:** 94.6%  |  **Target:** 80%+ (met)

| Package | Coverage | Status |
|---------|----------|--------|
| `github.com/inovacc/selo` (core) | 93.1% | Healthy |
| `github.com/inovacc/selo/compat` | 94.6% | Healthy |
| `github.com/inovacc/selo/cmd/selo` (CLI) | 88.3% | Healthy |
| `github.com/inovacc/selo/mcp` | 93.7% | Healthy |
| `github.com/inovacc/selo/internal/codegen` | 96.0% | Healthy |

Measured with `task cover` (`go test -covermode=atomic -coverprofile=… ./...`) on 2026-06-20.
