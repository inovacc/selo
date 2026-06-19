# Design Spec: `selo` multi-language code generation

**Status:** Draft for review · **Date:** 2026-06-19 · **Author:** (advisor)
**Decision inputs (chosen):** languages = TypeScript, JavaScript, Ruby, Java, C# · output = module + golden vectors + test file · coverage = all 13 kinds · process = spec + plan, then build.

## 1. Goal
From the verified Go `selo` library (the single source of truth), generate idiomatic
**validation / formatting / origin** code for all 13 Brazilian document kinds in **TypeScript,
JavaScript, Ruby, Java, and C#**. Every generated target ships with **golden test vectors**
produced by the Go library and a **runnable test** that asserts against them, so correctness is
verifiable in each ecosystem and cannot silently drift.

Non-goal (this round): publishable package metadata (npm publish config, gem release, Maven
Central coordinates). We emit a *runnable* module + tests, not a release pipeline per language.

## 2. The dominant risk and the mitigation
Porting 13 check-digit algorithms into 5 languages is up to ~65 implementations — each a chance
to ship a subtly-wrong validator (negative modulo, 0-based vs 1-based indexing, integer width,
Unicode). This is the "wrong at scale" failure the project has guarded against (RG/IE plans).

**Mitigation — golden test vectors as the correctness contract.** The Go library generates, per
kind, a JSON vector file of valid/invalid/format/origin cases (and round-trip generate samples).
Each generated module ships a test that runs those vectors. A wrong port fails its test. The Go
test that *produces* the vectors guarantees the vectors themselves match the verified library.

## 3. Document-kind taxonomy (drives the architecture)
The 13 kinds do **not** share one algorithm shape. Grouping them is the key design step:

| Group | Kinds | Shape | Generator needs |
|---|---|---|---|
| **A. Numeric mod-11 check-digit** | CPF, PIS/PASEP/NIS, RENAVAM, CNH, RG, IE | weight-vector(s) + DV reduction | declarative `CheckDigitSpec` |
| **B. Mod-11 verify (sum≡0)** | CNS | weighted sum mod 11 == 0; two sub-formats (def./prov.) | `CheckDigitSpec` variant (verify mode) |
| **C. Alphanumeric mod-11** | CNPJ | char→value map (0-9, A-Z→17-42) + weights 2..9 RL + legacy numeric | `CheckDigitSpec` + char-map |
| **D. Pattern / regex** | license plate (national + Mercosul) | format only, no check digit | regex template per language |
| **E. Composite** | PIX key | dispatch: CPF/CNPJ key (reuse A/C) + email/phone/EVP regex | composition + regex |
| **F. Table lookup (validation + origin)** | CEP, phone, Título Eleitoral | format + data table (CEP→UF ranges, DDD→UF set, voter UF code + mod-11) | embedded data tables |

DV reduction rules vary even within group A: `11 - (sum % 11)` (CPF/RG), "rightmost digit of
`sum % 11`" (IE), special remainder `0/1 → 0` (CPF/CNPJ), check-char encodings (`10→X`, `11→0`),
and the CPF all-equal rejection. The spec model must capture all of these as data.

**Origin** (UF/region) applies to CPF (region from 9th digit), CEP (prefix range), phone (DDD),
voter ID (embedded UF code). UF-scoped kinds (RG, IE) take a UF parameter.

## 4. The golden-vector contract (stable JSON schema)
Emitted by Go, per kind, e.g. `vectors/cpf.json`:
```json
{
  "kind": "cpf",
  "validate": [
    {"input": "529.982.247-25", "valid": true},
    {"input": "52998224725",    "valid": true},
    {"input": "111.111.111-11", "valid": false},
    {"input": "529.982.247-26", "valid": false},
    {"input": "123",            "valid": false}
  ],
  "format": [
    {"input": "52998224725", "output": "529.982.247-25"},
    {"input": "123",         "error": "ErrInvalidLength"}
  ],
  "origin":   [ {"input": "529.982.247-25", "output": "<region>"} ],
  "ufScoped": false,
  "generateRoundTrip": 100
}
```
Invalid cases are produced **systematically** from the Go side (wrong DV, wrong length, all-equal,
bad chars, near-miss check digits) to exercise each validator branch — not hand-picked. Valid
cases mix curated authoritative samples (RG/IE) with `Generate()` output. The `generateRoundTrip`
count tells each language test to generate N and assert each `Validate==true` *where generation is
implemented*.

## 5. Generated surface (per kind, per language)
- **Required:** `validate(input) -> bool`; `format(input) -> string | error`; `origin(input)`
  for groups with origin; UF-param variants for RG/IE.
- **Best-effort (stretch):** `generate()`. Cross-language random generation is a large add and is
  **not required** in the first build — vectors cover validation/format/origin fully, and round-trip
  generate tests run only where `generate()` was emitted. (Recommendation: ship Validate+Format+
  Origin first; add Generate per-language in a follow-up.)
