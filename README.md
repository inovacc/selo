# 🇧🇷 Selo

[![Go Reference](https://pkg.go.dev/badge/github.com/inovacc/selo.svg)](https://pkg.go.dev/github.com/inovacc/selo)
[![Go Report Card](https://goreportcard.com/badge/github.com/inovacc/selo)](https://goreportcard.com/report/github.com/inovacc/selo)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A complete Go toolkit to **validate, generate, format, and geolocate** Brazilian documents —
exposed identically through a **library**, a **CLI**, and an **MCP server**.

Unlike validation-only libraries, every supported document type can also be *generated*
(valid fake data) and *formatted* (canonical mask), and geolocatable documents resolve their
issuing federative unit (UF).

## ✨ Supported documents

| Kind (`selo.Kind`) | Validate | Generate | Format | Origin (UF) |
|---|:--:|:--:|:--:|:--:|
| **CPF** | ✅ | ✅ | `###.###.###-##` | ✅ region |
| **CNPJ** (incl. alphanumeric) | ✅ | ✅ | `##.###.###/####-##` | — |
| **CNH** (driver's license) | ✅ | ✅ | identity | — |
| **PIS/PASEP/NIS** | ✅ | ✅ | `###.#####.##-#` | — |
| **RENAVAM** (vehicle) | ✅ | ✅ | identity | — |
| **Título Eleitoral** (voter ID) | ✅ | ✅ | grouped | ✅ UF code |
| **CEP** (postal code) | ✅ | ✅ | `#####-###` | ✅ UF range |
| **Phone** (BR telephone) | ✅ | ✅ | `(##) #####-####` | ✅ DDD |
| **License plate** (national + Mercosul) | ✅ | ✅ | dash | — |
| **CNS** (health card) | ✅ | ✅ | identity | — |
| **RG** (SP) | ✅ | ✅ | `##.###.###-#` | — |
| **Inscrição Estadual** (SP) | ✅ | ✅ | `###.###.###.###` | — |
| **PIX key** (CPF/CNPJ/email/phone/EVP) | ✅ | ✅ (EVP) | identity | — |

`RG` and `Inscrição Estadual` are **UF-scoped** (`selo.UFScoped`): `ValidateUF(value, uf)` /
`ImplementedUFs()`. RG ships SP (RJ uses a different, unverified algorithm — see ISSUES); IE ships SP, with more states tracked in
[`docs/IE-NOTES.md`](docs/IE-NOTES.md).

## 📦 Install

**Library:**
```bash
go get github.com/inovacc/selo
```

**CLI** — installs the `selo` binary into `$(go env GOBIN)` (or `$(go env GOPATH)/bin`):
```bash
go install github.com/inovacc/selo/cmd/selo@latest   # or @v1.1.0 to pin a release
```
Make sure that directory is on your `PATH`, then run `selo --help`.

Requires **Go 1.25+** (the MCP server depends on `modelcontextprotocol/go-sdk`, which requires
Go 1.25; the core library and CLI otherwise have only Cobra as a runtime dependency).

## 🔧 Library usage

### Ergonomic per-type API

```go
import "github.com/inovacc/selo"

cpf := selo.NewCPF()
cpf.Validate("529.982.247-25")     // true (accepts formatted or raw)
cpf.Generate()                     // a fresh valid CPF
cpf.Format("52998224725")          // "529.982.247-25"
cpf.Origin("52998224725")          // issuing region

selo.NewCEP().Origin("01310-100")          // "SP"
selo.NewPhone().Origin("(11) 98765-4321")  // "SP"
```

### Generic, registry-driven API

Every type self-registers, so you can dispatch by `Kind`:

```go
ok, err := selo.Validate(selo.KindCNH, "12345678900")
s,  err := selo.Generate(selo.KindPIS)
m,  err := selo.Format(selo.KindCEP, "01310100")     // "01310-100"
kind, ok := selo.Detect("529.982.247-25")             // auto-detect (KindCPF, true)
selo.Kinds()                                          // all registered kinds, sorted
```

### Errors

Failures use comparable sentinels (`errors.Is`):

```go
_, err := selo.NewCPF().Format("123")
errors.Is(err, selo.ErrInvalidLength) // true
```

`ErrInvalidLength`, `ErrInvalidFormat`, `ErrUnknownKind`, `ErrUnsupported`, `ErrUFNotImplemented`.

### PIX keys

```go
selo.NewPIX().Validate("529.982.247-25")        // true (CPF key)
selo.DetectPIXKind("+5511998765432")            // ("phone", true)
selo.NewPIX().Generate()                        // a random EVP (UUIDv4) key
```

### Synthetic people (GenPerson)

Generate one coherent fake identity carrying **every** document type — all valid and
sharing the same UF (CPF region, Voter-ID code, phone DDD, and CEP all agree). For
test data / fixtures only (synthetic, never real PII):

```go
p := selo.GeneratePerson(selo.WithUF(selo.UFSP), selo.WithVehicle(), selo.WithCompany())
// p.CPF, p.RG, p.CNH, p.PIS, p.Renavam, p.VoterID, p.CNS, p.CEP, p.Phone, p.PIXKeys, p.Vehicle, p.Company
```

## 🖥️ CLI

The CLI derives one subcommand per registered kind automatically. After `go install` (above)
run `selo <cmd>`; from a checkout, use `go run ./cmd/selo <cmd>`:

```bash
go run ./cmd/selo cpf --generate --count 5
go run ./cmd/selo cnpj --validate 39.591.842/0000-10
go run ./cmd/selo pis --format 12001234564
go run ./cmd/selo cep --origin 01310-100          # -> SP
go run ./cmd/selo rg --validate 24.678.131-2 --uf SP
go run ./cmd/selo cpf --validate --from cpfs.txt  # bulk; '-' reads stdin
go run ./cmd/selo detect 529.982.247-25           # auto-detect kind
go run ./cmd/selo version
go run ./cmd/selo person --uf SP --count 5 --json   # synthetic people
```

Flags per kind: `-g/--generate`, `-v/--validate`, `--format`, `--origin` (geolocatable kinds),
`-f/--from FILE|-` (bulk validate), `-n/--count N`, `-b/--bulk N` (bulk generate — implies
`--generate`), `--uf` (RG only). **Exit code is `1` when a document is invalid** (scriptable);
genuine errors also exit `1`.

## 🤖 MCP server

Expose the toolkit to AI agents over stdio (`selo mcp` once installed):

```bash
go run ./cmd/selo mcp
```

Tools (kind enums sourced from the registry): `validate_document`, `generate_document`,
`format_document`, `detect_document`, `list_document_types`, `generate_person`, `generate_code`.
Logs go to stderr; the protocol runs on stdin/stdout.

## 🌍 Code generation

Generate validators in other languages from the *same verified algorithms*. `selo gen` emits
**validate / format / origin / generate** code for all 13 kinds in **TypeScript, JavaScript, Ruby,
Java, C#, and Python**, each shipped with Go-produced golden test vectors and a runnable test suite:

```bash
selo gen --lang ts     --kind cpf --out ./out      # one kind, one language
selo gen --lang python --kind all --out ./generated/python   # all 13 kinds
```

Supported languages: `ts`, `js`, `ruby`, `java`, `csharp`, `python`. A CI matrix runs each target's golden
vectors on real toolchains, so a wrong port fails its own tests. The MCP `generate_code` tool
returns the same file set. Full details in [`docs/CODEGEN.md`](docs/CODEGEN.md).

## 🔁 Migrating from `paemuri/brdoc`

The `compat` subpackage mirrors `paemuri/brdoc` v3's flat `Is*` API, so migration is a one-line
import swap:

```go
import "github.com/inovacc/selo/compat" // was: github.com/paemuri/brdoc/v3

compat.IsCPF("529.982.247-25")
compat.IsCEP("01310-100")                    // (bool, UF)
compat.IsRG("24.678.131-2", compat.UF("SP")) // (bool, error)
```

A compile-time signature-parity guard keeps the wrappers aligned with the upstream API.

## 🧪 Development

```bash
task test        # fast unit tests (-short)
task test:full   # full suite incl. fuzz seed corpus
task lint        # golangci-lint
task cover       # coverage profile (written to the system temp dir)
```

Tests are table-driven with `Generate->Validate` round-trip invariants, native fuzz targets per
check-digit type, and runnable godoc examples.

## 🗺️ Roadmap

See [`docs/ROADMAP.md`](docs/ROADMAP.md) and [`docs/BACKLOG.md`](docs/BACKLOG.md). **Inscrição
Estadual** shipped its first state (SP) — remaining UFs are tracked in
[`docs/IE-NOTES.md`](docs/IE-NOTES.md). Other highlights: **multi-state RG** and reproducible
`GenPerson` output via a seed. The `GenPerson` generator itself is **shipped**.

## 📄 License

MIT © Dyam Marcano. See [LICENSE](LICENSE).
