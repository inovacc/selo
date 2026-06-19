# Bug Tracker

Confirmed defects in `github.com/inovacc/selo`. Limitations and by-design constraints live in
[ISSUES.md](ISSUES.md); future work in [BACKLOG.md](BACKLOG.md).

**Status as of 2026-06-19:** no open bugs. `go build/vet/test ./...` pass on all four packages
(total coverage 92.2%), and an independent whole-branch review of the latest changes (plans 001–006)
found zero Critical and zero Important issues.

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
- **CNPJ accepted all-equal inputs** (e.g. `00000000000000`). Fixed 2026-06 (commit `3797e6c`,
  advisor plan 002) by adding a shared `allEqualBytes` guard to `CNPJ.Validate`, matching CPF.
  Regression test present.
