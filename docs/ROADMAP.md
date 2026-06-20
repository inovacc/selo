# Roadmap

Status of `github.com/inovacc/selo` — a Go toolkit to validate, generate, format, and
geolocate Brazilian documents through a library, a CLI, and an MCP server.

**Overall: ~85% of the planned v1 surface is complete.** The core library, CLI, MCP server,
paemuri compat layer, and synthetic-person generator are all shipped. The main remaining work
is breadth (more Inscrição Estadual UFs, multi-state RG) and reproducible generation.

## Phase 1 — Core library & document types — ✅ COMPLETE
- [COMPLETE] `Document` interface (`Kind`/`Validate`/`Generate`/`Format`) + optional
  `OriginResolver` and `UFScoped` capabilities, discovered by type assertion.
- [COMPLETE] Self-registering type registry (`Register`/`Get`/`Kinds`/`Validate`/`Generate`/`Format`).
- [COMPLETE] 13 document kinds: CPF, CNPJ (alphanumeric + legacy), CNH, PIS/PASEP/NIS,
  RENAVAM, Título Eleitoral, CEP, phone, license plate, CNS, RG (SP/RJ), Inscrição Estadual
  (SP), PIX keys.
- [COMPLETE] `Detect` (auto-detect kind) and `DetectPIXKind`.
- [COMPLETE] Comparable error sentinels (`ErrInvalidLength`, `ErrInvalidFormat`, `ErrUnknownKind`,
  `ErrUnsupported`, `ErrUFNotImplemented`).

## Phase 2 — CLI (`cmd/selo`) — ✅ COMPLETE
- [COMPLETE] Cobra CLI deriving one subcommand per registered kind from the registry.
- [COMPLETE] Per-kind flags: `--validate`, `--generate`, `--format`, `--origin`, `--from FILE|-`
  (bulk + stdin), `--count`, `--uf` (UF-scoped kinds).
- [COMPLETE] `detect`, `person` (synthetic identities), `version` subcommands.
- [COMPLETE] Exit code 1 on invalid input (scriptable).

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
  valid and sharing a UF (CPF region, voter-ID code, phone DDD, CEP agree). Options:
  `WithUF`, `WithVehicle`, `WithCompany`.

## Phase 6 — Breadth & polish — 🚧 IN PROGRESS
- [COMPLETE] Repo hygiene: `.gitattributes` enforces LF repo-wide (settled the CRLF/CI noise).
- [COMPLETE] CLI: `--bulk N` bulk document generation; `--uf` now surfaces "UF not implemented"
  instead of a bare "invalid"; `scanBuf` shared buffer made call-local (latent data race removed).
- [IN PROGRESS] **Inscrição Estadual** beyond SP — MG/RJ/RS/PR researched but deferred for lack
  of ≥2 verifiable samples; 26 UFs remain (see [IE-NOTES.md](IE-NOTES.md)).
- [PLANNED] **Multi-state RG** — extend beyond SP/RJ where per-UF rules are documented; verify
  the RJ algorithm independently first.
- [PLANNED] **Reproducible `GenPerson`** — accept a `*rand.Rand` seed for deterministic fixtures.
- [PLANNED] Remove the deprecated `ValidateDocument` after 2026-07-18 (see BACKLOG).

## Phase 7 — Multi-language code generation — ✅ COMPLETE
- [COMPLETE] `selo gen` emits validate/format/origin code for all 13 kinds in **TypeScript,
  JavaScript, Ruby, Java, and C#**, each with Go-produced golden vectors and a runnable test suite.
- [COMPLETE] `internal/codegen` framework (spec + golden vectors + data tables + per-language
  emitters), the `selo gen` CLI, and the MCP `generate_code` tool.
- [COMPLETE] CI matrix (`codegen.yml`) verifies every target on real toolchains (all 5 green).
- [PLANNED] Cross-language `generate()` parity (targets currently validate/format/origin only).
- See [CODEGEN.md](CODEGEN.md).

## Test Coverage
**Current:** 92.2%  |  **Target:** 80%+ (met)

| Package | Coverage | Status |
|---------|----------|--------|
| `github.com/inovacc/selo` (core) | 94.2% | Healthy |
| `github.com/inovacc/selo/compat` | 95.3% | Healthy |
| `github.com/inovacc/selo/cmd/selo` (CLI) | 87.0% | Healthy |
| `github.com/inovacc/selo/mcp` | 84.4% | Healthy (error-path branches lightest) |

Measured with `task cover` (`go test -covermode=atomic -coverprofile=… ./...`) on 2026-06-19.
