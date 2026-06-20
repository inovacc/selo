# 🚀 selo - Setup Guide

## 📦 Package Contents

This repository contains the complete **selo** toolkit for validating, generating, formatting, and geolocating Brazilian documents — 13 kinds (CPF, CNPJ, CNH, PIS/PASEP/NIS, RENAVAM, Título Eleitoral, CEP, phone, license plate, CNS, RG, Inscrição Estadual, PIX) — exposed as a library, a CLI, and an MCP server.

### 📂 Project Structure

```
selo/
├── cmd/
│   └── selo/                   # Cobra CLI (subcommand per kind; detect/person/gen/mcp/version)
├── cpf.go, cnpj.go, rg.go, ie.go, …  # One file per document type
├── document.go                 # Document interface + Kind constants
├── registry.go                 # Self-registering type registry
├── person.go                   # GeneratePerson (synthetic identities; seedable)
├── compat/                     # paemuri/brdoc drop-in compat layer
├── mcp/                        # MCP server (stdio; 7 tools)
├── internal/codegen/           # multi-language code generation (selo gen)
├── generated/                  # committed reference output per target language
├── cpf_cnpj_test.go, *_test.go # Test suites (per type)
├── doc.go                      # Package documentation
├── docs/                       # ROADMAP, ARCHITECTURE, CODEGEN, ADRs, …
├── CHANGELOG.md                # Version history
├── LICENSE                     # MIT License
├── README.md                   # Complete documentation
├── go.mod                      # Module configuration (Go 1.25)
└── go.sum
```

## 🔧 Installation Steps

### 1. Clone or upload to GitHub

```bash
# Extract the ZIP
git clone https://github.com/inovacc/selo.git
cd selo
```

### 2. Verify Installation

```bash
# Run tests
go test -v ./...

# Run benchmarks
go test -bench=. -benchmem

# Check coverage
go test -cover
```

### 3. Try the CLI

```bash
# Install the CLI
go install github.com/inovacc/selo/cmd/selo@latest

# Single operations
selo cpf  --generate
selo cnpj --generate
selo cpf  --validate 123.456.789-09
selo cnpj --validate 12.ABC.345/01DE-35

# Bulk validation
selo cpf  --validate --from cpfs.txt
selo cnpj --validate --from cnpjs.txt
type cpfs.txt  | selo cpf  --validate --from -
type cnpjs.txt | selo cnpj --validate --from -

# Generate many
selo cpf  --generate --count 10
selo cnpj --generate --count 5

# Synthetic identities (deterministic with --seed)
selo person --uf SP --json
selo person --uf SP --seed 42 --count 3 --json

# Generate validators in other languages (ts, js, ruby, java, csharp, python)
selo gen --lang python --kind all --out ./generated/python

# Run the MCP server over stdio
selo mcp
```

## 📚 Usage in Your Project

### Install the package

```bash
go get github.com/inovacc/selo
```

### Import and use

```go
package main

import (
  "fmt"
  "github.com/inovacc/selo"
)

func main() {
  // CPF
  cpf := selo.NewCPF()
  fmt.Println(cpf.Validate("123.456.789-09")) // true or false

  // CNPJ
  cnpj := selo.NewCNPJ()
  fmt.Println(cnpj.Validate("12.ABC.345/01DE-35")) // true or false
}
```

## 🧪 Quality Checks

### Run all tests

```bash
go test -v ./...
```

We use the `testify` assertion library (`assert`/`require`) for clearer tests. Typical pattern:

```go
result, err := DoThing()
require.NoError(t, err)
assert.Equal(t, "expected", result)
```

### Check test coverage

```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run linter

```bash
golangci-lint run
```

### Run benchmarks

```bash
go test -bench=. -benchmem
```

## 📖 Documentation

### Generate godoc

```bash
godoc -http=:6060
# Visit: http://localhost:6060/pkg/github.com/inovacc/selo/
```

### View online

After pushing to GitHub:

- https://pkg.go.dev/github.com/inovacc/selo

## 🔐 Security

### Report vulnerabilities

```bash
# Check for known vulnerabilities (also enforced in CI via the quality-check workflow)
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## 🚀 Release Process

### Creating a new release

1. Update CHANGELOG.md
2. Update version in go.mod (if needed)
3. Commit changes:
   ```bash
   git commit -m "Release v0.1.0"
   ```
4. Create and push tag:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```
5. Create release on GitHub with release notes

## 🎯 Next Steps

1. ✅ Upload to GitHub: `https://github.com/inovacc/selo`
2. ✅ Enable GitHub Actions (CI will run automatically)
3. ✅ Add repository description and topics
4. ✅ Releases published through v1.4.0 (see CHANGELOG.md)
5. 📝 Share on social media/communities
6. 📊 Monitor usage via pkg.go.dev

## 📊 Badges Setup

Add these to your README (after first release):

```markdown
[![Go Reference](https://pkg.go.dev/badge/github.com/inovacc/selo.svg)](https://pkg.go.dev/github.com/inovacc/selo)
[![Go Report Card](https://goreportcard.com/badge/github.com/inovacc/selo)](https://goreportcard.com/report/github.com/inovacc/selo)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
```

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## 📞 Support

- 🐛 Issues: https://github.com/inovacc/selo/issues
- 💬 Discussions: https://github.com/inovacc/selo/discussions

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Made with ❤️ by INOVACLOUD CONSULTORIA LTDA**

Repository: https://github.com/inovacc/selo
