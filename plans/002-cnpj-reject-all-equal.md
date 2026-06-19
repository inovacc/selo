# Plan 002: CNPJ rejects all-equal inputs (e.g. `00000000000000`), matching CPF

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat 15c0c91..HEAD -- brdoc.go brdoc_test.go cnpj_registry_test.go`
> If any of those changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch,
> treat it as a STOP condition.

## Status
- **Priority**: P1
- **Effort**: S
- **Risk**: LOW (behavior change — gated by new tests)
- **Depends on**: none (lands cleaner after 001 so CI validates it)
- **Category**: bug
- **Planned at**: commit `15c0c91`, 2026-06-18

## Why this matters
`CNPJ.Validate("00000000000000")` currently returns **`true`**. All-zero input has
all-zero check digits, which satisfy the mod-11 math, and — unlike CPF — `CNPJ.Validate`
has no guard against all-equal digit strings. A document of all identical digits is
never a real CNPJ; accepting it is a false positive that lets placeholder/sentinel
values pass validation. CPF already rejects this class (`isAccepted` /
`notAcceptedCPF`); this plan brings CNPJ to parity.

## Current state
File: `brdoc.go` (root package `selo`).

- **CPF's existing all-equal guard** (the pattern to mirror):
  ```go
  // brdoc.go:279-282
  func (c *CPF) isAccepted(value string) bool {
      // Reject CPFs with all equal digits
      return !slices.Contains(notAcceptedCPF, c.digits(value))
  }
  ```
  (`notAcceptedCPF` is a package var pre-populated in `init()` with the 10 all-equal CPF strings; `c.digits(value)` strips formatting.)

- **`CNPJ.Validate` — no such guard**:
  ```go
  // brdoc.go:314-350
  func (c *CNPJ) Validate(value string) bool {
      cleaned := c.digits(value)          // strips formatting; uppercases; keeps [0-9A-Z]
      if len(cleaned) != CnpjLength {     // CnpjLength == 14
          return false
      }
      ch12 := cleaned[12]
      if ch12 < '0' || ch12 > '9' { return false }
      dv1 := int(ch12 - '0')
      ch13 := cleaned[13]
      if ch13 < '0' || ch13 > '9' { return false }
      dv2 := int(ch13 - '0')
      base := cleaned[:12]
      dv1Calc, err := c.calculateDV(base)
      if err != nil { return false }
      dv2Calc, err := c.calculateDV(base + strconv.Itoa(dv1Calc))
      if err != nil { return false }
      return dv1Calc == dv1 && dv2Calc == dv2
  }
  ```
  Note CNPJ is **alphanumeric**: `cleaned` may contain `A–Z` in the first 12 positions (the check digits, positions 12–13, must be numeric). An "all-equal" guard must therefore compare the cleaned string's characters, not assume digits.

- **Convention**: validators return `bool`; helpers are lowercase methods on the type; tests are table-driven with testify. CNPJ tests live in `cnpj_registry_test.go` and the legacy `brdoc_test.go`.

## Commands you will need
| Purpose | Command | Expected on success |
|---|---|---|
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (focused) | `go test ./... -run CNPJ` | all pass |
| Test (full) | `go test ./...` | all packages `ok` |
| Format check | `gofmt -l brdoc.go cnpj_registry_test.go` | prints nothing |

## Scope
**In scope** (only files you may modify):
- `brdoc.go` (add the guard)
- `cnpj_registry_test.go` (add regression tests)

**Out of scope** (do NOT touch):
- CPF code — already guarded.
- `calculateDV` / the alphanumeric algorithm — the math is correct; only the missing guard is the bug.
- The `compat/` package — its tests already use a non-all-zeros invalid CNPJ sample; do not change them.

## Git workflow
- Branch: `advisor/002-cnpj-all-equal` off the current branch.
- Conventional commit, e.g. `fix(cnpj): reject all-equal inputs like CPF`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Add an all-equal guard to `CNPJ.Validate`
In `brdoc.go`, inside `CNPJ.Validate`, **immediately after** the length check
(`if len(cleaned) != CnpjLength { return false }`), add a rejection for strings whose
characters are all identical:
```go
	// Reject all-equal inputs (e.g. "00000000000000"); never a real CNPJ.
	if cnpjAllEqual(cleaned) {
		return false
	}
