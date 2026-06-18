# 🇧🇷 brdoc

[![Go Reference](https://pkg.go.dev/badge/github.com/inovacc/brdoc.svg)](https://pkg.go.dev/github.com/inovacc/brdoc)
[![Go Report Card](https://goreportcard.com/badge/github.com/inovacc/brdoc)](https://goreportcard.com/report/github.com/inovacc/brdoc)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A complete Go toolkit to **validate, generate, format, and geolocate** Brazilian documents —
exposed identically through a **library**, a **CLI**, and an **MCP server**.

Unlike validation-only libraries, every supported document type can also be *generated*
(valid fake data) and *formatted* (canonical mask), and geolocatable documents resolve their
issuing federative unit (UF).

## ✨ Supported documents

| Kind (`brdoc.Kind`) | Validate | Generate | Format | Origin (UF) |
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
| **RG** (SP/RJ) | ✅ | ✅ | `##.###.###-#` | — |
| **PIX key** (CPF/CNPJ/email/phone/EVP) | ✅ | ✅ (EVP) | identity | — |

## 📦 Install

```bash
go get github.com/inovacc/brdoc
```

Requires **Go 1.25+** (the MCP server depends on `modelcontextprotocol/go-sdk`, which requires
Go 1.25; the core library and CLI otherwise have only Cobra as a runtime dependency).

## 🔧 Library usage

### Ergonomic per-type API

```go
import "github.com/inovacc/brdoc"

cpf := brdoc.NewCPF()
cpf.Validate("529.982.247-25")     // true (accepts formatted or raw)
cpf.Generate()                     // a fresh valid CPF
cpf.Format("52998224725")          // "529.982.247-25"
cpf.Origin("52998224725")          // issuing region

brdoc.NewCEP().Origin("01310-100")          // "SP"
brdoc.NewPhone().Origin("(11) 98765-4321")  // "SP"
```

### Generic, registry-driven API

Every type self-registers, so you can dispatch by `Kind`:

```go
ok, err := brdoc.Validate(brdoc.KindCNH, "12345678900")
s,  err := brdoc.Generate(brdoc.KindPIS)
m,  err := brdoc.Format(brdoc.KindCEP, "01310100")     // "01310-100"
kind, ok := brdoc.Detect("529.982.247-25")             // auto-detect (KindCPF, true)
brdoc.Kinds()                                          // all registered kinds, sorted
```

### Errors

Failures use comparable sentinels (`errors.Is`):

```go
_, err := brdoc.NewCPF().Format("123")
errors.Is(err, brdoc.ErrInvalidLength) // true
```

`ErrInvalidLength`, `ErrInvalidFormat`, `ErrUnknownKind`, `ErrUnsupported`, `ErrUFNotImplemented`.

### PIX keys

```go
brdoc.NewPIX().Validate("529.982.247-25")        // true (CPF key)
brdoc.DetectPIXKind("+5511998765432")            // ("phone", true)
brdoc.NewPIX().Generate()                        // a random EVP (UUIDv4) key
```

## 🖥️ CLI

The CLI derives one subcommand per registered kind automatically:

```bash
go run ./cmd/brdoc cpf --generate --count 5
go run ./cmd/brdoc cnpj --validate 39.591.842/0000-10
go run ./cmd/brdoc pis --format 12001234564
go run ./cmd/brdoc cep --origin 01310-100          # -> SP
go run ./cmd/brdoc rg --validate 24.678.131-2 --uf SP
go run ./cmd/brdoc cpf --validate --from cpfs.txt  # bulk; '-' reads stdin
go run ./cmd/brdoc detect 529.982.247-25           # auto-detect kind
go run ./cmd/brdoc version
```

Flags per kind: `-g/--generate`, `-v/--validate`, `--format`, `--origin` (geolocatable kinds),
`-f/--from FILE|-` (bulk), `-n/--count N`, `--uf` (RG only). **Exit code is `1` when a document
is invalid** (scriptable); genuine errors also exit `1`.

## 🤖 MCP server

Expose the toolkit to AI agents over stdio:

```bash
go run ./cmd/brdoc mcp
```

Tools (kind enums sourced from the registry): `validate_document`, `generate_document`,
`format_document`, `detect_document`, `list_document_types`. Logs go to stderr; the protocol
runs on stdin/stdout.

## 🔁 Migrating from `paemuri/brdoc`

The `compat` subpackage mirrors `paemuri/brdoc` v3's flat `Is*` API, so migration is a one-line
import swap:

```go
import "github.com/inovacc/brdoc/compat" // was: github.com/paemuri/brdoc/v3

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

See [`docs/BACKLOG.md`](docs/BACKLOG.md). Highlights: a **fake-person generator** (one coherent
synthetic identity carrying every document type), **Inscrição Estadual** (per-UF), and
**multi-state RG**.

## 📄 License

MIT © Dyam Marcano. See [LICENSE](LICENSE).
