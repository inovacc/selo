# Inscrição Estadual (IE) — research log & roadmap

> Status as of 2026-06-20 (v1.6.0). **Shipped: SP, MG, RS, PR (4 UFs).** RJ
> researched again and **kept blocked** (see below). First batch (SP) shipped in
> plan 006; MG/RS/PR added in v1.6.0 after a fresh, adversarially-verified research
> spike (the per-UF algorithms, samples, and sources are folded into this doc).
> IE has no national standard — each of the 27 federative units defines its own
> length(s), weights, mod, and check-digit rule(s); several accept multiple formats.
> This is the single biggest gap in the Brazilian-document Go ecosystem
> (`paemuri/brdoc` issue #7, never shipped). The discipline here mirrors plan 004
> (RG): **a UF is implemented only with an authoritative algorithm AND ≥2 sourced,
> independently-published samples whose check digits verify by hand.** No invented
> samples — an internally-consistent validator can accept fake numbers while
> rejecting real ones.

## Architecture

`ie.go` mirrors `rg.go` (the `Document` + `UFScoped` exemplar):

- `type IE struct{}`, self-registers via `init() { Register(&IE{}) }`, `Kind() == KindIE`
  (`KindIE = "ie"`, added to `document.go`).
- `ieTable map[UF]ieAlgo` holds only **verified** UFs. `ieAlgo` carries:
  - `lengths []int` — accepted cleaned-digit lengths (allows a UF to grow extra formats),
  - `validate func(d string) bool`,
  - `generate func() string` (masked; `nil` when constructive generation isn't implemented),
  - `mask func(d string) string` (identity when no mask defined).
- `ValidateUF`: unimplemented UF → `ErrUFNotImplemented` (wrapped); accepted-length mismatch
  for a supported UF → `ErrInvalidFormat`; correct shape, bad check → `(false, nil)`.
- `Validate`: tries each implemented UF, first match wins.
- `Format`: applies the mask of the first implemented UF under which the value validates;
  otherwise `ErrInvalidFormat` (IE masks are UF-specific, so the UF is inferred from which
  algorithm accepts the value).
- `Generate`: random implemented UF that supports generation → a valid masked IE.

**CLI/MCP**: derive from the registry's `Kinds()`, and the `--uf` flag is already wired for
`UFScoped` kinds — so once IE registered, `selo ie --uf SP --validate <n>` works with **no
CLI/MCP edits**. Verified: `selo ie --uf SP --validate 110.042.490.114` → `valid` (exit 0).
UX note (shared with RG): an unimplemented `--uf` now surfaces a distinct "UF not implemented"
message (fixed in v1.2.0) — `ValidateUF` returns `ErrUFNotImplemented` and the CLI reports it.

## Per-UF status

| UF | Status | Notes |
|----|--------|-------|
| SP | **ready** ✅ | 12 digits; two check digits (pos 9 and 12), mod-11 "rightmost digit" rule. Implemented + tested. |
| MG | **ready** ✅ | 13 digits; D1 digit-sum method (zero inserted after municipio, alternating 1,2 weights), D2 mod-11. Official SINTEGRA-MG worked example anchors it. |
| RS | **ready** ✅ | 10 digits; single mod-11 DV, weights `[2,9,8,7,6,5,4,3,2]`. Official SINTEGRA-RS worked example anchors it. |
| PR | **ready** ✅ | 10 digits; two mod-11 DVs (DV2 folds in `2·DV1`). Official SEFA-PR reference routine + worked example anchor it. |
| RJ | **blocked** ⛔ | 8 digits, single DV, `DV = 11-(sum mod 11)`, 10/11→0. The official SINTEGRA-RJ page publishes the rule and format but **NOT the weight vector**. The vector `[2,7,6,5,4,3,2]` and every sample come only from community impls; validating those samples with that same vector is circular, so it fails the authoritative-algorithm bar. Re-add once an official-anchored algorithm or sample appears. |

**SP, MG, RS, and PR** meet the "authoritative algorithm + ≥2 independently-verified
samples" bar — each anchored by an OFFICIAL worked example (SINTEGRA-MG/RS, SEFA-PR) that is
itself a verified sample, plus independent reference-impl corroboration. The fresh research
(MG/RS/PR/RJ) was adversarially verified: every cited check digit was re-derived from the
stated algorithm and matched, and published negative controls reject. **RJ alone stays
blocked** — shipping it would mean trusting a weight vector no authoritative source publishes.

## SP — verified algorithm

Format `AAA.AAA.AAA.AAA` (12 digits). Positions 9 and 12 are check digits; each is the
**rightmost (units) digit of (weighted sum mod 11)** — so a remainder of 10 yields 0.

- **D1 (position 9)**: weights `[1,3,4,5,6,7,8,10]` applied to digits 1..8.
- **D2 (position 12)**: weights `[3,2,10,9,8,7,6,5,4,3,2]` applied to digits 1..11.

### Sources
- SEFAZ-SP / Sintegra — "Rotina de Consistência da Inscrição Estadual Paulista":
  <https://portal.fazenda.sp.gov.br/servicos/icms/Paginas/sintegra-rotina-consistencia.aspx>
- Sintegra cadastro SP: <http://www.sintegra.gov.br/Cad_Estados/cad_SP.html>
- Cross-check reference implementation (27 UF rules):
  <https://marcoluglio.github.io/br/inscricaoestadualcpfcnpj/>

### Verified samples (pinned in `TestIE_AuthoritativeSamples`)
Both independently published, both check digits verified by hand:

- **110.042.490.114** — SEFAZ-SP/Sintegra worked example.
  - D1: `1·1+1·3+0·4+0·5+4·6+2·7+4·8+9·10 = 164`; `164 mod 11 = 10` → **0** (9th digit). ✓
  - D2: `1·3+1·2+0·10+0·9+4·8+2·7+4·6+9·5+0·4+1·3+1·2 = 125`; `125 mod 11 = 4` → **4** (12th). ✓
- **388.108.598.269** — published valid SP IE example.
  - D1: sum `250`; `250 mod 11 = 8` → **8** (9th digit). ✓
  - D2: sum `295`; `295 mod 11 = 9` → **9** (12th digit). ✓

> Provenance caveat: these are official-documentation / published worked examples, not
> confirmed real-company registrations (real IE numbers aren't publicly verifiable for
> privacy reasons). The SEFAZ-SP example is as authoritative as the source gets.

## MG, RS, PR — verified algorithms (v1.6.0)

Added after a fresh research spike whose every sample was re-derived and adversarially
verified. Each is anchored by an OFFICIAL worked example (which doubles as a pinned sample)
plus independent reference-impl corroboration (Thiagocfn/InscricaoEstadual PHP,
Printi/gammasoft JS). Samples are pinned in `TestIE_AuthoritativeSamples` /
`TestIE_ValidateUF` (with published negative controls).

### MG — 13 digits, format `AAA.AAA.AAA/AAAA`
3 municipio + 6 inscrição + 2 ordem + 2 DV.
- **D1 (digit-sum method)**: insert a `0` right after the 3-digit municipio code (→ 12
  digits), multiply left→right by alternating weights `1,2,1,2,…`, sum the **digits** of
  each product; `D1 = next-multiple-of-ten(total) − total`.
- **D2 (mod-11)**: over the 11 base digits + D1 (12 digits), weights left→right
  `[3,2,11,10,9,8,7,6,5,4,3,2]`; `D2 = 11 − (sum mod 11)`, remainder 0/1 → 0.
- Sources: SINTEGRA-MG <http://www.sintegra.gov.br/Cad_Estados/cad_MG.html>; Thiagocfn; Printi.
- Samples: `0623079040081` (official, D1=8 D2=1), `4333908330177`, `7023259570005`.

### RS — 10 digits, format `AAA/AAAAAAA`
3 municipio + 6 empresa + 1 DV. Single mod-11 DV: weights left→right `[2,9,8,7,6,5,4,3,2]`
over the first 9 digits; `DV = 11 − (sum mod 11)`, result 10/11 → 0.
- Sources: SINTEGRA-RS <http://www.sintegra.gov.br/Cad_Estados/cad_RS.html>; Thiagocfn; Printi.
- Samples: `2243658792` (official, DV=2), `0305169149`, `0963205056`.

### PR — 10 digits, format `AAA.AAAAA-AA`
8 base + 2 DV. Both mod-11: DV1 weights `[3,2,7,6,5,4,3,2]`; DV2 weights `[4,3,2,7,6,5,4,3]`
**plus `2·DV1` added to the sum**; each `DV = 11 − (sum mod 11)`, result 10/11 → 0.
- Sources: SEFA-PR <https://www.fazenda.pr.gov.br/Pagina/calculo-digito-verificador> (algorithm
  + worked example); Thiagocfn.
- Samples: `1234567850` (official, DV1=5 DV2=0), `4447953604`.

### Multi-language codegen parity gap
The Go library validates SP/MG/RS/PR IE, but the multi-language code generator
(`internal/codegen`) still emits **SP-only** IE (`spec.go` KindIE). MG's digit-sum D1 method
isn't expressible by the current `CheckDigit`/`DVRule` model — extending codegen IE needs a
new DV rule and is tracked as a separate follow-up (does not block the Go library).

## Generation & format coverage
- **SP, MG, RS, PR**: constructive `Generate`/`GenerateRand` (random base + computed DVs) and
  `Format` masks all implemented and round-trip tested.
- Other UFs: when added, generation is optional — leave `generate: nil` and document the
  limitation rather than fake it.

## Remaining-UF roadmap (follow-up plans)
Ship in batches, each gated by the same "authoritative source + ≥2 verified samples" rule.
First batch (SP, MG, RS, PR) is done; **RJ is blocked** on an official weight vector. Next
batch (highest population/value): BA, PE, CE, GO, SC, DF, ES, … through the remaining UFs.

- [x] SP  [x] MG  [x] RS  [x] PR  — ✅ shipped
- [ ] RJ  — ⛔ blocked (official page omits the weight vector; see Per-UF status)
- [ ] BA  [ ] PE  [ ] CE  [ ] GO  [ ] SC  [ ] DF  [ ] ES  [ ] PA  [ ] MA  [ ] MT
- [ ] MS  [ ] PB  [ ] RN  [ ] AL  [ ] PI  [ ] AM  [ ] SE  [ ] RO  [ ] TO  [ ] AC
- [ ] AP  [ ] RR

Per-UF many states accept multiple formats and a few use mod 10 — shape each `ieAlgo` entry
accordingly (the `lengths []int` field already supports multi-length UFs).

## Follow-up
- ✅ `GeneratePerson` carries a UF-consistent `IE` for the implemented UFs (SP/MG/RS/PR),
  generated for the person's own UF (`person.go`).
- Close the codegen parity gap: teach `internal/codegen` to emit MG/RS/PR IE (needs a
  digit-sum DV rule for MG); until then the generated targets validate SP IE only.
- Source an official-anchored RJ algorithm/sample to unblock RJ.
