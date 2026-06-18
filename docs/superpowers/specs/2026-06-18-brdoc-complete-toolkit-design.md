# Design Spec — Complete Brazilian-Document Toolkit

> **Status:** Approved design (brainstorming output). Ready for implementation planning.
> **Date:** 2026-06-18
> **Project:** `github.com/inovacc/brdoc` (pre-rebrand — see §10)
> **Companion analysis:** [`docs/FEATURE-GAP-paemuri.md`](../../FEATURE-GAP-paemuri.md) (per-type algorithms, paemuri comparison, upstream-issue research)

---

## 1. Goal & Non-Goals

### Goal
Evolve the current CPF/CNPJ project into a **complete Brazilian-document toolkit**: one Go module with a single core **library** plus three thin adapters — a **Cobra CLI**, an **MCP server** (`brdoc mcp`, stdio), and a **paemuri-compatible subpackage**. The toolkit *validates, generates, formats, and resolves origin* for all standard Brazilian document types, becoming a strict superset of `paemuri/brdoc` (which is validation-only).

### v1 success criteria
1. All 11 standard document types fully supported (validate + generate + format + origin where applicable).
2. PIX key validation included.
3. CLI exposes every type with no per-type boilerplate drift.
4. MCP server exposes the toolkit as agent tools over stdio.
5. `compat/` lets a paemuri user migrate with a one-line import change.
6. ≥80% coverage (currently ~95%), fuzz tests, godoc examples, lint-clean.
7. Rebrand (`/branding:names`) is a near-mechanical rename afterward.

### Non-Goals (v1)
- HTTP/REST or gRPC service tier (deferred; MCP is the only network-ish surface).
- **Inscrição Estadual** (27 per-UF algorithms) — v2.
- **RG beyond SP/RJ** — v2.

---

## 2. Core Architecture — Interface + Registry (Hybrid)

The library keeps **ergonomic concrete types** *and* gains a **registry + interface** so the CLI and MCP adapters derive everything from one source of truth.

### 2.1 The `Document` interface

```go
// Kind is the stable identifier for a document type, e.g. "cpf".
type Kind string

const (
    KindCPF      Kind = "cpf"
    KindCNPJ     Kind = "cnpj"
    KindCNH      Kind = "cnh"
    KindPIS      Kind = "pis"
    KindRenavam  Kind = "renavam"
    KindVoterID  Kind = "voter_id"   // Título Eleitoral
    KindCEP      Kind = "cep"
    KindPhone    Kind = "phone"
    KindPlate    Kind = "plate"
    KindCNS      Kind = "cns"
    KindRG       Kind = "rg"
    KindPIX      Kind = "pix"
)

// Document is implemented by every document type.
type Document interface {
    Kind() Kind
    Validate(value string) bool
    Generate() string
    Format(value string) (string, error)
}
```

### 2.2 Optional capability interfaces (discovered via type assertion)

```go
// OriginResolver returns geolocation/origin info (CPF region, CEP/phone/voter UF).
type OriginResolver interface {
    Origin(value string) (string, error)
}

// UFScoped is implemented by types whose validation depends on a federative unit (RG).
type UFScoped interface {
    ValidateUF(value string, uf UF) (bool, error)
    ImplementedUFs() []UF
}
```

Adapters check `if r, ok := doc.(OriginResolver); ok { ... }` so capabilities stay opt-in without bloating the base interface.

### 2.3 Registry + generic dispatch

```go
// registry.go
func Register(d Document)                                  // called from each type's init()
func Get(kind Kind) (Document, bool)
func Kinds() []Kind                                        // sorted, stable
func Validate(kind Kind, value string) (bool, error)       // ErrUnknownKind if absent
func Generate(kind Kind) (string, error)
func Format(kind Kind, value string) (string, error)
func Detect(value string) (Kind, bool)                     // generalizes today's ValidateDocument auto-detect
```

