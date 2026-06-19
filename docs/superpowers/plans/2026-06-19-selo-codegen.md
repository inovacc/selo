# selo multi-language code generation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use superpowers:subagent-driven-development (or
> executing-plans) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** From the verified Go `selo` library, generate idiomatic **validate/format/origin** code
for all 13 document kinds in TypeScript, JavaScript, Ruby, Java, and C#, each shipped with
**golden test vectors** (produced by Go) and a runnable test.

**Architecture:** A Go generator (`internal/codegen`) holds a declarative per-kind spec + data
tables + per-language `text/template` emitters (`//go:embed`). The Go library is the single source
of truth; it emits golden JSON vectors per kind, and every generated module ships a test that runs
them. Correctness is enforced by vectors, not by hand. Surfaces: `selo gen` CLI + `generate_code`
MCP tool. Reference: design spec `docs/superpowers/specs/2026-06-19-selo-codegen-design.md`.

**Tech Stack:** Go 1.25 (`text/template`, `//go:embed`, `encoding/json`); generated targets use
Vitest (TS/JS), Minitest (Ruby), JUnit/Maven (Java), xUnit/dotnet (C#).

## Global Constraints
- **Do not change any Go algorithm.** This is additive; `selo` remains the source of truth.
- **Generated code must pass its golden vectors.** A wrong port = a failing vector test.
- **Vectors come only from the live `selo` library** (never hand-authored values).
- Surface per kind: `validate`, `format`, `origin` (where the kind supports it), UF-param variants
  for RG/IE. `generate()` is **deferred** (follow-up), per the design decision.
- Errors: idiomatic per language (throw/raise/typed exception); vectors assert the failure occurs.
- `internal/codegen` is internal (not public API). New deps: none in the core module.
- Match repo conventions: table-driven Go tests with testify; conventional commits; gofmt/vet clean.
- Commit the **TypeScript** reference output + vectors under `generated/typescript/`; other
  languages are generated on demand (and in the opt-in verify task).

---

## Milestone M1 — Generator framework (Go, no language output yet)

### Task 1: Kind spec model + registry
**Files:** Create `internal/codegen/spec.go`, `internal/codegen/spec_test.go`.
**Interfaces — Produces:**
- `type DVRule int` (`DVElevenMinus`, `DVModRemainder`, `DVRightmostDigit`, `DVSumZero`)
- `type CheckDigit struct { Weights []int; Rule DVRule; SpecialZeroOneToZero bool; EncodeXAt, EncodeZeroAt int }`
- `type KindPlan struct { Kind selo.Kind; Group string; Lengths []int; Checks []CheckDigit; CharMap bool; AllEqualReject bool; Mask string; Origin OriginKind; UFScoped bool; Pattern string }`
- `var Plans map[selo.Kind]KindPlan` and `func PlanFor(k selo.Kind) (KindPlan, bool)`

- [ ] **Step 1: failing test** — assert `Plans` has an entry for every `selo.Kinds()` and each
  entry's `Mask`/`Lengths` are non-empty (or `Pattern` set for group D).
```go
func TestPlans_CoverAllKinds(t *testing.T) {
    for _, k := range selo.Kinds() {
        if _, ok := codegen.PlanFor(k); !ok { t.Errorf("no codegen plan for kind %q", k) }
    }
}
```
- [ ] **Step 2:** run → FAIL (package missing).
- [ ] **Step 3:** implement the types + `Plans` mapping all 13 kinds to the §3 taxonomy groups
  (A: CPF, PIS, RENAVAM, CNH, RG, IE; B: CNS; C: CNPJ; D: plate; E: PIX; F: CEP, phone, voter_id).
  Source the weights/rules/masks from the existing Go files (`cpf.go`, `cnpj.go`, `rg.go`, `ie.go`,
  `pis.go`, etc.) — read them; do not invent.
- [ ] **Step 4:** run → PASS. `go vet ./...`.
- [ ] **Step 5:** commit `feat(codegen): kind spec model + registry`.

### Task 2: Golden vector emitter
**Files:** Create `internal/codegen/vectors.go`, `internal/codegen/vectors_test.go`.
**Interfaces — Produces:** `type Vector struct{ Validate []ValidateCase; Format []FormatCase; Origin []OriginCase; UFScoped bool; GenerateRoundTrip int }`; `func Vectors(k selo.Kind) (Vector, error)`; `func WriteVectors(dir string) error` (writes `<kind>.json`).
- Valid cases: curated authoritative samples (RG/IE/IE-NOTES, CPF/CNPJ from README) + N from `selo.Generate(k)`. Invalid cases: systematic mutations (wrong DV last digit, truncated length, all-equal, injected letter) — each confirmed `false` by `selo.Validate(k, …)` before inclusion.
- [ ] **Step 1: failing test** — every emitted `validate` case's `valid` field must equal
  `selo.Validate(kind, input)`; every `format` case matches `selo.Format`; ≥4 valid and ≥4 invalid
  per kind.
- [ ] **Step 2:** run → FAIL.
- [ ] **Step 3:** implement `Vectors`/`WriteVectors`; build invalids via mutation helpers that
  re-check against `selo` so a mutation that accidentally stays valid is dropped.
- [ ] **Step 4:** run → PASS; `go test ./internal/codegen` green.
- [ ] **Step 5:** commit `feat(codegen): golden vector emitter from the selo source of truth`.

### Task 3: Data-table extraction
**Files:** Create `internal/codegen/data.go`, `internal/codegen/data_test.go`.
**Produces:** `func CEPRanges() []UFRange`, `func DDDtoUF() map[string]selo.UF`, `func CPFRegions() map[int]string` — serializable forms of the tables `selo` uses, for emitters to render as language constants.
- [ ] Step 1: failing test — spot-check known mappings via `selo` origin (e.g. CEP `01310-100`→SP,
  DDD 11→SP) match the extracted tables. Step 2: FAIL. Step 3: implement (read from selo's tables).
  Step 4: PASS. Step 5: commit `feat(codegen): extract CEP/DDD/region data tables`.

### Task 4: `selo gen` CLI + MCP tool skeleton
**Files:** Create `cmd/selo/gen.go`, `cmd/selo/gen_test.go`; modify `mcp/server.go` (+ a tool).
**Produces:** `selo gen --lang <l> --kind <k>|all --out <dir>` (langs/kinds listed from the registry; emitters registered by M2+); MCP `generate_code{lang,kind}`.
- [ ] Step 1: failing test — `selo gen --help` lists langs `ts,js,ruby,java,csharp`; `--lang bogus`
  exits non-zero. Step 2: FAIL. Step 3: implement command wiring + a `map[lang]Emitter` registry
  (empty until M2). Step 4: PASS. Step 5: commit `feat(cli): selo gen command + MCP generate_code`.

**M1 gate:** `go build ./... && go vet ./... && go test ./...` green; `selo gen --help` works.

---

## Milestone M2 — TypeScript emitter (reference; all 13 kinds)

### Task 5: TS mod-11 engine + group A/B/C templates
**Files:** Create `internal/codegen/emit_ts.go`, `internal/codegen/templates/ts/*.tmpl`.
- One shared `mod11.ts` reducer parameterized by `CheckDigit` (weights, rule, special-zero,
  X/0 encoding). Per-kind module renders `validate`/`format` from the `KindPlan`.
- [ ] Step 1: failing Go test — `EmitTS(plan, vec)` returns non-empty source for CPF containing
  `export function isCPF`. Step 2: FAIL. Step 3: implement emitter + templates for CPF, CNPJ
  (char-map), PIS, RENAVAM, CNH, CNS (sum≡0), RG/IE (UF param). Step 4: PASS. Step 5: commit.

### Task 6: TS templates for irregular kinds (D/E/F)
**Files:** Add `internal/codegen/templates/ts/{plate,pix,cep,phone,voterid}.tmpl`.
- plate = regex (national + Mercosul); pix = compose CPF/CNPJ + email/phone/EVP regex; cep/phone/
  voterid = format + embedded data tables (from Task 3) + `origin`.
- [ ] Step 1: failing test — `EmitTS` covers all 13 `selo.Kinds()` (non-empty each). Step 2: FAIL.
  Step 3: implement. Step 4: PASS. Step 5: commit `feat(codegen): TS templates for plate/pix/cep/phone/voter`.

### Task 7: TS project scaffolding + vector test harness
**Files:** Add `internal/codegen/templates/ts/{package.json,tsconfig.json,vitest.config,index,test}.tmpl`.
- Emit `package.json` (vitest devDep), `tsconfig.json`, `vitest.config.ts`, `src/index.ts`
  (re-exports), and `test/<kind>.test.ts` that loads `vectors/<kind>.json` and asserts
  validate/format/origin.
- [ ] Step 1: failing test — emitted file set for `--lang ts --out tmp` includes the 13 modules,
  13 tests, 13 vector files, and the 4 scaffold files. Step 2: FAIL. Step 3: implement. Step 4: PASS.
  Step 5: commit.

### Task 8: Wire `selo gen --lang ts`, commit reference output, snapshot test, verify task
**Files:** Modify `cmd/selo/gen.go` (register TS emitter); create `generated/typescript/**`
(committed reference output); add `internal/codegen/golden_test.go`; modify `Taskfile.yml`.
- [ ] Step 1: `selo gen --lang ts --out generated/typescript --kind all`.
- [ ] Step 2: Go snapshot test asserts re-emitting equals the committed `generated/typescript` tree.
- [ ] Step 3: add `task gen:verify:ts` → `cd generated/typescript && npm i && npx vitest run`
  (runs only if node present; document the guard).
- [ ] Step 4: run `task gen:verify:ts` → all 13 kinds' vectors pass (if node available in env).
- [ ] Step 5: `go test ./...` green; commit `feat(codegen): TypeScript target (all 13 kinds, vectors, tests)`.

**M2 gate:** generated TS validates all 13 kinds' vectors (validate/format/origin) with zero failures;
Go snapshot + vector tests green.

---

## Milestones M3–M6 — JavaScript, Ruby, Java, C# (each mirrors M2)
Each milestone reuses the M2 structure: a per-language reducer + group templates + irregular-kind
templates + scaffolding + a vector test harness, then `task gen:verify:<lang>`.

- **M3 JavaScript:** `emit_js.go` + `templates/js/*` (ESM; vitest). Gate: `task gen:verify:js` green.
- **M4 Ruby:** `emit_ruby.go` + `templates/ruby/*` (module per kind; Minitest loads vectors; Rakefile).
  Gate: `task gen:verify:ruby` green.
- **M5 Java:** `emit_java.go` + `templates/java/*` (`com.inovacc.selo`; JUnit5 + Maven `pom.xml`).
  Gate: `task gen:verify:java` green.
- **M6 C#:** `emit_csharp.go` + `templates/csharp/*` (xUnit + `dotnet test`; `.csproj`/`.sln`).
  Gate: `task gen:verify:csharp` green.

Per milestone (task pattern, mirroring Tasks 5–8): (a) reducer + group A/B/C templates; (b) irregular
D/E/F templates; (c) scaffolding + vector test harness; (d) wire `selo gen --lang <l>`, add
`task gen:verify:<l>`, verify all 13 kinds, commit. Each ends with all 13 kinds' vectors passing.

---

## Milestone M7 — Docs, Taskfile, CHANGELOG
- [ ] `docs/CODEGEN.md` — how `selo gen` works, the golden-vector contract, per-language layout,
  `task gen:verify:*`, and how to add a language/kind.
- [ ] README — "Code generation" section (`selo gen --lang ts --kind cpf`, supported langs/kinds).
- [ ] `Taskfile.yml` — `gen:ts/js/ruby/java/csharp` (emit) and `gen:verify:*` (run if toolchain present).
- [ ] CHANGELOG `[Unreleased]`/next minor entry. Commit `docs: document selo gen code generation`.

## Test plan
- Go: `internal/codegen` — registry coverage, vector correctness vs `selo`, emitter non-emptiness,
  golden snapshots of the TS reference output; `cmd/selo` — `gen` flag/help/error tests.
- Generated: per language, a vector-driven test per kind (validate/format/origin). Run via
  `task gen:verify:<lang>` where the toolchain exists; the TS suite is the reference.
- Verification commands: `go build ./... && go vet ./... && go test ./...`; `task gen:verify:ts` (etc.).

## Done criteria (whole feature)
- [ ] `selo gen --lang {ts,js,ruby,java,csharp} --kind all --out <dir>` emits module + vectors +
  tests for all 13 kinds.
- [ ] Each language's vector tests pass for all 13 kinds (validate/format/origin) where the toolchain
  is available; the committed TS reference passes in CI-capable form.
- [ ] `go build/vet/test ./...` green; gofmt clean; `internal/codegen` covered by Go tests.
- [ ] `docs/CODEGEN.md`, README section, Taskfile targets, CHANGELOG updated.

## Risks / escape hatches
- If a kind's algorithm doesn't fit the declarative `CheckDigit` model (e.g. CNS dual-format, voter
  UF computation), give it a dedicated template fragment rather than forcing the spec — note it in
  `KindPlan.Group`. **STOP and report** if any kind would require changing a Go algorithm.
- If a target toolchain is unavailable in the build env, emit + Go-snapshot-test the output and skip
  `gen:verify:<lang>` (don't fake a pass); record it.
- Generation parity (`generate()` in target languages) is explicitly **out of scope** here; if a
  vector needs a generated sample, use `selo.Generate` on the Go side, not target-language generation.

## Maintenance notes
- Algorithm change flow: edit Go → vectors change → committed TS snapshot + each language's vector
  test fail → regenerate. The generator, not humans, keeps the 5×13 surface correct.
- Adding a language = one `emit_<lang>.go` + a `templates/<lang>/` set + a `gen:verify:<lang>` task.
- Adding a kind = one `KindPlan` entry (+ a template fragment if irregular); all languages pick it up.
