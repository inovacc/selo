# Contributing to selo

First off, thank you for considering contributing to selo! 🎉

## 🤝 How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples**
- **Describe the behavior you observed and what you expected**
- **Include Go version and OS information**

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear and descriptive title**
- **Provide a detailed description of the suggested enhancement**
- **Explain why this enhancement would be useful**
- **List any similar features in other libraries**

### Pull Requests

1. Fork the repository
2. Create a new branch from `main`:
   ```bash
   git checkout -b feature/amazing-feature
   ```

3. Make your changes following our coding standards:

- Write clear, commented code
- Follow Go best practices and idioms
- Add tests for new functionality
- Update documentation as needed

4. Ensure all tests pass:
   ```bash
   go test -v ./...
   go test -race ./...
   ```

5. Format your code:
   ```bash
   go fmt ./...
   ```

6. Run linter:
   ```bash
   golangci-lint run
   ```

7. Commit your changes (the project uses [Conventional Commits](https://www.conventionalcommits.org/);
   no AI attribution / `Co-Authored-By` lines):
   ```bash
   git commit -m 'feat: add amazing feature'
   ```

8. Push to your fork:
   ```bash
   git push origin feature/amazing-feature
   ```

9. Open a Pull Request

## 📝 Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` for formatting
- Write clear and concise comments
- Keep functions focused and small
- Use meaningful variable and function names

## 🧪 Testing

- Write unit tests for all new functionality
- Maintain test coverage above 90%
- Add benchmark tests for performance-critical code
- Test edge cases and error conditions
- Use `testify`'s `require`/`assert` for clearer, concise assertions

Example test structure (using testify):

```go
package yourpkg

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid case", "input", "output", false},
        {"error case", "bad", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Feature(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## 📚 Documentation

- Update README.md for user-facing changes
- Add godoc comments for exported functions and types
- Include examples in documentation
- Update CHANGELOG.md (if exists)

## 🏗️ Project Structure

```
selo/
├── *.go                  # core library: one document type per file
│                         # (cpf.go, cnpj.go, cnh.go, …, rg.go, ie.go, pix.go),
│                         # plus document.go (interface), registry.go, person.go
├── compat/               # paemuri/brdoc Is* drop-in compat layer
├── cmd/selo/             # Cobra CLI (subcommand per kind; detect/person/gen/mcp/version)
├── mcp/                  # stdio MCP server (registry-backed tools)
├── internal/codegen/     # multi-language code generation (selo gen)
├── generated/            # committed reference output per target language
├── docs/                 # ROADMAP, MILESTONES, ARCHITECTURE, CODEGEN, …
├── Taskfile.yml          # task runner (test/lint/cover/gen)
├── go.mod
├── LICENSE
└── README.md
```

## 🔍 Code Review Process

1. Maintainers will review your PR
2. Address any feedback or requested changes
3. Once approved, a maintainer will merge your PR

## 📋 Checklist

Before submitting your PR, ensure:

- [ ] Code follows project style guidelines
- [ ] Tests are added/updated and passing
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive
- [ ] No unnecessary dependencies added
- [ ] Code is formatted with `go fmt`
- [ ] No linter warnings

## 🎯 Priority Areas

We're particularly interested in contributions for:

- **Inscrição Estadual breadth** — UFs beyond SP (authoritative algorithm + ≥2 verifiable samples
  required; see [docs/IE-NOTES.md](docs/IE-NOTES.md))
- **Multi-state RG** — UFs beyond SP, each with a sourced algorithm + samples
- **Additional code-generation target languages** (beyond TS / JS / Ruby / Java / C# / Python)
- Performance improvements, bug fixes, and test-coverage improvements

## ❓ Questions?

Feel free to open an issue for questions or reach out through:

- GitHub Issues
- GitHub Discussions

## 📜 License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to selo! 🚀
