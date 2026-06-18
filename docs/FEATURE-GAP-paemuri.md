# Feature-Gap & Enhancement Plan — `inovacc/brdoc` vs `paemuri/brdoc`

> Comparison target: [`github.com/paemuri/brdoc/v3`](https://github.com/paemuri/brdoc) (Unlicense, validation-only).
> Local project: `github.com/inovacc/brdoc` (Go 1.24, root package `brdoc`, CLI at `cmd/brdoc/`).
> Authored 2026-06-18. Actionable by another agent without further research.

---

## 1. Executive Summary

The local `inovacc/brdoc` project **substantially exceeds** `paemuri/brdoc` in *depth*: it ships generation, formatting/masking, public cleaning, a fully-featured Cobra CLI (single-value, bulk file/stdin streaming, count flags), benchmarks, ~95% test coverage, and CPF region inference via the 9th digit — none of which paemuri offers (paemuri is a flat, validation-only `Is*` function library). However, `paemuri` has far greater **breadth**: it validates 11 document classes (CPF, CNPJ, CEP, CNH, PIS/PASEP/NIS, RENAVAM, license plate, CNS, RG, Voter ID/Título Eleitoral, phone), while the local project covers only 2 (CPF, CNPJ). paemuri also returns geolocation (UF) for CEP/phone and distinguishes Mercosul vs national plates. The strategic play: **keep the local project's superior per-document depth and close the breadth gap** by porting paemuri's document types into the local architecture (struct + `Generate`/`Validate`/`Format`/`CheckOrigin` + CLI subcommand + tests + benchmarks), turning a validation-only competitor into a full generate-validate-format toolkit across all 11+ Brazilian document types. Net result: a strict superset of paemuri with no peer in the Go ecosystem.

---

## 2. Side-by-Side Capability Matrix

Rows = union of document types. Columns: paemuri (validate only) vs local (validate / generate / format / CLI).

| Document Type | paemuri: Validate | local: Validate | local: Generate | local: Format | local: CLI |
|---|:---:|:---:|:---:|:---:|:---:|
| **CPF** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **CNPJ** (incl. alphanumeric) | ✅ | ✅ | ✅ | ✅ | ✅ |
| **CEP** (postal code + UF) | ✅ | ❌ | ❌ | ❌ | ❌ |
| **CNH** (driver's license) | ✅ | ❌ | ❌ | ❌ | ❌ |
| **PIS / PASEP / NIS / NIT** | ✅ | ❌ | ❌ | ❌ | ❌ |
| **RENAVAM** (vehicle reg.) | ✅ | ❌ | ❌ | ❌ | ❌ |
| **License plate** (national + Mercosul) | ✅ | ❌ | ❌ | ❌ | ❌ |
| **CNS** (health card) | ✅ | ❌ | ❌ | ❌ | ❌ |
| **RG** (SP/RJ only) | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Título Eleitoral** (voter ID) | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Phone** (BR telephone + UF) | ✅ | ❌ | ❌ | ❌ | ❌ |
| Region/UF inference | ✅ (CEP, phone) | ✅ (CPF region digit) | — | — | — |
| Bulk file/stdin validation | ❌ | ✅ | — | — | ✅ |
| Fake/test data generation | ❌ | ✅ | ✅ | — | ✅ |
| Public clean/strip helpers | ❌ (unexported) | ✅ | — | — | — |

**Reading:** local wins every column it competes in (depth); paemuri wins 9 of 11 rows on presence (breadth).

---

## 3. Gap Analysis — Features to IMPLEMENT (present in paemuri, missing locally)

### Architecture pattern to mirror

The local project is a **flat root package** (`package brdoc`, files `brdoc.go`, `doc.go`, `brdoc_test.go`) with the struct-method idiom:

```go
type CPF struct{ ... }
func NewCPF() *CPF
func (c *CPF) Generate() string
func (c *CPF) Validate(value string) bool
func (c *CPF) Format(value string) (string, error)
func (c *CPF) CheckOrigin(value string) string   // optional, geolocation
```

…plus a top-level dispatcher `ValidateDocument(doc string) (docType string, isValid bool)` and one Cobra subcommand per type in `cmd/brdoc/main.go` using `-g/--generate`, `-v/--validate`, `-f/--from`, `-n/--count` flags.

**Recommendation:** as the type count grows past ~4, split each document into its own file in the root package (still `package brdoc`) to keep `brdoc.go` from ballooning. Suggested files: `cnh.go`, `pis.go`, `renavam.go`, `voterid.go`, `cep.go`, `plate.go`, `cns.go`, `rg.go`, `phone.go`, each with a sibling `*_test.go`. This preserves the existing import path (`brdoc.NewCNH()`) — no new sub-packages, no breaking change. Each new type implements the same 4–5 method contract. CLI adds one `cobra.Command` per type, registered via `rootCmd.AddCommand(...)` in `init()`.

---

### 3.1 CNH — Carteira Nacional de Habilitação (driver's license)

- **What it is:** 11-digit national driver's-license number with two check digits.
- **Format/length:** exactly 11 digits, no mask/punctuation in canonical form.
- **Validation algorithm (two check digits, mod-11 with -2 base offset):**
  1. Reject if length ≠ 11 or any non-digit. Reject all-equal (e.g. `11111111111`).
  2. **DV1:** sum `dᵢ × wᵢ` for i=0..8 with weights `w = 9,8,7,6,5,4,3,2,1`. Let `r = sum % 11`. `dv1 = r >= 10 ? 0 : r`. Track an offset flag: if `r >= 10`, set `dsc = 2`, else `dsc = 0`.
  3. **DV2:** sum `dᵢ × wᵢ` for i=0..8 with weights `w = 1,2,3,4,5,6,7,8,9`. Let `r = (sum % 11) - dsc`. If `r < 0`, `r += 11`. `dv2 = r >= 10 ? 0 : r`.
  4. Valid iff `dv1 == d[9]` and `dv2 == d[10]`.
- **Generate:** emit 9 random digits, compute DV1/DV2 by the above, append. Reject the all-equal accident and retry.
- **Format:** no official mask; `Format` can return the cleaned 11-digit string or `(value, nil)`.
- **Slots into:** `cnh.go` (`type CNH struct{}`, `NewCNH`, `Generate`, `Validate`, `Format`), `cnh_test.go`, CLI subcommand `brdoc cnh -g|-v|-f`, benchmark `BenchmarkCNHValidate`/`BenchmarkCNHGenerate`.

### 3.2 PIS / PASEP / NIS / NIT

- **What it is:** 11-digit social-security/worker-registration number (PIS/PASEP/NIS/NIT share the algorithm).
- **Format/length:** 11 digits; common mask `###.#####.##-#`.
- **Validation algorithm (single check digit, mod-11, fixed weights):**
  1. Strip non-digits; require length 11. (Optionally reject all-equal.)
  2. Weights `w = 3,2,9,8,7,6,5,4,3,2` over the first 10 digits.
  3. `sum = Σ dᵢ·wᵢ`; `mod = sum % 11`; `dv = mod <= 1 ? 0 : 11 - mod`.
  4. Valid iff `dv == d[10]`.
- **Generate:** 10 random digits → compute `dv` → append. **Format:** apply `###.#####.##-#` mask (or return error on bad length).
- **Slots into:** `pis.go` / `pis_test.go`, CLI `brdoc pis`, benchmarks.

### 3.3 RENAVAM — vehicle registration

- **What it is:** 11-digit national vehicle registration number, one check digit.
- **Format/length:** 11 digits (modern; older 9-digit forms left-pad with zeros to 11).
- **Validation algorithm (single check digit, `(sum*10) % 11`):**
  1. Require length 11, all digits. Optionally reject all-equal.
  2. Weights `w = 3,2,9,8,7,6,5,4,3,2` over the first 10 digits (right-aligned).
  3. `sum = Σ dᵢ·wᵢ`; `dv = (sum * 10) % 11`; if `dv == 10`, `dv = 0`.
  4. Valid iff `dv == d[10]`.
- **Generate:** 10 random digits → compute `dv` → append. **Format:** zero-pad to 11; no standard separators.
- **Slots into:** `renavam.go` / `renavam_test.go`, CLI `brdoc renavam`, benchmarks.

### 3.4 Título Eleitoral — Voter ID

- **What it is:** 12-digit voter registration with an embedded 2-digit UF code and two check digits.
- **Format/length:** 12 digits; logical layout `SSSSSSSS UU D1 D2` (8 sequence + 2 UF + 2 DV).
- **Validation algorithm (UF range + two mod-11 check digits):**
  1. Require length 12, all digits. Optionally reject all-equal.
  2. **UF check:** digits at positions 8–9 (the `UU` pair) must be in `01..28` (01–27 = states+DF, 28 = exterior). Reject otherwise.
  3. **DV1** over the first 8 digits with weights `2,3,4,5,6,7,8,9`: `mod = sum % 11`; `dv1 = (mod == 10 || mod == 11) ? 0 : mod`.
  4. **DV2** over the two UF digits **plus dv1** (3 values) with weights `7,8,9`: `mod = sum % 11`; `dv2 = (mod == 10 || mod == 11) ? 0 : mod`.
  5. Valid iff `dv1 == d[10]` and `dv2 == d[11]`.
- **Generate:** 8 random sequence digits + random UF in `01..28` → compute DV1, DV2 → concat. **CheckOrigin:** map UF code → state name (bonus geolocation, mirrors `CPF.CheckOrigin`). **Format:** no canonical mask; spaced groups optional.
- **Slots into:** `voterid.go` / `voterid_test.go`, CLI `brdoc titulo` (alias `voter`), benchmarks. Add UF-code→state map constant block.

### 3.5 CEP — postal code (format-only + UF inference)

- **What it is:** 8-digit postal code, no check digit; first 3 digits map to a UF via official numeric ranges.
- **Format/length:** 8 digits, mask `#####-###`.
- **Validation:** regex `^\d{5}-?\d{3}$`; valid iff the first-3-digit prefix falls in a known UF range. **CheckOrigin** returns the UF.
- **Generate:** pick a UF range, emit a random 8-digit code inside it (gives plausible fakes). **Format:** apply `#####-###`.
- **Slots into:** `cep.go` / `cep_test.go`, CLI `brdoc cep`. Requires a UF-range table (port from paemuri or BCB/Correios ranges).

### 3.6 License Plate — national + Mercosul (format-only)

- **What it is:** vehicle plate; legacy `ABC-1234` and Mercosul `ABC1D23`.
- **Validation:** `IsNationalPlate` = `^[A-Z]{3}-?\d{4}$`; `IsMercosulPlate` = `^[A-Z]{3}\d[A-Z]\d{2}$`; `IsPlate` = either. No check digit.
- **Generate:** random letters/digits per pattern (national or Mercosul variant flag, mirroring `CNPJ.GenerateLegacy`). **Format:** insert/strip the national dash.
- **Slots into:** `plate.go` / `plate_test.go`, CLI `brdoc plate [--mercosul]`.

### 3.7 CNS — Cartão Nacional de Saúde (health card)

- **What it is:** 15-digit national health-card number; definitive (prefix 1/2) vs provisional (prefix 7/8/9).
- **Validation (mod-11 divisibility):** strip non-digits, length 15; weighted sum with weights `15,14,…,1` by position must satisfy `sum % 11 == 0`. Regex gates the prefix class.
- **Generate:** construct a 15-digit number satisfying `sum % 11 == 0` (definitive form: seed first 11 digits, compute the 4-digit remainder block per the algorithm; retry on overflow).
- **Slots into:** `cns.go` / `cns_test.go`, CLI `brdoc cns`.

### 3.8 RG — Registro Geral (SP & RJ only)

- **What it is:** state ID card; only SP and RJ algorithms are well-defined.
- **Validation (SP):** format `\d{2}.?\d{3}.?\d{3}-?[0-9xX]`; strip to digits/X; mod-11 with weights `2..9` by position; check char `X` = 10, `0` = 11. Returns `(valid bool, err error)` — returns an explicit "federative unit not implemented" error for other UFs.
- **Slots into:** `rg.go` / `rg_test.go`. **Note:** RG needs a UF argument and an error return — diverges from the plain `Validate(value) bool` signature. Add `func (r *RG) Validate(value string, uf UF) (bool, error)` and a sentinel `ErrUFNotImplemented`. CLI `brdoc rg --uf SP`.

### 3.9 Phone — Brazilian telephone (format + UF inference)

- **What it is:** optional `+55`/`0055` prefix, 2-digit DDD area code, 8- or 9-digit subscriber number.
- **Validation:** regex for the full shape; DDD must map to a known UF, else invalid. **CheckOrigin** returns the UF from the DDD.
- **Generate:** pick a valid DDD → random 9-digit mobile (or 8-digit landline). **Format:** `(##) #####-####`.
- **Slots into:** `phone.go` / `phone_test.go`, CLI `brdoc phone`. Requires a DDD→UF table.

---

## 4. Enhancements (strengthen local beyond paemuri)

Only value-adding items listed.

1. **Unified `Validate(kind, value)` dispatcher.** Extend the existing `ValidateDocument` into `func Validate(kind DocKind, value string) (bool, error)` with a `DocKind` enum (CPF, CNPJ, CNH, PIS, RENAVAM, VoterID, CEP, Plate, CNS, RG, Phone) and an auto-detect mode (`ValidateDocument` already auto-detects CPF vs CNPJ — generalize it by length/shape). Single entry point for callers.
2. **Error sentinel values + `errors.Is`.** Define `var ErrInvalidLength`, `ErrInvalidFormat`, `ErrUFNotImplemented`, `ErrUnknownDocType` and wrap with `%w`. `Format` already returns errors; make them comparable. (Per Go standards: always `errors.Is`/`errors.As`.)
3. **Region/UF inference everywhere applicable.** Add `CheckOrigin` to Voter ID (UF code), CEP (range), Phone (DDD), mirroring `CPF.CheckOrigin`. Unify into a small `UF` type with the 27 constants (steal paemuri's exported `UF` set — Unlicense permits verbatim reuse).
4. **Fake-data generation for every new type.** paemuri has *zero* generation; making `Generate()` available for CNH/PIS/RENAVAM/VoterID/CEP/Plate/CNS/Phone is the single biggest differentiator for test-data tooling. CLI `--count N` already exists; reuse it per subcommand.
5. **Masking/formatting for every new type** with the documented masks (PIS `###.#####.##-#`, CEP `#####-###`, phone `(##) #####-####`, plate dash). paemuri offers none.
6. **Batch validation parity.** The CPF/CNPJ `--from FILE|-` streaming bulk path (bufio.Scanner, 1 MB buffer) should be factored into a shared helper and reused by every new subcommand — instant bulk validation for all 11 types.
7. **Fuzz tests** (`func FuzzCPFValidate(f *testing.F)`, etc.) for each check-digit validator: round-trip `Generate→Validate` must always be true; random strings must never panic. Go 1.24 native fuzzing.
8. **Exhaustive table-driven tests** with real-world valid samples + boundary cases (all-equal, wrong length, bad UF code, off-by-one check digit) per type; keep ≥80% coverage (project currently ~95%).
9. **Godoc** examples (`ExampleCNH_Validate`, etc.) so the package renders runnable examples on pkg.go.dev — paemuri has thin docs; this is a discoverability edge.
10. **PIX key validation** (BR financial context, BCB spec): validate the 5 key kinds — CPF, CNPJ, email, phone (E.164 `+55...`), and EVP (UUIDv4 random key). Reuse the CPF/CNPJ/phone validators; add `pix.go`. Genuinely net-new vs paemuri and high-value for fintech consumers.
11. **`golangci-lint` + Taskfile `test:full`** target gating the new code (project already has `Taskfile.yml`, `.golangci.yml`); add `-short` skips only if any generator becomes slow (none should).

---

## 5. Prioritized Roadmap

Ordered by value ÷ effort (quick high-value wins first). Each row is a self-contained unit of work.

| # | Item | Type | Value | Effort | Notes |
|---|---|:---:|:---:|:---:|---|
| 1 | **PIS/PASEP/NIS** struct (`pis.go`) + Validate/Generate/Format + CLI `pis` + table tests + bench | implement | H | S | Single mod-11 digit, fixed weights `3,2,9,8,7,6,5,4,3,2`; mask `###.#####.##-#`. Cleanest port. |
| 2 | **RENAVAM** struct (`renavam.go`) + full method set + CLI + tests + bench | implement | H | S | Single digit `(sum*10)%11`, same weights as PIS. 11 digits, no mask. |
| 3 | **CNH** struct (`cnh.go`) + full method set + CLI + tests + bench | implement | H | M | Two check digits w/ `-2` base offset (see §3.1). Reject all-equal. |
| 4 | **Título Eleitoral** struct (`voterid.go`) + Validate/Generate/CheckOrigin(UF) + CLI + tests | implement | H | M | UF code `01..28` + two mod-11 DVs (§3.4). Adds geolocation parity. |
| 5 | **Error sentinels + `errors.Is`** across all Format/Validate paths (`errors.go`) | enhance | H | S | `ErrInvalidLength/Format/UnknownDocType/UFNotImplemented`; wrap with `%w`. Do before/with #1–4. |
| 6 | **Shared bulk `--from` helper** extracted from CPF/CNPJ; reuse in all subcommands | enhance | H | S | Factor existing bufio.Scanner path into `internalBulk(cmd, validator)`; wire into every new CLI cmd. |
| 7 | **License plate** (`plate.go`) national+Mercosul + Generate/Format + CLI `plate [--mercosul]` | implement | M | S | Regex-only validation; generation is pure pattern fill. No check digit. |
| 8 | **CEP** (`cep.go`) Validate+CheckOrigin(UF)+Generate+Format + CLI `cep` | implement | M | M | Needs UF numeric-range table (port from paemuri, Unlicense). Mask `#####-###`. |
| 9 | **Phone** (`phone.go`) Validate+CheckOrigin(UF)+Generate+Format + CLI `phone` | implement | M | M | Needs DDD→UF table. Mask `(##) #####-####`; 8/9-digit subscriber. |
| 10 | **CNS** (`cns.go`) Validate+Generate + CLI `cns` | implement | M | M | mod-11 divisibility (`sum%11==0`), weights `15..1`; generation needs constructive solve. |
| 11 | **Unified `Validate(kind, value)` + `DocKind` enum** generalizing `ValidateDocument` | enhance | M | S | One dispatch entry point + auto-detect by length/shape. Do after ≥4 types exist. |
| 12 | **RG (SP/RJ)** (`rg.go`) with `Validate(value, uf)` + `ErrUFNotImplemented` | implement | M | M | Divergent signature (UF arg + error). SP weights `2..9`, check `X=10/0=11`. |
| 13 | **Fuzz tests** for every check-digit validator (round-trip + no-panic) | enhance | M | S | Go 1.24 native fuzzing; `Generate→Validate` invariant. One per type. |
| 14 | **PIX key validation** (`pix.go`): CPF/CNPJ/email/E.164 phone/EVP UUIDv4 | enhance | M | M | BCB spec; reuse existing validators. Net-new vs paemuri; fintech value. |
| 15 | **Godoc `Example*` functions** for each type | enhance | L | S | Renders runnable examples on pkg.go.dev. |
| 16 | **`UF` type + 27 constants** consolidated (`uf.go`) | enhance | L | S | Underpins #4/#8/#9/#12 CheckOrigin; port verbatim from paemuri (Unlicense). |
| 17 | **Inscrição Estadual** (`ie.go`) per-UF Validate + CLI `ie --uf XX` | implement | H | L | ⭐ **Differentiator** — paemuri issue #7 open since inception, never shipped by anyone. 27 distinct per-UF algorithms (length + check digits vary). Land incrementally (SP, RJ, MG, RS, PR first). Diverging signature `Validate(value string, uf UF) (bool, error)`. |
| 18 | **RG multi-state** extend `rg.go` beyond SP/RJ | implement | M | L | ⭐ **Differentiator** — paemuri issue #22 only ships SP/RJ. Add more UF algorithms where check-digit rules are documented; explicit `ErrUFNotImplemented` elsewhere. |
| 19 | **CNPJ edge-case hardening test** — add `39591842000010`-class samples | enhance | M | S | paemuri bug #26/#27 (valid CNPJ falsely rejected). Add the case to `cnpj` table tests to guard against the same regression. |

---

## 6. Concrete Next 3 Steps

All in the root `package brdoc`, mirroring `brdoc.go`'s struct-method idiom and `cmd/brdoc/main.go`'s subcommand pattern. Follow the user's Go standards: table-driven tests, ≥80% coverage, `go run` (never build-then-run), Cobra, `errors.Is`, structured errors.

### Step 1 — Implement PIS (`pis.go`) + tests (Roadmap #1)

Create `D:\weaver-sync\development\personal\projects\brdoc\pis.go`:

```go
package brdoc

import "errors"

const PisLength = 11

var pisWeights = [10]int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

type PIS struct{}

func NewPIS() *PIS { return &PIS{} }

func (p *PIS) Validate(value string) bool {
	d := onlyDigits(value) // shared helper; or local strip mirroring CPF.digits
	if len(d) != PisLength {
		return false
	}
	sum := 0
	for i := 0; i < 10; i++ {
		sum += int(d[i]-'0') * pisWeights[i]
	}
	mod := sum % 11
	dv := 0
	if mod > 1 {
		dv = 11 - mod
	}
	return dv == int(d[10]-'0')
}

func (p *PIS) Generate() string {
	// emit 10 random digits via the package rand.Rand, compute dv, append.
}

func (p *PIS) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != PisLength {
		return "", ErrInvalidLength // defined in errors.go (Step adjacent)
	}
	// mask ###.#####.##-#
	return d[0:3] + "." + d[3:8] + "." + d[8:10] + "-" + d[10:11], nil
}
```

Create `D:\weaver-sync\development\personal\projects\brdoc\pis_test.go` — table-driven (valid samples, wrong length, off-by-one DV, non-digit), plus `func BenchmarkPISValidate` / `BenchmarkPISGenerate` and a `Generate→Validate` round-trip subtest. Use `testify` (already a dep). Run with `go test -run TestPIS ./...` and `go test -bench=PIS -benchmem ./...`.

### Step 2 — Wire the CLI subcommand in `cmd/brdoc/main.go`

Add a `pisCmd` mirroring `cpfCmd`/`cnpjCmd` (lines 90 / 204 in `main.go`), with `-g/--generate`, `-v/--validate`, `-f/--from`, `-n/--count` flags, registered via `rootCmd.AddCommand(pisCmd)` in `init()`. Reuse the existing bulk `--from` streaming logic (extract it to a shared helper per Roadmap #6 to avoid copy-paste). Verify with:

```
go run ./cmd/brdoc pis --generate --count 5
go run ./cmd/brdoc pis --validate 120.1234.567-8
```

### Step 3 — Add error sentinels (`errors.go`) (Roadmap #5)

Create `D:\weaver-sync\development\personal\projects\brdoc\errors.go`:

```go
package brdoc

import "errors"

var (
	ErrInvalidLength      = errors.New("brdoc: invalid document length")
	ErrInvalidFormat      = errors.New("brdoc: invalid document format")
	ErrUnknownDocType     = errors.New("brdoc: unknown document type")
	ErrUFNotImplemented   = errors.New("brdoc: federative unit not implemented")
)
```

Retrofit `CPF.Format` and `CNPJ.Format` to return these (wrapped with `%w` where context is added) so callers can use `errors.Is`. Add tests asserting `errors.Is(err, brdoc.ErrInvalidLength)`. This is the shared foundation every subsequent type (CNH, RENAVAM, Voter ID, …) depends on, so land it alongside Step 1.

> After Steps 1–3, repeat the same three-file recipe (`<type>.go`, `<type>_test.go`, CLI subcommand) for RENAVAM → CNH → Título Eleitoral, in roadmap order. A shared `onlyDigits(string) string` helper and the `UF` type (`uf.go`) should be added once and reused.

---

## 7. Upstream Issues — Validation & Differentiators

**Repo status (as of research):** `paemuri/brdoc` is **active, not archived** — 146 stars, default branch `main`, module at major **v3**, last push 2025-06-21. **3 open issues, 0 open PRs.** This means most of paemuri's roadmap has already merged (CNH, CEP, RENAVAM, plate, CNS, PIS/PASEP, Voter ID, phone, alphanumeric CNPJ) — so §3 above is a port of *shipped* features, while the table below is where upstream is *still stuck*. Those stuck items are the local project's clearest greenfield differentiators.

| # | Title | State | Class | What it asks | Local action |
|---|---|---|---|---|---|
| 7 | Inscrição Estadual validator | **OPEN** | new-doc-type | Per-UF state tax registration; 27 distinct algorithms | ⭐ Roadmap #17 — **nobody has shipped this**; biggest open gap in the ecosystem |
| 22 | Validador de RG | **OPEN** | new-doc-type | Multi-state RG; rules vary per UF (upstream only SP/RJ) | ⭐ Roadmap #18 — extend beyond SP/RJ for a real edge |
| 21 | Suporte a celular/telefone | **OPEN (stale)** | new-doc-type | Phone validation — *already merged* via PR #28; issue not closed | Skip — covered by Roadmap #9 (phone) |
| 6 | Generic `IsDocument` fn | closed (rejected) | api-design | One dispatch fn over all doc types | ✅ **Already shipped locally** as `ValidateDocument()`; generalize via Roadmap #11 (`Validate(kind, value)`) |
| 16 | Remove multi-UF check | closed (rejected) | api-design | Tension over single vs. multiple UF acceptance | Decide deliberately in the `UF`/`CheckOrigin` design (Roadmap #16); expose both `*From(uf...)` and plain variants |
| 5 / 13 | CPF w/o special chars; CNH w/ hyphen | closed | enhancement | Accept loosely-formatted input | Local already strips via `onlyDigits`; keep tolerant parsing the default across all new types |
| 26 / 27 | `IsCNPJ` false-negative on `39591842000010` | closed (bug) | bug | Valid CNPJ wrongly rejected | Roadmap #19 — add the case to `cnpj` table tests as a regression guard |

**Strategic takeaways**

1. **Inscrição Estadual (#7) and multi-state RG (#22)** are the two items *even the upstream leader has not solved* — shipping them (incrementally, top UFs first) makes `inovacc/brdoc` strictly ahead of `paemuri/brdoc`, not just at parity.
2. **The generic dispatcher paemuri rejected (#6) already exists locally** — lean into it as a marketing/API point; generalize it to a typed `Validate(kind, value)` (Roadmap #11) rather than re-litigating the upstream debate.
3. **Borrow upstream's hard-won bug fixes** — the `39591842000010` CNPJ false-negative (#26/#27) and the alphanumeric-CNPJ rule (#29/#30/#31, already in local) are battle-tested edge cases; pin them into the test suite so the local project never regresses where paemuri did.
4. **Licensing is friction-free:** paemuri is **Unlicense (public domain)** — UF tables, CEP/DDD ranges, and check-digit constants may be copied verbatim. (Confirm the local project's own license header policy before vendoring, per the user's BSD-3-Clause default.)