```
Then add the helper near the other CNPJ methods (anywhere in `brdoc.go` after the
`CNPJ` type declaration):
```go
// cnpjAllEqual reports whether every character of cleaned is identical.
func cnpjAllEqual(cleaned string) bool {
	for i := 1; i < len(cleaned); i++ {
		if cleaned[i] != cleaned[0] {
			return false
		}
	}
	return len(cleaned) > 0
}
```
Operate on `cleaned` (post-`digits`) so formatted all-zero input like `"00.000.000/0000-00"` is also rejected. Do not change anything else in the function.

**Verify**: `go build ./...` → exit 0.

### Step 2: Add regression tests
In `cnpj_registry_test.go`, add a test that pins the new behavior **and** confirms the
existing valid cases still pass (so the guard didn't over-reject):
```go
func TestCNPJ_RejectsAllEqual(t *testing.T) {
	c := NewCNPJ()
	// All-equal must be rejected (the bug this fixes).
	for _, bad := range []string{
		"00000000000000",
		"00.000.000/0000-00",
		"11111111111111",
	} {
		if c.Validate(bad) {
			t.Errorf("Validate(%q) = true, want false (all-equal must be rejected)", bad)
		}
	}
	// A real CNPJ still validates (guard must not over-reject).
	if !c.Validate("39591842000010") {
		t.Error("Validate(\"39591842000010\") = false, want true (known-valid regression sample)")
	}
}
```
(`39591842000010` is the project's established valid-CNPJ regression sample; its canonical mask is `39.591.842/0000-10`.)

**Verify**: `go test ./... -run CNPJ` → all pass, including `TestCNPJ_RejectsAllEqual`.

### Step 3: Full suite + format
**Verify**:
- `go test ./...` → all packages `ok` (nothing elsewhere depended on all-zeros being valid).
- `gofmt -l brdoc.go cnpj_registry_test.go` → prints nothing.

## Test plan
- New test `TestCNPJ_RejectsAllEqual` in `cnpj_registry_test.go`, modeled structurally
  on the existing table-style CNPJ tests in that file.
- Cases: all-zeros raw, all-zeros formatted, all-ones, plus one known-valid CNPJ to
  prove no over-rejection.
- Verification: `go test ./...` → all pass; the new test is included.

## Done criteria
ALL must hold:
- [ ] `go build ./...` exits 0.
- [ ] `go test ./...` exits 0; `TestCNPJ_RejectsAllEqual` exists and passes.
- [ ] `NewCNPJ().Validate("00000000000000")` is `false` (covered by the new test).
- [ ] `NewCNPJ().Validate("39591842000010")` is still `true` (covered by the new test).
- [ ] `gofmt -l brdoc.go cnpj_registry_test.go` prints nothing.
- [ ] Only `brdoc.go` and `cnpj_registry_test.go` modified (`git status`).
- [ ] `plans/README.md` status row updated.

## STOP conditions
Stop and report (do not improvise) if:
- `CNPJ.Validate` in the live `brdoc.go` no longer matches the "Current state" excerpt.
- The full suite (`go test ./...`) fails after the guard — some other test or the
  `compat` package may assert all-zeros is valid; report which test, do not weaken the guard.
- You find an existing `cnpjAllEqual`/all-equal helper already present (avoid a duplicate;
  reuse it and report).

## Maintenance notes
- For the reviewer: confirm the guard runs on `cleaned` (post-formatting strip), and that
  it returns `false` for empty input safely (the `len > 0` guard handles it; `Validate`
  already returns false on wrong length before reaching it).
- This is a deliberate behavior change. If any downstream consumer somehow relied on
  all-zeros validating (none in this repo), that's a breaking change to call out in release notes.
- Consider documenting in `docs/BACKLOG.md` that this item (listed under "Hardening / Tech
  Debt — CNPJ accepts all-zeros") is now resolved.
