# Bug Tracker

Confirmed defects in `github.com/inovacc/selo`. Limitations and by-design constraints live in
[ISSUES.md](ISSUES.md); future work in [BACKLOG.md](BACKLOG.md).

**Status as of 2026-06-20:** no open bugs. `go build/vet/test ./...` pass on all five packages
(total coverage 94.2%), and the codegen CI matrix is green across all six target languages
(TypeScript, JavaScript, Ruby, Java, C#, Python).

## Open
_None._

## Severity scale
- **Critical** — data corruption, crash, or a validator that accepts invalid / rejects valid real documents.
- **High** — incorrect result in a common path; no safe workaround.
- **Medium** — incorrect result in an edge case, or a workaround exists.
- **Low** — cosmetic or minor inconvenience.

## Template (copy when filing)
```
### BUG-NNN: <short title>
- **Severity:** Critical | High | Medium | Low
- **Status:** Open | In progress | Fixed (commit) | Won't fix
- **Area:** <package / kind / command>
- **Found:** YYYY-MM-DD
- **Repro:** <minimal steps or failing test>
- **Expected:** <...>
- **Actual:** <...>
- **Notes / fix:** <...>
```

## Fixed
- **Python golden test broke after `task gen:verify:python`.** `TestGoldenPython_NoExtraDeterministicFiles`
  walked `generated/python` without skipping Python toolchain dirs, so `pip install -e .` / pytest
  artifacts (`*.egg-info/`, `__pycache__/`, `.pytest_cache/`) were flagged as "extra" files and failed
  the test. Fixed 2026-06-20 by skipping those dirs, mirroring how the TS/JS gates skip `node_modules`.
  (CI never hit it — its `go test` runs on a clean checkout — but a local `gen:verify:python` then
  `go test` would.)
- **CNPJ accepted all-equal inputs** (e.g. `00000000000000`). Fixed 2026-06 (commit `3797e6c`,
  advisor plan 002) by adding a shared `allEqualBytes` guard to `CNPJ.Validate`, matching CPF.
  Regression test present.
