# Code generation (`selo gen`)

`selo` generates idiomatic **validation / formatting / origin** code for all 13 document kinds
in **TypeScript, JavaScript, Ruby, Java, C#, and Python**, from the verified Go library — the
single source of truth. Every generated target ships with **golden test vectors** produced by Go
and a runnable test suite, so a wrong port fails its own tests.

## Quick start

```bash
# Emit one kind for one language to ./out
selo gen --lang ts --kind cpf --out ./out

# Emit all 13 kinds
selo gen --lang ruby --kind all --out ./generated/ruby

# Languages: ts | js | ruby | java | csharp | python   Kinds: all + the 13 document kinds
```

Or via Task (toolchain-guarded — skips the language test run when its toolchain is absent):

```bash
task gen:ts          # emit the TypeScript target into generated/typescript
task gen:verify:ts   # emit + run the vitest suite (needs node)
task gen:all         # emit all six language targets
# gen:js / gen:ruby / gen:java / gen:csharp / gen:python and gen:verify:<lang> likewise
```

MCP agents can call the **`generate_code`** tool (`{lang, kind}`) to get the file set back.

## The correctness contract: golden vectors

The risk in porting 13 check-digit algorithms into 6 languages is shipping a subtly-wrong
validator. The defense: for each kind, Go emits a `vectors/<kind>.json` file of valid/invalid/
format/origin cases (produced by the *verified* `selo` library — valid samples include
authoritative ones and `Generate()` output; invalid samples are systematic mutations re-checked
against `selo`). Each generated module ships a test that runs those vectors. **A wrong port fails
its vector test.** Correctness is enforced, not assumed.

## Per-language output layout

```
typescript/  src/<kind>.ts, src/{mod11,data,index}.ts, test/<kind>.test.ts,
             vectors/<kind>.json, package.json, tsconfig.json, vitest.config.ts
javascript/  src/*.js, test/*.test.js, vectors/, package.json, vitest.config.js
ruby/        lib/selo/<kind>.rb, lib/selo/{mod11,data}.rb, test/<kind>_test.rb,
             vectors/, Gemfile, Rakefile        (Minitest)
java/        src/main/java/com/inovacc/selo/<Kind>.java, .../Mod11.java, .../Data.java,
             src/test/java/.../<Kind>Test.java, vectors/, pom.xml   (JUnit 5 + Jackson)
csharp/      src/Selo/<Kind>.cs, src/Selo/{Mod11,Data}.cs,
             src/Selo.Tests/<Kind>Tests.cs, vectors/, Selo.sln, *.csproj   (xUnit)
python/      selo/<kind>.py, selo/{mod11,data,__init__}.py, tests/test_<kind>.py,
             vectors/, pyproject.toml   (pytest, stdlib-only)
```

The TypeScript output is committed under `generated/typescript/` as the reference baseline (a Go
snapshot test pins it byte-for-byte); the others are generated on demand and verified in CI.

## Architecture (`internal/codegen`)

- `spec.go` — a declarative per-kind model (`KindPlan`, `CheckDigit`, `DVRule`) mapping each kind
  to its group (numeric mod-11, alphanumeric CNPJ, pattern/plate, composite PIX, table-lookup
  CEP/phone/voter). Sourced from the live Go algorithms.
- `vectors.go` — produces the golden vectors from the `selo` public API (the only place that runs
  the real algorithms).
- `data.go` — extracts the data tables (CEP ranges, DDD→UF, CPF region, voter UF names) for
  emitters to render as language constants (exposed via `selo`'s `tables.go`).
- `emit_<lang>.go` + `templates/<lang>/*.tmpl` (`//go:embed`) — one emitter per language: a shared
  mod-11 reducer + per-kind renderers + scaffolding + the vector test harness. Each self-registers
  for its `Lang` in `init()`.
- `golden_test.go` / `golden_<lang>_test.go` — snapshot the deterministic emitted files and
  re-validate the committed vectors against `selo` (line-ending-normalized for cross-OS stability).

The CLI (`cmd/selo/gen.go`) and the MCP `generate_code` tool derive the supported languages/kinds
from this registry.

## CI verification

`.github/workflows/codegen.yml` runs a matrix over `[ts, js, ruby, java, csharp, python]` on real
toolchains (node / ruby / JDK+Maven / .NET / Python), executing each language's vector tests. This is the
authoritative gate for the targets that a given dev machine can't run locally. It is path-scoped
(runs only when `internal/codegen/**`, `generated/**`, `cmd/selo/gen.go`, or the workflow change).

## Adding a language

1. Add `emit_<lang>.go` (implement `Emitter`, self-register the `Lang` in `init()`) + a
   `templates/<lang>/` set: a shared mod-11 reducer, per-kind renderers, scaffolding, and a
   vector-driven test harness. Mirror an existing emitter (TS is the reference).
2. Add `gen:<lang>` / `gen:verify:<lang>` to `Taskfile.yml` and a matrix entry in `codegen.yml`.
3. Translate the algorithms *faithfully* from the proven TS logic (the irregular kinds — CNPJ
   char-map, CNH coupled DVs, voter-ID DV2 dependency, IE rightmost-digit, CNS sum≡0 — and the
   mod-11 DV-fold disambiguation in `mod11`). The vector tests are the proof.

## Adding a kind

Add the kind to `selo` (it self-registers), then add a `KindPlan` entry in `spec.go`. All six
emitters pick it up; regenerate and let the CI matrix verify.

## Limitations

- Generated targets provide **Validate / Format / Origin** (and UF-scoped variants for RG/IE).
  Cross-language `generate()` is not yet emitted (tracked in `docs/BACKLOG.md`).
- Generated code is produced on demand (except the committed TypeScript reference); run
  `selo gen` / `task gen:<lang>` to materialize a target.
