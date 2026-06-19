# Inscrição Estadual (IE) — research log & roadmap

> Status as of 2026-06-19 (plan 006, design/spike). **First batch shipped: SP only.**
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
Minor UX note (shared with RG, out of scope): an unimplemented `--uf` prints `invalid`/exit 1
rather than a distinct "UF not implemented" message — a future CLI polish, not an IE bug.

## Per-UF status

| UF | Status | Notes |
|----|--------|-------|
| SP | **ready** ✅ | 12 digits; two check digits (pos 9 and 12), mod-11 "rightmost digit" rule. Implemented + tested. |
| MG | needs-research | 13 digits, mod-11; algorithm is unusually involved (zero-padded UF prefix, two DVs). Found descriptions but not ≥2 independently-verifiable samples — defer. |
| RJ | needs-research | 8 digits, single DV, weights 2..7 mod 11. Plausible algorithm located; need ≥2 sourced samples to verify before shipping. |
| RS | needs-research | 10 digits, single DV mod 11. Need authoritative algorithm + samples. |
| PR | needs-research | 10 digits, two DVs mod 11 (PR Fazenda publishes the general DV method). Need ≥2 sourced samples. |

Only **SP** met the "authoritative algorithm + ≥2 verified samples" bar within this spike.
The other four first-batch candidates are deferred (not implemented) rather than shipped
unverified — per the plan, shipping SP alone verified is success; shipping wrong ones is
failure.

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

## Generation & format coverage
- **SP**: constructive `Generate` (random base + computed DVs) and `Format` mask both
  implemented and round-trip tested.
- Other UFs: when added, generation is optional — leave `generate: nil` and document the
  limitation rather than fake it.

## Remaining-UF roadmap (follow-up plans)
Ship in batches, each gated by the same "authoritative source + ≥2 verified samples" rule.
Suggested next batch (highest population/value): **MG, RJ, RS, PR** (finish the first-batch
five), then BA, PE, CE, GO, SC, DF, ES, … through the remaining 26 UFs.

- [ ] MG  [ ] RJ  [ ] RS  [ ] PR  — finish first batch
- [ ] BA  [ ] PE  [ ] CE  [ ] GO  [ ] SC  [ ] DF  [ ] ES  [ ] PA  [ ] MA  [ ] MT
- [ ] MS  [ ] PB  [ ] RN  [ ] AL  [ ] PI  [ ] AM  [ ] SE  [ ] RO  [ ] TO  [ ] AC
- [ ] AP  [ ] RR

Per-UF many states accept multiple formats and a few use mod 10 — shape each `ieAlgo` entry
accordingly (the `lengths []int` field already supports multi-length UFs).

## Follow-up
- Once IE matures, consider adding an `IE` field to `GenPerson` (`person.go`) for the
  person's UF (currently absent) — a natural extension.
