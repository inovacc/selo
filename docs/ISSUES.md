# Known Issues & Limitations

Documented limitations and constraints of `github.com/inovacc/selo` as of 2026-06-19. These are
known, by-design, or tracked — not regressions. Bugs (if any) live in [BUGS.md](BUGS.md);
prioritized work in [BACKLOG.md](BACKLOG.md).

## Coverage / scope limitations

### RG supports only SP (RJ demoted — its algorithm differs)
`RG.Validate`/`ValidateUF` implement only the verified **SP** check-digit algorithm (mod 11,
weights 2..9, verified against four sources — `TestRG_AuthoritativeSamples`); every other UF returns
`(false, ErrUFNotImplemented)`. **RJ was removed (2026-06-19):** research found RJ uses a *different*
algorithm than SP (an SP-valid RG can be invalid under RJ rules), and no authoritative RJ algorithm
or verifiable samples were obtainable — so rather than validate RJ with the wrong (SP) algorithm,
`UFRJ` now returns `ErrUFNotImplemented`. RG check digits are not nationally standardized;
*workaround:* check `ImplementedUFs()` before relying on a UF. (Re-add RJ once its spec + ≥2 real
samples are sourced — see BACKLOG "Multi-state RG".)

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

### Default `Generate()` is non-seedable; deterministic generation requires the seeded API
The default `Generate()` / `GeneratePerson()` paths use `math/rand/v2`'s global, non-seedable
source. For reproducible fixtures use the seeded API (shipped v1.3.0): `GeneratePerson(WithSeed(n))`
or `WithRand(r)`, the registry `GenerateRand(kind, r)` helper, and the `RandGenerator` interface
(`GenerateRand(*rand.Rand)`) on every document type — the same seed yields identical output. The
same seed is available at the CLI (`selo person --seed N`) and MCP (`generate_person` `seed`)
surfaces (v1.4.0).

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
