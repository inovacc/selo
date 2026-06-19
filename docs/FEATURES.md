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
CEP · phone · license plate (national + Mercosul) · CNS · RG (SP/RJ) · Inscrição Estadual (SP) ·
PIX keys (CPF/CNPJ/email/phone/EVP).

### Architecture
- **`Document` interface** + optional `OriginResolver` and `UFScoped` capabilities (type assertion).
- **Self-registering registry** — types register in `init()`; `Validate`/`Generate`/`Format` dispatch
  by `Kind`; `Kinds()` lists all; `Detect` auto-detects.
- **Comparable error sentinels** (`errors.Is`).

### Surfaces
- **Library** — ergonomic per-type API (`NewCPF()…`) and a generic registry-driven API.
- **CLI** (`cmd/selo`) — one subcommand per kind, derived from the registry; bulk `--from FILE|-`,
  `--count`, `--uf`, scriptable exit codes; plus `detect`, `person`, `version`.
- **MCP server** (`selo mcp`) — 6 stdio tools with registry-sourced kind enums.

### Compatibility
- **`compat` subpackage** — drop-in replacement for `paemuri/brdoc` v3's `Is*` API, with a
  compile-time signature-parity guard.

### Synthetic data
- **`GeneratePerson`** — one coherent fake identity carrying every document type, all valid and
  UF-consistent; options `WithUF`, `WithVehicle`, `WithCompany`.

## In progress
- **Inscrição Estadual breadth** — SP shipped; MG/RJ/RS/PR + remaining UFs pending verified
  algorithms/samples ([IE-NOTES.md](IE-NOTES.md)).

## Proposed
- **Multi-state RG** — extend RG beyond SP/RJ where per-UF check-digit rules are documented
  (verify RJ independently first).
- **Seedable generation** — `WithSeed` / `*rand.Rand` so `GeneratePerson` and per-type `Generate`
  produce deterministic fixtures.
- **IE field in `GeneratePerson`** — include the person's UF Inscrição Estadual once IE coverage
  is broad enough.
- **Richer CLI UX** — distinct "UF not implemented" message for UF-scoped kinds (vs. plain `invalid`).