- Error reporting: idiomatic per language (TS/JS throw or return discriminated result; Ruby raises;
  Java/C# throw a typed exception or return a result type). Vector "error" cases assert the failure
  mode, mapped per language. The sentinel name (`ErrInvalidLength`, …) travels in the vector; each
  emitter maps it to its idiom.

## 6. Generator architecture
New Go package `internal/codegen` (internal: not part of the public library API):
- `spec.go` — `CheckDigitSpec`, `FormatSpec`, `OriginTable`, `KindPlan` (which group + which
  emitter features a kind uses); a registry `Kind → KindPlan`.
- `vectors.go` — imports `selo`; produces the golden vectors from the live library (the only place
  that runs the real algorithms).
- `data.go` — extracts the embedded data tables (CEP ranges, DDD→UF, CPF region map) from `selo`
  so emitters can render them as language constants.
- Per-language emitters using `text/template` with `//go:embed templates/<lang>/*.tmpl`:
  `emit_ts.go`, `emit_js.go`, `emit_ruby.go`, `emit_java.go`, `emit_csharp.go`. Each consumes the
  spec + tables + vectors and writes module + test + vector files.
- A small shared **mod-11 engine template fragment** per language (the weight-vector reducer) that
  the group A/B/C kinds parameterize — one reducer per language, not one per kind.

Surfaces:
- CLI: `selo gen --lang ts|js|ruby|java|csharp|all --kind <kind>|all --out <dir>` (lists langs/kinds
  from the registry; defaults: all kinds, `--out ./generated`).
- MCP tool: `generate_code{ lang, kind }` returning the file set (or a single file).

## 7. Output layout (`--out ./generated`)
```
generated/
  typescript/{src/<kind>.ts, src/index.ts, test/<kind>.test.ts, vectors/<kind>.json,
              package.json, tsconfig.json, vitest.config.ts}
  javascript/{src, test, vectors, package.json}
  ruby/{lib/selo/<kind>.rb, test/<kind>_test.rb, vectors, Gemfile, Rakefile}
  java/{src/main/java/com/inovacc/selo/<Kind>.java, src/test/java/..., vectors, pom.xml}
  csharp/{Selo/<Kind>.cs, Selo.Tests/<Kind>Tests.cs, vectors, Selo.sln, *.csproj}
```
Build files are the **minimum to run the tests** (per the chosen "module + test" scope), not full
publish scaffolding.

## 8. Verification strategy
1. **Vectors never drift from `selo`** — a Go test in `internal/codegen` regenerates vectors and
   asserts they equal the committed ones (golden-file). Changing an algorithm forces a vector update.
2. **Emitter output is snapshot-tested** — Go golden-file tests over the generated source so template
   changes are reviewable diffs.
3. **Generated code passes its vectors** — each language ships a test runner. Running them needs that
   toolchain, so it is **opt-in**: `task gen:verify:ts|ruby|java|csharp` runs the suite *if* the
   toolchain is present; a CI matrix job per language (node/ruby/jdk/dotnet) runs them on PRs that
   touch `internal/codegen` or `templates/`. The non-negotiable guarantee (vectors correct + tests
   shipped) holds even without the toolchains.

## 9. Milestones (the plan will sequence these)
- **M1 — Framework:** `internal/codegen` spec types, vector emitter, data extraction, Go tests. No
  language output yet. Gate: `go test ./internal/codegen` green; vectors produced for all 13 kinds.
- **M2 — TypeScript (all 13 kinds):** emitter + module + vectors + Vitest tests; `task gen:verify:ts`
  green. This is the reference emitter that proves every kind group end-to-end.
- **M3 — JavaScript:** derive from the TS design (shared shape).
- **M4 — Ruby.** **M5 — Java.** **M6 — C#.**
- Each language milestone gate: generated code validates all 13 kinds' vectors (validate/format/origin)
  with zero failures.

The irregular kinds (CNPJ alphanumeric, PIX, plate, CEP/phone/voter origin tables) are the hard part
*in every language*; each milestone handles them kind-by-kind with the shared data tables.

## 10. Risks & mitigations
- **Correctness across languages** → golden vectors (the backbone).
- **Per-language integer/modulo/Unicode quirks** → the shared mod-11 reducer per language is written
  once and vector-tested; documented assumptions.
- **Maintenance (5×13)** → spec-driven core + one reducer per language minimize per-kind code; honest
  that irregular kinds remain bespoke. Algorithm changes flow from Go → vectors → failing language
  tests → regenerate.
- **CI toolchain weight** → opt-in per-language verify jobs; default CI keeps running just Go.
- **Scope** → large, multi-milestone; phased so TypeScript (M2) delivers standalone value early.

## 11. Open questions (resolve before/within planning)
1. **Generate() parity:** ship Validate+Format+Origin first and defer cross-language `generate()`?
   (Recommended: yes — bounds the hardest part; vectors fully cover the read path.)
2. **Error idiom:** throw vs result-type for TS/Java/C#? (Recommended: throw a typed error/exception
   matching each ecosystem's norm; vectors assert the failure occurs, not its representation.)
3. **CI execution now or later:** wire the per-language matrix jobs in this project, or document
   `task gen:verify:*` and add CI later? (Recommended: add the Taskfile targets now; CI matrix as a
   follow-up to avoid heavy default CI.)
4. **Where generated output lives:** committed under `generated/` in this repo (as examples/tests) or
   generated on demand only? (Recommended: commit the TypeScript reference output + vectors so the
   snapshot tests have a baseline; others generated on demand + in CI.)

## 12. Out of scope
- Publishable package release pipelines per language.
- Languages beyond the five chosen (Python/PHP/Rust/Go-as-target) — easy to add later via a new
  emitter once the framework exists.
- Changing any Go algorithm (this is additive; Go remains the source of truth).