- Concrete types remain public & ergonomic: `brdoc.NewCPF().Validate(s)`, `brdoc.NewCNPJ().Generate()`.
- Each type registers a singleton in `init()`: `func init() { Register(&CPF{}) }`.
- **`Detect`** replaces/extends the existing `ValidateDocument(doc) (docType, isValid)`; the old function is kept as a thin, **deprecated** wrapper (per the user's deprecation policy) pointing at `Detect`/`Validate`.

### 2.4 Concurrency-safe generation
Replace the `init()`-seeded module-level `*rand.Rand` with **`math/rand/v2`** top-level functions (goroutine-safe in Go 1.22+; project is on Go 1.24). Registered singletons can therefore serve concurrent `Generate()` calls without per-instance state or mutexes. `NewCPF()`/`NewCNPJ()` constructors are retained (now effectively stateless) for backward compatibility.

### 2.5 The RG exception
RG validation needs a UF and returns an error. It implements the base `Document` interface (where `Validate(value)` tries all *implemented* UFs and `Format` masks) **and** `UFScoped`. The CLI exposes `--uf`, and the MCP `validate_document` tool takes an optional `uf` field used only for RG. SP & RJ ship in v1; other UFs return `ErrUFNotImplemented`.

---

## 3. Repository Layout — Root Package + Adapter Subpackages

```
<module root>/                 package brdoc
  document.go                  Kind, Document, OriginResolver, UFScoped, UF type
  registry.go                  Register / Get / Kinds / Validate / Generate / Format / Detect
  errors.go                    sentinel errors
  uf.go                        UF type + 27 constants, UF lookup tables (CEP ranges, DDD map)
  meta.go                      brand-name constants (binary name, CLI Use, MCP server name) — see §10
  cpf.go        cpf_test.go
  cnpj.go       cnpj_test.go
  cnh.go        cnh_test.go
  pis.go        pis_test.go
  renavam.go    renavam_test.go
  voterid.go    voterid_test.go
  cep.go        cep_test.go
  phone.go      phone_test.go
  plate.go      plate_test.go
  cns.go        cns_test.go
  rg.go         rg_test.go
  pix.go        pix_test.go
  doc.go                       package godoc + runnable Example* functions
cmd/brdoc/
  main.go                      Cobra root, registry-driven subcommands, `mcp` subcommand
mcp/
  server.go    server_test.go  go-sdk MCP adapter over the registry (in-memory transport tests)
compat/
  compat.go    compat_test.go  paemuri-exact Is* signatures
Taskfile.yml   .golangci.yml   docs/
```

Import path stays the module root → `brdoc.NewCPF()`. Adapters import the root package; the root package imports neither adapter (clean dependency direction).

---

## 4. v1 Document Coverage

| Kind | Validate | Generate | Format (mask) | Origin | Notes |
|---|:--:|:--:|---|:--:|---|
| CPF | ✅ | ✅ | `###.###.###-##` | ✅ region (9th digit) | exists; migrate to pattern |
| CNPJ | ✅ | ✅ | `##.###.###/####-##` | — | exists; alphanumeric + legacy |
| CNH | ✅ | ✅ | identity (11 digits) | — | two DV, −2 offset |
| PIS/PASEP/NIS | ✅ | ✅ | `###.#####.##-#` | — | single mod-11 DV |
| RENAVAM | ✅ | ✅ | identity (11 digits) | — | `(sum*10)%11` |
| Título Eleitoral | ✅ | ✅ | spaced groups (opt) | ✅ UF code | UF `01..28` + 2 DV |
| CEP | ✅ | ✅ (range-aware) | `#####-###` | ✅ UF (prefix range) | format + UF range |
| Phone | ✅ | ✅ (DDD-aware) | `(##) #####-####` | ✅ UF (DDD) | 8/9-digit subscriber |
| License plate | ✅ | ✅ (national/Mercosul) | dash insert/strip | — | regex only |
| CNS | ✅ | ✅ (constructive) | identity (15 digits) | — | `sum%11==0`, weights 15..1 |
| RG (SP/RJ) | ✅ (per UF) | ✅ | `##.###.###-#` | — | `UFScoped`; v2 expands |
| **PIX key** | ✅ | ✅ (EVP/UUIDv4) | identity | — | dispatches to CPF/CNPJ/email/E.164/EVP-UUID |

**Algorithms:** fully specified in [`docs/FEATURE-GAP-paemuri.md`](../../FEATURE-GAP-paemuri.md) §3. They are the implementation contract; reproduce them in code with table tests pinning the documented samples.

**PIX key validation** (`pix.go`): accept the 5 BCB key kinds — CPF, CNPJ, email (RFC 5322-lite), phone (E.164 `+55…` via the phone validator), and EVP (random key = UUIDv4). `Validate` returns true if the value is a well-formed key of *any* kind; an exported `DetectPIXKind(value) (string, bool)` reports which. `Generate` returns a random EVP key (UUIDv4 — itself a valid PIX key); `Format` returns the cleaned value (identity). `ErrUnsupported` is defined but reserved for future use.

---

## 5. Public API Surface

### 5.1 Root package (`brdoc`)
- Concrete constructors + methods: `NewCPF()`, `NewCNPJ()`, … `(*CPF).Validate/Generate/Format`, `(*CPF).Origin`.
- Registry dispatchers: `Validate(kind,value)`, `Generate(kind)`, `Format(kind,value)`, `Detect(value)`, `Kinds()`.
- `ValidateDocument(doc)` retained as a **deprecated** wrapper over `Detect`+`Validate`.

### 5.2 `compat/` subpackage (paemuri drop-in)
Mirror paemuri/brdoc v3 signatures exactly so migration is an import swap:
```go
func IsCPF(s string) bool
func IsCNPJ(s string) bool
func IsCNH(s string) bool
func IsPIS(s string) bool
func IsRENAVAM(s string) bool
func IsVoterID(s string) bool
func IsCNS(s string) bool
func IsPlate(s string) bool
func IsNationalPlate(s string) bool
func IsMercosulPlate(s string) bool
func IsCEP(s string) (bool, UF)
func IsCEPFrom(s string, ufs ...UF) bool
func IsPhone(s string) (bool, UF)
func IsPhoneFrom(s string, ufs ...UF) bool
func IsRG(s string, uf UF) (bool, error)
```
All implemented as thin wrappers over the root registry/types. (A README migration note documents the one-line swap.)

### 5.3 Error model (`errors.go`)
```go
var (
    ErrInvalidLength    = errors.New("brdoc: invalid document length")
    ErrInvalidFormat    = errors.New("brdoc: invalid document format")
    ErrUnknownKind      = errors.New("brdoc: unknown document kind")
    ErrUnsupported      = errors.New("brdoc: operation not supported for this kind")
    ErrUFNotImplemented = errors.New("brdoc: federative unit not implemented")
)
```
Wrap with `%w` where context is added. All comparisons use `errors.Is`/`errors.As`. `CPF.Format`/`CNPJ.Format` are retrofitted to return these sentinels.

---

## 6. CLI Design (`cmd/brdoc`)

- Subcommands are **generated by iterating `brdoc.Kinds()`** — adding a type to the registry automatically yields its subcommand (no boilerplate drift).
- Per-kind command: `brdoc <kind> [-g|--generate] [-v|--validate VALUE] [--format VALUE] [--origin VALUE] [-f|--from FILE|-] [-n|--count N] [--uf SP]`.
  - `--uf` is only meaningful for RG.
  - `--origin` only present for `OriginResolver` types.
- Top-level: `brdoc detect <value>` (auto-detect kind), `brdoc mcp` (start MCP server), `brdoc version`.
- The existing `bufio.Scanner` bulk `--from` streaming path (1 MB max line, file/stdin via `-`) is **extracted into a shared helper** and reused by every kind.
- Preserve current UX niceties: `SilenceUsage`/`SilenceErrors`, exit code 1 on any invalid input, Windows-friendly examples.

---

## 7. MCP Server (`brdoc mcp`, stdio)

- go-sdk (`github.com/modelcontextprotocol/go-sdk/mcp`), `StdioTransport` (blocks). **Logger → stderr** (stdout is JSON-RPC).
- Server: `Name` sourced from `meta.go`; `Version` from build info.
- **Tools** (typed inputs with `jsonschema` tags; `kind` enums sourced from `brdoc.Kinds()`):
  | Tool | Input | Output |
  |---|---|---|
  | `validate_document` | `{kind, value, uf?}` | `{valid bool, origin?}` |
  | `generate_document` | `{kind, count?}` | `{values []string}` |
  | `format_document` | `{kind, value}` | `{formatted string}` |
  | `detect_document` | `{value}` | `{kind string, valid bool}` |
  | `list_document_types` | `{}` | `{kinds []string}` |
- Errors surface via `result.IsError = true` + `TextContent`.
- **Tests:** in-memory transports (`mcp.NewInMemoryTransports()`), assert each tool round-trips.

---

## 8. Testing Strategy (TDD)

Per the user's Go standards — TDD, table-driven, ≥80% coverage:
- **Table-driven** per type: real valid samples, all-equal rejection, wrong length, off-by-one DV, bad UF code, formatted vs unformatted input.
- **Fuzz** (Go 1.24 native): `FuzzXxxValidate` asserting (a) `Generate→Validate` round-trip is always true, (b) arbitrary input never panics.
- **Regression pins:** paemuri's `IsCNPJ` false-negative case `39591842000010` (issue #26/#27) added to CNPJ tests; alphanumeric-CNPJ samples retained.
- **Godoc `Example*`** functions per type (render on pkg.go.dev).
- **MCP** in-memory transport tests; **compat** tests asserting parity with root behavior.
- `testify` (already a dep) for assertions.
- **Taskfile:** `test` runs `-short`; `test:full` runs the full suite incl. fuzz seed corpus. `golangci-lint run --fix ./... --timeout=5m` gate.
- Coverage profile written to temp with datetime suffix (per global standards).

---

## 9. Cross-Cutting Enhancements (folded into the above)

1. Unified `Validate(kind,value)` / `Generate` / `Format` dispatch (registry) — §2.3.
2. `Detect` auto-detection generalizing `ValidateDocument` — §2.3.
3. Sentinel errors + `errors.Is` — §5.3.
4. `Origin`/UF inference for CPF, CEP, phone, voter ID — §2.2, §4.
5. `Generate` + `Format` for every type (paemuri has neither) — §4.
6. Shared bulk `--from` helper — §6.
7. Fuzz tests, godoc examples — §8.
8. PIX key validation — §4.
9. `UF` type + 27 constants consolidated in `uf.go` (ported verbatim from paemuri — Unlicense permits it; add the project's own license header) — §3.

---

## 10. Rebrand Readiness (for later `/branding:names`)

All brand-bearing strings live in **one place** (`meta.go`): binary name, Cobra root `Use`/`Short`, MCP server `Name`. The Go *package* name stays `brdoc` (it is a domain term — "BR doc(uments)" — not a brand) to avoid churn, unless branding explicitly requires otherwise. Post-branding, the rename reduces to:
1. `go.mod` module path → new path.
2. Repo-wide import rewrite.
3. `meta.go` constant values.
4. Binary/output artifact names (GoReleaser config).

No brand name is hard-coded across many files, so the rename is mechanical and low-risk.

---

## 11. Build Sequence (Milestones)

| Milestone | Deliverable | Notes |
|---|---|---|
| **M0 — Foundation** | `document.go`, `registry.go`, `errors.go`, `uf.go`, `meta.go`; migrate **CPF & CNPJ** onto the interface/registry with **zero behavior change**; `compat/` skeleton; `math/rand/v2` swap | All existing tests stay green; `ValidateDocument` becomes deprecated wrapper |
| **M1 — CLI engine** | Registry-driven CLI subcommand generation; `detect`; shared bulk `--from` helper; `version` | CPF/CNPJ CLI behavior preserved |
| **M2 — Type breadth** | Add PIS, RENAVAM → CNH, Voter ID → CEP, phone, plate, CNS, RG(SP/RJ) | Each: `<type>.go` + `<type>_test.go` + registry registration; CLI/compat light up automatically |
| **M3 — MCP** | `mcp/server.go` + `brdoc mcp` subcommand + in-memory tests | go-sdk stdio |
| **M4 — PIX** | `pix.go` + tests + CLI `pix` + MCP coverage | reuse CPF/CNPJ/phone validators |
| **M5 — Hardening** | Fuzz tests, godoc examples, README/docs refresh, coverage ≥80%, lint gate, Taskfile `test`/`test:full` | regression pins included |
| **v2 (post-rebrand)** | Inscrição Estadual (27 UFs), multi-state RG | scoped separately |
| **Rebrand** | `/branding:names` → apply per §10 | mechanical rename |

---

## 12. Risks & Open Questions

- **RG signature divergence** — handled via `UFScoped` + CLI `--uf` + optional MCP field; accepted as the one non-uniform type.
- **CNS / CEP / phone generation** require constructive or range-aware generation (not pure random) to produce *valid* fakes — flagged as the higher-effort generators in M2/M4.
- **CEP range & DDD→UF tables** must be sourced (port from paemuri, Unlicense) and kept current; treated as data in `uf.go`.
- **Deprecation:** `ValidateDocument` wrapper carries a `// Deprecated:` note with a removal date ≥30 days out, logged on use, tracked in `docs/BACKLOG.md` (per global policy).

---

## 13. Definition of Done (v1)

- [ ] 11 standard types + PIX implemented with Validate/Generate/Format/Origin per §4.
- [ ] CLI auto-derives all subcommands from the registry; `detect`, `mcp`, bulk `--from` work.
- [ ] `brdoc mcp` serves the 5 tools over stdio; in-memory tests pass.
- [ ] `compat/` mirrors paemuri signatures; parity tests pass.
- [ ] Sentinel errors + `errors.Is` throughout.
- [ ] Fuzz + table tests + godoc examples; coverage ≥80%; lint-clean.
- [ ] Regression pins (CNPJ `39591842000010`) green.
- [ ] Brand strings isolated in `meta.go`; rename path documented.
