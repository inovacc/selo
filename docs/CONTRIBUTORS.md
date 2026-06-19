# Contributors & Contributing Guide

## Maintainers
- **Dyam Marcano** (<dyam.marcano@gmail.com>) — author & maintainer.
- Organization: **inovacc** — <https://github.com/inovacc/selo>.

See the git history for the full contributor list (`git shortlog -sne`).

> A root [`CONTRIBUTING.md`](../CONTRIBUTING.md) also exists; this file is the docs-tree summary
> and toolchain reference. Where they differ, the root file governs PR mechanics.

## Toolchain
- **Go 1.25+** — pinned by the `go` directive in [`go.mod`](../go.mod) (the MCP `go-sdk` requires it).
- **golangci-lint** — used for format + lint (`task lint`).
- **Task** ([taskfile.dev](https://taskfile.dev)) — task runner; see [`Taskfile.yml`](../Taskfile.yml).
- **goreleaser** — release/build snapshots (`task build-dev`, `task build-prod`).

## Common commands
```bash
task test        # fast unit tests (go test -short ./...)
task test:full   # full suite: race, fuzz seed corpus, benchmarks
task lint        # golangci-lint fmt + run
task cover       # coverage profile to the system temp dir, prints total
```
Without Task, the equivalents are `go test -short ./...`, `go test -race -p=1 ./...`,
`golangci-lint run --timeout=5m`, and
`go test -covermode=atomic -coverprofile=cover.out ./... && go tool cover -func=cover.out`.

## Conventions
- **Conventional commits** — the history uses `feat:`, `fix:`, `refactor:`, `test:`, `docs:`,
  `ci:` (with scopes, e.g. `fix(cnpj): …`). No AI attribution / `Co-Authored-By` lines.
- **One type per file** — each document type lives in its own file (`cpf.go`, `cnpj.go`, `rg.go`,
  `ie.go`, …); mirror the nearest existing type when adding one.
- **Self-registration** — new document types register in `init()` via `Register(&T{})`; the CLI and
  MCP server then pick them up automatically (no per-type edits there).
- **Table-driven tests** with `Generate → Validate` round-trip invariants, native fuzz targets per
  check-digit type, and runnable godoc examples.
- **Errors** — comparable sentinels checked with `errors.Is` (never `==`).
- **License:** MIT (see [LICENSE](../LICENSE)); keep new files consistent with it.

## Adding a new document type (checklist)
1. Create `>kind<.go` (`package selo`) implementing `Document`; add `+ UFScoped`/`OriginResolver`
   if applicable. Mirror the closest existing type.
2. Add a `Kind>Name<` constant to `document.go`.
3. `func init() { Register(&>Kind<{}) }`.
4. Add `>kind<_test.go`: table tests, round-trip, fuzz, and — for check-digit types —
   **externally-sourced** sample(s) (never invent a sample to make a test pass).
5. `task test:full && task lint`. CLI/MCP need no edits (registry-derived).

## Adding a new UF to a UF-scoped type (RG / IE)
- Provide an **authoritative algorithm** and **≥2 independently-sourced samples** verified by hand.
  If you cannot, mark the UF `needs-research` (see [IE-NOTES.md](IE-NOTES.md)) — do not ship it.
