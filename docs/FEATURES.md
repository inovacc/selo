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
CEP · phone · license plate (national + Mercosul) · CNS · RG (SP) · Inscrição Estadual (SP) ·
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
  `mcp`, `version`.
- **MCP server** (`selo mcp`) — 7 stdio tools with registry-sourced kind enums.

### Compatibility
- **`compat` subpackage** — drop-in replacement for `paemuri/brdoc` v3's `Is*` API, with a
  compile-time signature-parity guard.

### Synthetic data
- **`GeneratePerson`** — one coherent fake identity carrying every document type (RG and IE for SP),
  all valid and UF-consistent; options `WithUF`, `WithSeed`/`WithRand`, `WithVehicle`, `WithCompany`,
  `Formatted`.

### Code generation
- **`selo gen`** (and the MCP `generate_code` tool) — emit standalone validate/format/origin/generate
  code for all 13 kinds in **TypeScript, JavaScript, Ruby, Java, C#, Python, and PHP**, each with
  Go-produced golden vectors and a runnable test suite; a CI matrix verifies every target on its real
  toolchain ([CODEGEN.md](CODEGEN.md)).

### Deterministic generation
- **Seedable generation** — `WithSeed(int64)` / `WithRand(*rand.Rand)` on `GeneratePerson`, the
  `RandGenerator` interface (`GenerateRand(*rand.Rand)`) on every document type, and the registry
  `GenerateRand(kind, r)` helper; exposed across the CLI (`selo person --seed`, `selo <kind>
  --generate/--bulk --seed`) and MCP (`generate_person` and `generate_document` `seed`). Same seed →
  identical output.

## In progress
- **Inscrição Estadual breadth** — SP shipped; MG/RJ/RS/PR + remaining UFs pending verified
  algorithms/samples ([IE-NOTES.md](IE-NOTES.md)).

## Proposed
- **Multi-state RG** — extend RG beyond SP where per-UF check-digit rules are documented. RJ was
  removed in v1.3.0 (its algorithm differs from SP); re-adding it (or any UF) is blocked on an
  authoritative algorithm + ≥2 verifiable samples.
- **More codegen targets** — additional languages beyond the current seven (e.g. Go, Kotlin, Rust).
