# Known Issues & Limitations

Documented limitations and constraints of `github.com/inovacc/selo` as of 2026-06-19. These are
known, by-design, or tracked — not regressions. Bugs (if any) live in [BUGS.md](BUGS.md);
prioritized work in [BACKLOG.md](BACKLOG.md).

## Coverage / scope limitations

### RG supports only SP and RJ
`RG.Validate`/`ValidateUF` implement the SP/RJ check-digit algorithm only; every other UF returns
`(false, ErrUFNotImplemented)`. **RG check digits are not nationally standardized** — other states
use different schemes. *Workaround:* check `ImplementedUFs()` before relying on a UF.

### RJ RG uses the SP algorithm but is not independently verified
The code applies the same algorithm to SP and RJ. The **SP** convention is verified against four
independent sources (see `TestRG_AuthoritativeSamples` and BACKLOG); no independent **RJ**-specific
source was found. Verify RJ before depending on it or building multi-state RG on top.

### Inscrição Estadual supports only SP
Only SP is implemented and verified (two sourced samples). MG/RJ/RS/PR were researched but deferred
for lack of ≥2 verifiable samples; the other 22 UFs are unstarted. Unsupported UFs return
`ErrUFNotImplemented`. Roadmap and per-UF research in [IE-NOTES.md](IE-NOTES.md).

### Sample provenance for RG and IE
The pinned regression samples are authoritative-tutorial / official-documentation worked examples,
**not confirmed real-person/company registrations** (real numbers are not publicly verifiable, for
privacy reasons). Their check digits were verified by hand and by an independent reviewer. This is
documented honestly in the tests and IE-NOTES.md.

## Behavioral limitations

### `GeneratePerson` and generators are not reproducible
Generators use `math/rand/v2`'s global, non-seedable source, so output cannot be pinned for
deterministic fixtures. Tracked as a planned `WithSeed`/`*rand.Rand` refactor in BACKLOG/ROADMAP.

### CLI reports unimplemented `--uf` as "invalid"
For a UF-scoped kind, `selo rg --uf AC` (or `ie --uf AC`) prints `invalid` and exits 1 rather than a
distinct "UF not implemented" message. The underlying API does return `ErrUFNotImplemented`; this is
a CLI UX nicety, not a correctness issue. Shared by RG and IE.

### `Generate` returns masked output
For consistency with the existing RG/IE generators, `Generate()` returns the canonical masked form
(e.g. `XX.XXX.XXX-C`), even though the `Document.Generate` doc comment says "unformatted".
`Validate` cleans input, so round-trips work either way.

## Toolchain / environment

### Go 1.25+ required
Adding the MCP `go-sdk` (v1.6.1) bumped `go.mod`'s go directive to 1.25.0. This is a consumer-visible
minimum even if you only use the core library.

### CRLF line endings in the Windows working tree
`core.autocrlf=true` with no `.gitattributes` means a local `gofmt -l` flags `.go` files on Windows,
though committed blobs are LF and Linux CI is unaffected. Tracked in BACKLOG (add `.gitattributes`).

### `golangci-lint` not always present in dev
The dev environment may lack `golangci-lint`; `go vet` is the local fallback. CI runs the full lint
(`task lint` / the reusable workflow). Install locally for parity.

## Deprecations
- `ValidateDocument(doc string) (string, bool)` is **deprecated** (use `Detect` + `Validate`);
  removal scheduled after 2026-07-18. See [BACKLOG.md](BACKLOG.md).
