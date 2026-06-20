# Features

What `github.com/inovacc/selo` does today, and proposed extensions. Status: **Completed** (shipped),
**In progress**, or **Proposed**. See [ROADMAP.md](ROADMAP.md) for phasing and
[BACKLOG.md](BACKLOG.md) for prioritized work.

## Completed

### Document operations (all kinds)
- **Validate** — check-digit / format validation, accepting formatted or raw input.
- **Generate** — produce a fresh valid document (masked form).
- **Format** — render the canonical Brazilian mask, or a sentinel error.
- **Origin** — resolve issuing UF/region for geolocatable kinds (CPF region, CEP range, phone DDD,
  voter-ID UF code).

### Supported kinds (13)
CPF · CNPJ (alphanumeric + legacy numeric) · CNH · PIS/PASEP/NIS · RENAVAM · Título Eleitoral ·
CEP · phone · license plate (national + Mercosul) · CNS · RG (SP) · Inscrição Estadual (SP/MG/RS/PR) ·
PIX keys (CPF/CNPJ/email/phone/EVP).

### Architecture
- **`Document` interface** + optional `OriginResolver` and `UFScoped` capabilities (type assertion).
- **Self-registering registry** — types register in `init()`; `Validate`/`Generate`/`Format` dispatch
  by `Kind`; `Kinds()` lists all; `Detect` auto-detects.
- **Comparable error sentinels** (`errors.Is`).

### Surfaces
- **Library** — ergonomic per-type API (`NewCPF()…`) and a generic registry-driven API.
- **CLI** (`cmd/selo`) — one subcommand per kind, derived from the registry; bulk `--from FILE|-`,
  `--count`, `--bulk`, `--uf`, scriptable exit codes; plus `detect`, `person` (`--seed`), `gen`,
  `mcp`, `completion`, `version`.
- **MCP server** (`selo mcp`) — 7 stdio tools with registry-sourced kind enums.
- **Shell completions** — `selo completion [bash|zsh|fish|powershell]` emits a completion script with
  per-shell install instructions.
- **Prebuilt binaries** — a `v*` tag auto-publishes linux/darwin/windows × amd64/arm64 archives +
  checksums + source via GoReleaser and the `inovacc/workflows` reusable release; `selo version`
  reports ldflags-injected version/commit/date in release builds.

### Compatibility
- **`compat` subpackage** — drop-in replacement for `paemuri/brdoc` v3's `Is*` API, with a
  compile-time signature-parity guard.

### Synthetic data
- **`GeneratePerson`** — one coherent fake identity carrying every document type (RG for SP; IE for
  SP/MG/RS/PR), all valid and UF-consistent; options `WithUF`, `WithSeed`/`WithRand`, `WithVehicle`,
  `WithCompany`, `Formatted`. Carries an optional UF-consistent `Address` (Street, Number,
  Neighborhood, City — a real municipality in the person's UF — UF, CEP); seeded output stays
  byte-identical (Address is the last rand draw).

### Code generation
- **`selo gen`** (and the MCP `generate_code` tool) — emit standalone validate/format/origin/generate
  code for all 13 kinds in **TypeScript, JavaScript, Ruby, Java, C#, Python, PHP, and Rust** (Rust is
  emitted as a Cargo library crate), each with Go-produced golden vectors and a runnable test suite; a
  CI matrix verifies every target on its real toolchain ([CODEGEN.md](CODEGEN.md)). *Parity gap:* the
  IE emitter is still SP-only (emitting MG/RS/PR needs a digit-sum DV rule — see BACKLOG).

### Quality
- **Benchmark suite** — Go 1.25 `b.Loop` benchmarks with alloc reporting for hot paths.
- **Fuzz coverage** — native fuzz targets across all 13 kinds and the `Detect` entry points.
- **Runnable examples** — godoc `Example` functions (`Detect`, registry `Validate`, `GeneratePerson`).

### Deterministic generation
- **Seedable generation** — `WithSeed(int64)` / `WithRand(*rand.Rand)` on `GeneratePerson`, the
  `RandGenerator` interface (`GenerateRand(*rand.Rand)`) on every document type, and the registry
  `GenerateRand(kind, r)` helper; exposed across the CLI (`selo person --seed`, `selo <kind>
  --generate/--bulk --seed`) and MCP (`generate_person` and `generate_document` `seed`). Same seed →
  identical output.

## In progress
- **Inscrição Estadual breadth** — SP/MG/RS/PR shipped; RJ blocked (its official page omits the
  weight vector); the remaining UFs pending verified algorithms/samples ([IE-NOTES.md](IE-NOTES.md)).

## Proposed
- **Codegen IE parity** — emit MG/RS/PR (not just SP) in the generated targets; needs a digit-sum DV
  rule in the codegen spec.
- **Multi-state RG** — extend RG beyond SP where per-UF check-digit rules are documented. RJ was
  removed in v1.3.0 (its algorithm differs from SP); re-adding it (or any UF) is blocked on an
  authoritative algorithm + ≥2 verifiable samples.
- **More codegen targets** — additional languages beyond the current eight (e.g. Go, Kotlin, Swift).
