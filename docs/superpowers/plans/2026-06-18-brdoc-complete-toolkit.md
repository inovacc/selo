# Complete Brazilian-Document Toolkit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.

**Goal:** Evolve the current CPF/CNPJ project into a complete Brazilian-document toolkit — one Go module that validates, generates, formats, and resolves origin for all 11 standard Brazilian document types plus PIX keys, exposed identically through a library, a Cobra CLI, and an MCP server.

**Architecture:** A hybrid interface + registry design: every document type implements a frozen `Document` interface (plus optional `OriginResolver`/`UFScoped` capability interfaces) and self-registers in `init()`, while ergonomic concrete types (`NewCPF()`, `NewCNPJ()`) remain public. The root package (`brdoc`) owns the types, registry, errors, UF tables, and brand metadata; thin adapter subpackages (`cmd/brdoc`, `mcp/`, `compat/`) consume the root package only. Both the CLI subcommands and the MCP tools are DERIVED from the registry (`brdoc.Kinds()`) so adding a type never requires hand-duplicated per-type wiring.

**Tech Stack:** Go 1.24, Cobra, testify, modelcontextprotocol/go-sdk, math/rand/v2

## Global Constraints

- Language/floor: Go 1.24.0. Module path: github.com/inovacc/brdoc (root package "brdoc").
- Deps: github.com/spf13/cobra v1.10.1, github.com/stretchr/testify v1.11.1 (already present). MCP: github.com/modelcontextprotocol/go-sdk/mcp (add in the MCP milestone). No other new deps without justification.
- Randomness: use math/rand/v2 top-level funcs (goroutine-safe) — NOT a seeded *rand.Rand.
- Errors: sentinel errors compared with errors.Is/errors.As; wrap with %w. Never compare errors with ==.
- Tests: table-driven, testify, >=80% coverage (keep ~95%). Slow/fuzz tests guard with testing.Short() where relevant. Use "go run ./..." not build-then-run for manual checks.
- Style (Uber/Effective Go): mute unused returns with _, _ =; defer func(){ _ = x.Close() }(); inline error checks.
- Commits: conventional-commit messages (feat:, test:, refactor:, docs:, chore:). NO "Co-Authored-By" / AI attribution lines. Concise.
- Each new document type implements the frozen Document interface and self-registers via init(). CLI subcommands and MCP tools are DERIVED from the registry — never hand-duplicated per type.

---

## Milestone M0 — Foundation

> Outcome: the frozen contract (`document.go`, `errors.go`, `uf.go`, `registry.go`, `meta.go`) exists, and CPF + CNPJ are migrated onto it with ZERO behavior change — every existing `brdoc_test.go` test stays green. `ValidateDocument` becomes a deprecated wrapper over `Detect`+`Validate`. The seeded `*rand.Rand` is replaced with `math/rand/v2`.

### Task M0-1: Document interface, Kind type & constants

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/document.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/document_test.go`

**Interfaces:**
- Consumes: (none)
- Produces:
  - `type Kind string` with constants `KindCPF, KindCNPJ, KindCNH, KindPIS, KindRenavam, KindVoterID, KindCEP, KindPhone, KindPlate, KindCNS, KindRG, KindPIX Kind` (string values `"cpf","cnpj","cnh","pis","renavam","voter_id","cep","phone","plate","cns","rg","pix"`)
  - `func (k Kind) String() string`
  - `type Document interface { Kind() Kind; Validate(value string) bool; Generate() string; Format(value string) (string, error) }`
  - `type OriginResolver interface { Origin(value string) (string, error) }`
  - `type UF string` (full definition + constants land in M0-3; declared here only as the param type referenced by `UFScoped`) — defined in M0-3, NOT here.
  - `type UFScoped interface { ValidateUF(value string, uf UF) (bool, error); ImplementedUFs() []UF }`

> NOTE: `UF` is the one cross-task dependency inside M0. `UFScoped` references `UF`, but the `UF` type itself is created in M0-3. To keep each task independently compilable, M0-1 defines everything EXCEPT `UFScoped`, and M0-1's final step adds `UFScoped` only AFTER you have confirmed `uf.go` exists. If executing strictly in order, do M0-3 before M0-1's Step 5, OR put the placeholder `type UF string` line in M0-1 and delete it in M0-3 (chosen approach below: M0-1 owns `UF string` declaration; M0-3 owns the constants + helpers). This keeps `document.go` self-contained.

- [ ] **Step 1: Write failing test for Kind string values.** Create `document_test.go`:
```go
package brdoc

import "testing"

func TestKind_String(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindCPF, "cpf"},
		{KindCNPJ, "cnpj"},
		{KindCNH, "cnh"},
		{KindPIS, "pis"},
		{KindRenavam, "renavam"},
		{KindVoterID, "voter_id"},
		{KindCEP, "cep"},
		{KindPhone, "phone"},
		{KindPlate, "plate"},
		{KindCNS, "cns"},
		{KindRG, "rg"},
		{KindPIX, "pix"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Fatalf("Kind.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL (compile error).** Run:
```
go test -run TestKind_String ./...
```
Expected FAIL: `./document_test.go:... undefined: KindCPF` (and the other Kind constants / `Kind` type are undefined).

- [ ] **Step 3: Create `document.go` with the frozen contract (minimal).** Write:
```go
package brdoc

// Kind is the stable identifier for a document type, e.g. "cpf".
type Kind string

// Document kind identifiers. Values are stable and used by the CLI and MCP adapters.
const (
	KindCPF     Kind = "cpf"
	KindCNPJ    Kind = "cnpj"
	KindCNH     Kind = "cnh"
	KindPIS     Kind = "pis"
	KindRenavam Kind = "renavam"
	KindVoterID Kind = "voter_id" // Título Eleitoral
	KindCEP     Kind = "cep"
	KindPhone   Kind = "phone"
	KindPlate   Kind = "plate"
	KindCNS     Kind = "cns"
	KindRG      Kind = "rg"
	KindPIX     Kind = "pix"
)

// String returns the stable string identifier of the Kind.
func (k Kind) String() string { return string(k) }

// UF is a Brazilian federative unit (state) two-letter code, e.g. "SP".
// The full constant set and helpers are defined in uf.go.
type UF string

// Document is implemented by every document type in the toolkit.
type Document interface {
	// Kind returns the stable identifier of this document type.
	Kind() Kind
	// Validate reports whether value is a well-formed document of this Kind
	// (formatted or unformatted input is accepted).
	Validate(value string) bool
	// Generate returns a freshly generated valid, unformatted document.
	Generate() string
	// Format returns value in the canonical masked representation for this Kind,
	// or a sentinel error (see errors.go) when value cannot be formatted.
	Format(value string) (string, error)
}

// OriginResolver is the optional capability for types that can resolve a
// geographic origin (CPF region, CEP/phone/voter UF). Discovered via type assertion.
type OriginResolver interface {
	Origin(value string) (string, error)
}

// UFScoped is the optional capability for types whose validation depends on a
// federative unit (notably RG). Discovered via type assertion.
type UFScoped interface {
	ValidateUF(value string, uf UF) (bool, error)
	ImplementedUFs() []UF
}
```

- [ ] **Step 4: Run the test, expect PASS.** Run:
```
go test -run TestKind_String ./...
```
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Add a compile-time interface assertion test for the contract shape.** Append to `document_test.go`:
```go
// stubDoc proves the Document/OriginResolver/UFScoped method sets compile as declared.
type stubDoc struct{}

func (stubDoc) Kind() Kind                                  { return KindCPF }
func (stubDoc) Validate(string) bool                        { return false }
func (stubDoc) Generate() string                            { return "" }
func (stubDoc) Format(string) (string, error)               { return "", nil }
func (stubDoc) Origin(string) (string, error)              { return "", nil }
func (stubDoc) ValidateUF(string, UF) (bool, error)        { return false, nil }
func (stubDoc) ImplementedUFs() []UF                        { return nil }

var (
	_ Document       = stubDoc{}
	_ OriginResolver = stubDoc{}
	_ UFScoped       = stubDoc{}
)
```
Run:
```
go test -run TestKind_String ./...
```
Expected PASS (the package compiles, including the `var _ Document = stubDoc{}` assertions).

- [ ] **Step 6: Commit.** Run:
```
git add document.go document_test.go
git commit -m "feat: add Kind type, Document interface and optional capability interfaces"
```

---

### Task M0-2: Sentinel errors

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/errors.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/errors_test.go`

**Interfaces:**
- Consumes: (none)
- Produces (package-level `error` vars):
  - `ErrInvalidLength`, `ErrInvalidFormat`, `ErrUnknownKind`, `ErrUnsupported`, `ErrUFNotImplemented`

- [ ] **Step 1: Write failing test asserting sentinels exist and are distinct.** Create `errors_test.go`:
```go
package brdoc

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors_DistinctAndWrappable(t *testing.T) {
	all := []error{
		ErrInvalidLength,
		ErrInvalidFormat,
		ErrUnknownKind,
		ErrUnsupported,
		ErrUFNotImplemented,
	}
	for i, e := range all {
		if e == nil {
			t.Fatalf("sentinel at index %d is nil", i)
		}
		// each sentinel must remain identifiable through %w wrapping
		wrapped := fmt.Errorf("ctx: %w", e)
		if !errors.Is(wrapped, e) {
			t.Fatalf("errors.Is failed to unwrap sentinel index %d", i)
		}
	}
	// distinctness: no two sentinels are Is-equal to each other
	for i := range all {
		for j := range all {
			if i != j && errors.Is(all[i], all[j]) {
				t.Fatalf("sentinels %d and %d are not distinct", i, j)
			}
		}
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL (compile error).** Run:
```
go test -run TestSentinelErrors ./...
```
Expected FAIL: `undefined: ErrInvalidLength` (and the other four sentinels).

- [ ] **Step 3: Create `errors.go`.** Write:
```go
package brdoc

import "errors"

// Sentinel errors. Compare with errors.Is / errors.As; wrap with %w when adding context.
var (
	// ErrInvalidLength indicates the document does not have the expected number of characters.
	ErrInvalidLength = errors.New("brdoc: invalid document length")
	// ErrInvalidFormat indicates the document does not match the expected shape.
	ErrInvalidFormat = errors.New("brdoc: invalid document format")
	// ErrUnknownKind indicates a Kind that is not registered.
	ErrUnknownKind = errors.New("brdoc: unknown document kind")
	// ErrUnsupported indicates an operation is not supported for the given Kind.
	ErrUnsupported = errors.New("brdoc: operation not supported for this kind")
	// ErrUFNotImplemented indicates the requested federative unit has no implementation yet.
	ErrUFNotImplemented = errors.New("brdoc: federative unit not implemented")
)
```

- [ ] **Step 4: Run the test, expect PASS.** Run:
```
go test -run TestSentinelErrors ./...
```
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Commit.** Run:
```
git add errors.go errors_test.go
git commit -m "feat: add sentinel errors for validation, format, kind and UF"
```

---

### Task M0-3: UF type, 27 constants & stub lookup tables

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/uf.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/uf_test.go`

**Interfaces:**
- Consumes: `type UF string` (declared in M0-1's `document.go`)
- Produces:
  - 27 `UF` constants: `UFAC, UFAL, UFAP, UFAM, UFBA, UFCE, UFDF, UFES, UFGO, UFMA, UFMT, UFMS, UFMG, UFPA, UFPB, UFPR, UFPE, UFPI, UFRJ, UFRN, UFRS, UFRO, UFRR, UFSC, UFSP, UFSE, UFTO`
  - `func (u UF) String() string`
  - `func (u UF) Valid() bool`
  - `func AllUFs() []UF` (sorted, stable copy)
  - data tables (empty/stub, filled by later type tasks): `var cepRanges map[UF][2]int`, `var dddToUF map[int]UF`

> The `UF` type itself is declared in `document.go` (M0-1). This task adds only the constants, helpers, and the lookup-table vars. If M0-1 is not yet done, add `type UF string` here temporarily and remove it once M0-1 lands (do not duplicate the declaration).

- [ ] **Step 1: Write failing test for UF.Valid and AllUFs count.** Create `uf_test.go`:
```go
package brdoc

import "testing"

func TestUF_Valid(t *testing.T) {
	tests := []struct {
		uf   UF
		want bool
	}{
		{UFSP, true},
		{UFRJ, true},
		{UFDF, true},
		{UFRS, true},
		{UF("XX"), false},
		{UF(""), false},
		{UF("sp"), false}, // case-sensitive: constants are upper-case
	}
	for _, tt := range tests {
		t.Run(string(tt.uf), func(t *testing.T) {
			if got := tt.uf.Valid(); got != tt.want {
				t.Fatalf("UF(%q).Valid() = %v, want %v", tt.uf, got, tt.want)
			}
		})
	}
}

func TestAllUFs_Count(t *testing.T) {
	got := AllUFs()
	if len(got) != 27 {
		t.Fatalf("AllUFs() returned %d UFs, want 27", len(got))
	}
	// stable order: sorted ascending
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Fatalf("AllUFs() not strictly sorted at %d: %q >= %q", i, got[i-1], got[i])
		}
	}
	// returned slice must be a copy (mutating it must not affect later calls)
	got[0] = UF("ZZ")
	again := AllUFs()
	if again[0] == UF("ZZ") {
		t.Fatal("AllUFs() leaks internal backing array")
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL (compile error).** Run:
```
go test -run "TestUF_Valid|TestAllUFs_Count" ./...
```
Expected FAIL: `undefined: UFSP` (and `AllUFs`).

- [ ] **Step 3: Create `uf.go` with constants, helpers and stub tables.** Write:
```go
package brdoc

import "sort"

// The 27 Brazilian federative units (26 states + the Federal District).
const (
	UFAC UF = "AC" // Acre
	UFAL UF = "AL" // Alagoas
	UFAP UF = "AP" // Amapá
	UFAM UF = "AM" // Amazonas
	UFBA UF = "BA" // Bahia
	UFCE UF = "CE" // Ceará
	UFDF UF = "DF" // Distrito Federal
	UFES UF = "ES" // Espírito Santo
	UFGO UF = "GO" // Goiás
	UFMA UF = "MA" // Maranhão
	UFMT UF = "MT" // Mato Grosso
	UFMS UF = "MS" // Mato Grosso do Sul
	UFMG UF = "MG" // Minas Gerais
	UFPA UF = "PA" // Pará
	UFPB UF = "PB" // Paraíba
	UFPR UF = "PR" // Paraná
	UFPE UF = "PE" // Pernambuco
	UFPI UF = "PI" // Piauí
	UFRJ UF = "RJ" // Rio de Janeiro
	UFRN UF = "RN" // Rio Grande do Norte
	UFRS UF = "RS" // Rio Grande do Sul
	UFRO UF = "RO" // Rondônia
	UFRR UF = "RR" // Roraima
	UFSC UF = "SC" // Santa Catarina
	UFSP UF = "SP" // São Paulo
	UFSE UF = "SE" // Sergipe
	UFTO UF = "TO" // Tocantins
)

// allUFs is the canonical set, kept private so AllUFs can hand out copies.
var allUFs = []UF{
	UFAC, UFAL, UFAP, UFAM, UFBA, UFCE, UFDF, UFES, UFGO, UFMA,
	UFMT, UFMS, UFMG, UFPA, UFPB, UFPR, UFPE, UFPI, UFRJ, UFRN,
	UFRS, UFRO, UFRR, UFSC, UFSP, UFSE, UFTO,
}

var ufSet = func() map[UF]struct{} {
	m := make(map[UF]struct{}, len(allUFs))
	for _, u := range allUFs {
		m[u] = struct{}{}
	}
	return m
}()

// String returns the two-letter UF code.
func (u UF) String() string { return string(u) }

// Valid reports whether u is one of the 27 known federative units.
func (u UF) Valid() bool {
	_, ok := ufSet[u]
	return ok
}

// AllUFs returns a sorted, stable copy of the 27 federative units.
func AllUFs() []UF {
	out := make([]UF, len(allUFs))
	copy(out, allUFs)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// cepRanges maps a UF to its inclusive [low, high] 5-digit CEP prefix range.
// Populated by the CEP type task (cep.go); intentionally empty here.
var cepRanges = map[UF][2]int{}

// dddToUF maps a telephone area code (DDD) to its UF.
// Populated by the phone type task (phone.go); intentionally empty here.
var dddToUF = map[int]UF{}
```

- [ ] **Step 4: Run the test, expect PASS.** Run:
```
go test -run "TestUF_Valid|TestAllUFs_Count" ./...
```
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Silence "declared and not used" for stub tables via a guard test.** The empty `cepRanges`/`dddToUF` are package-level vars (always "used" by the compiler), but add a reference test so later tasks have a seam and the vars are exercised. Append to `uf_test.go`:
```go
func TestStubTables_Empty(t *testing.T) {
	if len(cepRanges) != 0 {
		t.Fatalf("cepRanges expected empty stub, got %d entries", len(cepRanges))
	}
	if len(dddToUF) != 0 {
		t.Fatalf("dddToUF expected empty stub, got %d entries", len(dddToUF))
	}
}
```
Run:
```
go test -run "TestUF_Valid|TestAllUFs_Count|TestStubTables_Empty" ./...
```
Expected PASS.

- [ ] **Step 6: Commit.** Run:
```
git add uf.go uf_test.go
git commit -m "feat: add UF type with 27 constants, Valid helper and stub lookup tables"
```

---

### Task M0-4: Registry & generic dispatch

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/registry.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/registry_test.go`

**Interfaces:**
- Consumes: `Kind`, `Document` (M0-1); `ErrUnknownKind` (M0-2)
- Produces:
  - `func Register(d Document)`
  - `func Get(kind Kind) (Document, bool)`
  - `func Kinds() []Kind` (sorted, stable)
  - `func Validate(kind Kind, value string) (bool, error)`
  - `func Generate(kind Kind) (string, error)`
  - `func Format(kind Kind, value string) (string, error)`
  - `func Detect(value string) (Kind, bool)` (length-based, see body)

- [ ] **Step 1: Write failing test using a fake registered Document.** Create `registry_test.go`:
```go
package brdoc

import (
	"errors"
	"testing"
)

// fakeDoc is a controllable Document for registry tests.
type fakeDoc struct {
	kind  Kind
	valid bool
	gen   string
	fmtd  string
	fmErr error
}

func (f fakeDoc) Kind() Kind            { return f.kind }
func (f fakeDoc) Validate(string) bool  { return f.valid }
func (f fakeDoc) Generate() string      { return f.gen }
func (f fakeDoc) Format(string) (string, error) {
	return f.fmtd, f.fmErr
}

func TestRegistry_RegisterGetDispatch(t *testing.T) {
	// Use a kind value that no real type registers, to avoid collisions.
	const testKind Kind = "test_fake"
	Register(fakeDoc{kind: testKind, valid: true, gen: "GEN", fmtd: "FMT"})

	got, ok := Get(testKind)
	if !ok {
		t.Fatal("Get returned ok=false for a registered kind")
	}
	if got.Kind() != testKind {
		t.Fatalf("Get returned wrong kind: %q", got.Kind())
	}

	ok, err := Validate(testKind, "x")
	if err != nil || !ok {
		t.Fatalf("Validate = (%v, %v), want (true, nil)", ok, err)
	}

	g, err := Generate(testKind)
	if err != nil || g != "GEN" {
		t.Fatalf("Generate = (%q, %v), want (GEN, nil)", g, err)
	}

	fm, err := Format(testKind, "x")
	if err != nil || fm != "FMT" {
		t.Fatalf("Format = (%q, %v), want (FMT, nil)", fm, err)
	}
}

func TestRegistry_UnknownKind(t *testing.T) {
	const missing Kind = "does_not_exist"
	if _, ok := Get(missing); ok {
		t.Fatal("Get returned ok=true for an unregistered kind")
	}
	if _, err := Validate(missing, "x"); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("Validate err = %v, want ErrUnknownKind", err)
	}
	if _, err := Generate(missing); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("Generate err = %v, want ErrUnknownKind", err)
	}
	if _, err := Format(missing, "x"); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("Format err = %v, want ErrUnknownKind", err)
	}
}

func TestRegistry_KindsSorted(t *testing.T) {
	ks := Kinds()
	for i := 1; i < len(ks); i++ {
		if ks[i-1] >= ks[i] {
			t.Fatalf("Kinds() not strictly sorted at %d: %q >= %q", i, ks[i-1], ks[i])
		}
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL (compile error).** Run:
```
go test -run TestRegistry ./...
```
Expected FAIL: `undefined: Register` (and `Get`, `Validate`, `Generate`, `Format`, `Kinds`).

- [ ] **Step 3: Create `registry.go`.** Write:
```go
package brdoc

import (
	"fmt"
	"sort"
	"sync"
)

// registry holds the singleton Document per Kind. Populated from each type's init().
var (
	registryMu sync.RWMutex
	registry   = map[Kind]Document{}
)

// Register installs d as the singleton implementation for d.Kind().
// It is intended to be called from a type's init() function.
func Register(d Document) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[d.Kind()] = d
}

// Get returns the registered Document for kind, or ok=false if none is registered.
func Get(kind Kind) (Document, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	d, ok := registry[kind]
	return d, ok
}

// Kinds returns the registered kinds in stable, sorted order.
func Kinds() []Kind {
	registryMu.RLock()
	out := make([]Kind, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	registryMu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Validate dispatches validation to the registered type for kind.
// It returns ErrUnknownKind (wrapped) if kind is not registered.
func Validate(kind Kind, value string) (bool, error) {
	d, ok := Get(kind)
	if !ok {
		return false, fmt.Errorf("%q: %w", kind, ErrUnknownKind)
	}
	return d.Validate(value), nil
}

// Generate dispatches generation to the registered type for kind.
func Generate(kind Kind) (string, error) {
	d, ok := Get(kind)
	if !ok {
		return "", fmt.Errorf("%q: %w", kind, ErrUnknownKind)
	}
	return d.Generate(), nil
}

// Format dispatches formatting to the registered type for kind.
func Format(kind Kind, value string) (string, error) {
	d, ok := Get(kind)
	if !ok {
		return "", fmt.Errorf("%q: %w", kind, ErrUnknownKind)
	}
	return d.Format(value)
}

// Detect attempts to identify the Kind of value by its cleaned length, then
// confirms with that type's Validate. It generalizes the legacy ValidateDocument
// auto-detect (CPF=11 digits, CNPJ=14 alphanumeric). Returns ok=false when no
// registered type both matches the length and validates.
func Detect(value string) (Kind, bool) {
	digits := onlyDigits(value)
	switch len(digits) {
	case CpfLength:
		if ok, _ := Validate(KindCPF, value); ok {
			return KindCPF, true
		}
	case CnpjLength:
		if ok, _ := Validate(KindCNPJ, value); ok {
			return KindCNPJ, true
		}
	}
	return "", false
}
```

> `onlyDigits` is the shared helper added in M0-8; `CpfLength`/`CnpjLength` already exist in `brdoc.go`. Detect's switch is intentionally narrow in M0 (CPF/CNPJ only) and is extended in later milestones as more types register. If M0-8 has not yet landed when you compile this, temporarily inline `len(value)` cleaning — but the prescribed order is M0-8 lands `onlyDigits` and rewires Detect's callers; here Detect already references `onlyDigits`, so run M0-8 Step that adds `onlyDigits` first if the build fails on the undefined symbol. (Recommended: implement M0-8's `onlyDigits` helper step before M0-4 Step 3, since both are in the same milestone.)

- [ ] **Step 4: Ensure `onlyDigits` exists, then run the test, expect PASS.** If `onlyDigits` is not yet defined, jump to M0-8 Step 1 (add `onlyDigits` to a shared `helpers.go`) and return. Then run:
```
go test -run TestRegistry ./...
```
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Commit.** Run:
```
git add registry.go registry_test.go
git commit -m "feat: add concurrency-safe registry with Validate/Generate/Format/Detect dispatch"
```

---

### Task M0-5: Brand metadata constants

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/meta.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/meta_test.go`

**Interfaces:**
- Consumes: (none)
- Produces (package-level string consts): `AppName`, `CLIUse`, `CLIShort`, `MCPServerName`

- [ ] **Step 1: Write failing test pinning the brand constants.** Create `meta_test.go`:
```go
package brdoc

import "testing"

func TestMetaConstants(t *testing.T) {
	if AppName == "" {
		t.Fatal("AppName must not be empty")
	}
	if CLIUse != "brdoc" {
		t.Fatalf("CLIUse = %q, want \"brdoc\"", CLIUse)
	}
	if MCPServerName == "" {
		t.Fatal("MCPServerName must not be empty")
	}
	if CLIShort == "" {
		t.Fatal("CLIShort must not be empty")
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL (compile error).** Run:
```
go test -run TestMetaConstants ./...
```
Expected FAIL: `undefined: AppName`.

- [ ] **Step 3: Create `meta.go`.** Write:
```go
package brdoc

// Brand-bearing strings live here in one place so a future rebrand
// (/branding:names) is a near-mechanical change. The Go package name stays
// "brdoc" (a domain term, not a brand).
const (
	// AppName is the human-facing application name.
	AppName = "brdoc"
	// CLIUse is the Cobra root command Use field (the binary name).
	CLIUse = "brdoc"
	// CLIShort is the Cobra root command short description.
	CLIShort = "Brazilian documents utilities (CPF/CNPJ and more)"
	// MCPServerName is the MCP server Implementation Name.
	MCPServerName = "brdoc"
)
```

- [ ] **Step 4: Run the test, expect PASS.** Run:
```
go test -run TestMetaConstants ./...
```
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Commit.** Run:
```
git add meta.go meta_test.go
git commit -m "feat: isolate brand metadata constants in meta.go for future rebrand"
```

---

### Task M0-6: Migrate CPF onto the pattern (Document + OriginResolver + self-register + math/rand/v2)

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/brdoc.go` (imports + `init()` lines 41-52; `Generate` line 69-86; add `Kind`/`Origin` methods + `init()` registration)
- Keep green: `D:/weaver-sync/development/personal/projects/brdoc/brdoc_test.go` (UNCHANGED — all existing tests must still pass)
- Create: `D:/weaver-sync/development/personal/projects/brdoc/cpf_registry_test.go`

**Interfaces:**
- Consumes: `Kind`, `KindCPF`, `Document`, `OriginResolver` (M0-1); `Register` (M0-4)
- Produces:
  - `func (c *CPF) Kind() Kind` → returns `KindCPF`
  - `func (c *CPF) Origin(value string) (string, error)` (wraps existing `CheckOrigin`)
  - `*CPF` now satisfies `Document` and `OriginResolver`; registered in `init()` as `Register(&CPF{})`
- Behavior unchanged: `Generate`, `Validate`, `Format`, `CheckOrigin` keep identical semantics; only the RNG source changes from seeded `*rand.Rand` to `math/rand/v2`.

> CURRENT CODE (from `brdoc.go`, read verbatim):
> - imports include `"math/rand"` and `"time"`.
> - lines 28-31: `var ( notAcceptedCPF []string; rng *rand.Rand )`
> - lines 41-52 `init()`: seeds `rng = rand.New(rand.NewSource(time.Now().UnixNano()))` then fills `notAcceptedCPF`.
> - `Generate` (69-86) uses `rng.Intn(10)`.
> - `CNPJ.generateDigits` (367-383) also uses `rng.Intn(...)` — that call site is migrated in M0-7; in THIS task only switch the variable and CPF call sites, leaving CNPJ working against the new RNG too (math/rand/v2 has the same `IntN` semantics).

- [ ] **Step 1: Write failing test that CPF self-registers and satisfies the interfaces.** Create `cpf_registry_test.go`:
```go
package brdoc

import "testing"

func TestCPF_Registered(t *testing.T) {
	d, ok := Get(KindCPF)
	if !ok {
		t.Fatal("CPF is not registered in the registry")
	}
	if d.Kind() != KindCPF {
		t.Fatalf("registered CPF Kind = %q, want %q", d.Kind(), KindCPF)
	}
}

func TestCPF_SatisfiesInterfaces(t *testing.T) {
	var _ Document = (*CPF)(nil)
	var _ OriginResolver = (*CPF)(nil)
}

func TestCPF_Origin(t *testing.T) {
	c := NewCPF()
	got, err := c.Origin("123.456.789-09")
	if err != nil {
		t.Fatalf("Origin returned error: %v", err)
	}
	if got != IsDigit9 {
		t.Fatalf("Origin = %q, want %q", got, IsDigit9)
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL.** Run:
```
go test -run "TestCPF_Registered|TestCPF_SatisfiesInterfaces|TestCPF_Origin" ./...
```
Expected FAIL: `(*CPF)(nil) (variable of type *CPF) does not implement Document (missing method Kind)` AND/OR `undefined: c.Origin`, plus `Get(KindCPF)` returns ok=false.

- [ ] **Step 3: Swap the RNG to math/rand/v2 (imports + vars + init).** In `brdoc.go`:

BEFORE (imports, lines 3-10):
```go
import (
	"fmt"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"time"
)
```
AFTER:
```go
import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
)
```

BEFORE (vars, lines 28-31):
```go
var (
	notAcceptedCPF []string
	rng            *rand.Rand
)
```
AFTER:
```go
var notAcceptedCPF []string
```

BEFORE (`init`, lines 41-52):
```go
func init() {
	// Initialize random number generator
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	// Initialize non-accepted CPFs (all digits equal)
	notAcceptedCPF = make([]string, 0, 10)

	for i := range 10 {
		value := strings.Repeat(strconv.Itoa(i), 11)
		notAcceptedCPF = append(notAcceptedCPF, value)
	}
}
```
AFTER:
```go
func init() {
	// Initialize non-accepted CPFs (all digits equal)
	notAcceptedCPF = make([]string, 0, 10)

	for i := range 10 {
		value := strings.Repeat(strconv.Itoa(i), 11)
		notAcceptedCPF = append(notAcceptedCPF, value)
	}

	// Self-register CPF as a Document/OriginResolver singleton.
	Register(&CPF{})
}
```

- [ ] **Step 4: Update CPF.Generate and CNPJ.generateDigits RNG call sites.** `math/rand/v2` uses `rand.IntN` (capital N), not `rand.Intn`.

In `CPF.Generate` (lines 72-74) BEFORE:
```go
	for i := range 9 {
		number[i] = rng.Intn(10)
	}
```
AFTER:
```go
	for i := range 9 {
		number[i] = rand.IntN(10)
	}
```

In `CNPJ.generateDigits` (lines 371-383) BEFORE:
```go
	if legacy {
		for i := range 12 {
			base[i] = byte('0' + rng.Intn(10))
		}
	} else {
		for i := range 12 {
			if rng.Intn(2) == 0 {
				base[i] = byte('0' + rng.Intn(10))
			} else {
				base[i] = byte('A' + rng.Intn(26))
			}
		}
	}
```
AFTER:
```go
	if legacy {
		for i := range 12 {
			base[i] = byte('0' + rand.IntN(10))
		}
	} else {
		for i := range 12 {
			if rand.IntN(2) == 0 {
				base[i] = byte('0' + rand.IntN(10))
			} else {
				base[i] = byte('A' + rand.IntN(26))
			}
		}
	}
```

- [ ] **Step 5: Add Kind() and Origin() to CPF.** Add after `CheckOrigin` (after line 143) in `brdoc.go`:
```go
// Kind returns KindCPF, identifying this type in the registry.
func (c *CPF) Kind() Kind { return KindCPF }

// Origin returns the issuing region for value, satisfying OriginResolver.
// It wraps CheckOrigin; an empty origin (value too short) yields ErrInvalidLength.
func (c *CPF) Origin(value string) (string, error) {
	origin := c.CheckOrigin(value)
	if origin == "" {
		return "", ErrInvalidLength
	}
	return origin, nil
}
```

- [ ] **Step 6: Run the new tests, expect PASS.** Run:
```
go test -run "TestCPF_Registered|TestCPF_SatisfiesInterfaces|TestCPF_Origin" ./...
```
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 7: Run the FULL existing suite to prove zero behavior change, expect PASS.** Run:
```
go test ./...
```
Expected PASS: all of `TestCPF_Generate`, `TestCPF_Validate`, `TestCPF_Format`, `TestCPF_CheckOrigin`, `TestCPF_Validate_ProvidedValid`, `TestCPF_Format_ProvidedValid_Unformatted`, and the CNPJ tests, still green. `ok  github.com/inovacc/brdoc`.

- [ ] **Step 8: Commit.** Run:
```
git add brdoc.go cpf_registry_test.go
git commit -m "refactor: migrate CPF onto Document/OriginResolver registry, swap to math/rand/v2"
```

---

### Task M0-7: Migrate CNPJ onto the pattern (Document + self-register + regression pin)

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/brdoc.go` (add `CNPJ.Kind`; register `&CNPJ{}` in `init()` — extend the `init()` edited in M0-6)
- Keep green: `D:/weaver-sync/development/personal/projects/brdoc/brdoc_test.go` (UNCHANGED)
- Create: `D:/weaver-sync/development/personal/projects/brdoc/cnpj_registry_test.go`

**Interfaces:**
- Consumes: `Kind`, `KindCNPJ`, `Document` (M0-1); `Register` (M0-4)
- Produces:
  - `func (c *CNPJ) Kind() Kind` → returns `KindCNPJ`
  - `*CNPJ` satisfies `Document` (Validate/Generate/Format already exist; `GenerateLegacy` retained as an extra method); registered in `init()`.
- Behavior unchanged: `Validate`, `Generate`, `GenerateLegacy`, `Format`, `calculateDV` keep identical semantics.

> The CNPJ regression sample `39591842000010` is a valid legacy numeric CNPJ that paemuri (#26/#27) once falsely rejected. The local alphanumeric SERPRO validator computes DV via `charToValue` + weights `2..9` repeating; `39591842000010` has base `395918420000` with check digits `10` and MUST validate true.

- [ ] **Step 1: Write failing test: CNPJ self-registers, satisfies Document, and the regression sample validates.** Create `cnpj_registry_test.go`:
```go
package brdoc

import "testing"

func TestCNPJ_Registered(t *testing.T) {
	d, ok := Get(KindCNPJ)
	if !ok {
		t.Fatal("CNPJ is not registered in the registry")
	}
	if d.Kind() != KindCNPJ {
		t.Fatalf("registered CNPJ Kind = %q, want %q", d.Kind(), KindCNPJ)
	}
}

func TestCNPJ_SatisfiesDocument(t *testing.T) {
	var _ Document = (*CNPJ)(nil)
}

func TestCNPJ_RegressionSample(t *testing.T) {
	// paemuri #26/#27: this valid legacy numeric CNPJ was once falsely rejected.
	c := NewCNPJ()
	const sample = "39591842000010"
	if !c.Validate(sample) {
		t.Fatalf("regression: CNPJ %q must validate true", sample)
	}
	formatted, err := c.Format(sample)
	if err != nil {
		t.Fatalf("Format(%q) error: %v", sample, err)
	}
	if formatted != "39.591.842/0001-0" && formatted != "39.591.842/0001-10" {
		// Mask is XX.XXX.XXX/XXXX-XX → 18 chars.
		if formatted != "39.591.842/0001-10" {
			t.Fatalf("Format(%q) = %q, want \"39.591.842/0001-10\"", sample, formatted)
		}
	}
	if !c.Validate(formatted) {
		t.Fatalf("formatted regression CNPJ %q must validate true", formatted)
	}
}
```

> NOTE on the expected mask: the existing `CNPJ.Format` (lines 343-363) builds `XX.XXX.XXX/XXXX-XX`. For `39591842000010` that is `39.591.842/0001-10`. The test above asserts exactly that; remove the defensive double-branch when you confirm the value (keep only the single `!= "39.591.842/0001-10"` check).

- [ ] **Step 2: Run the test, expect FAIL.** Run:
```
go test -run "TestCNPJ_Registered|TestCNPJ_SatisfiesDocument|TestCNPJ_RegressionSample" ./...
```
Expected FAIL: `(*CNPJ)(nil) ... does not implement Document (missing method Kind)` and `Get(KindCNPJ)` ok=false. (`TestCNPJ_RegressionSample` may already pass on validation but the package won't compile until `Kind` exists.)

- [ ] **Step 3: Add CNPJ.Kind().** Add after `CNPJ.Format` (after line 363) in `brdoc.go`:
```go
// Kind returns KindCNPJ, identifying this type in the registry.
func (c *CNPJ) Kind() Kind { return KindCNPJ }
```

- [ ] **Step 4: Register CNPJ in init().** In `brdoc.go` `init()` (as edited in M0-6), BEFORE:
```go
	// Self-register CPF as a Document/OriginResolver singleton.
	Register(&CPF{})
}
```
AFTER:
```go
	// Self-register CPF and CNPJ as Document singletons.
	Register(&CPF{})
	Register(&CNPJ{})
}
```

- [ ] **Step 5: Tidy the regression test to a single assertion.** Replace the body of `TestCNPJ_RegressionSample` with the confirmed expectation:
```go
func TestCNPJ_RegressionSample(t *testing.T) {
	c := NewCNPJ()
	const sample = "39591842000010"
	if !c.Validate(sample) {
		t.Fatalf("regression: CNPJ %q must validate true", sample)
	}
	formatted, err := c.Format(sample)
	if err != nil {
		t.Fatalf("Format(%q) error: %v", sample, err)
	}
	const wantMask = "39.591.842/0001-10"
	if formatted != wantMask {
		t.Fatalf("Format(%q) = %q, want %q", sample, formatted, wantMask)
	}
	if !c.Validate(formatted) {
		t.Fatalf("formatted regression CNPJ %q must validate true", formatted)
	}
}
```

- [ ] **Step 6: Run the new tests, expect PASS.** Run:
```
go test -run "TestCNPJ_Registered|TestCNPJ_SatisfiesDocument|TestCNPJ_RegressionSample" ./...
```
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 7: Run the FULL suite, expect PASS (zero behavior change).** Run:
```
go test ./...
```
Expected PASS: all CPF + CNPJ existing tests (`TestCNPJ_Validate`, `TestCNPJ_Generate`, `TestCNPJ_GenerateLegacy`, `TestCNPJ_Format`, `TestCNPJ_CalculateDV_Manual`, `TestCNPJ_Validate_ProvidedValid`, etc.) green. `ok  github.com/inovacc/brdoc`.

- [ ] **Step 8: Commit.** Run:
```
git add brdoc.go cnpj_registry_test.go
git commit -m "refactor: migrate CNPJ onto Document registry; pin 39591842000010 regression"
```

---

### Task M0-8: Shared onlyDigits helper & deprecate ValidateDocument

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/helpers.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/helpers_test.go`
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/brdoc.go` (`ValidateDocument` lines 484-501 → deprecated thin wrapper over `Detect`+`Validate`)
- Keep green: `D:/weaver-sync/development/personal/projects/brdoc/brdoc_test.go` (`TestValidateDocument` UNCHANGED — must still pass)

**Interfaces:**
- Consumes: `Detect` (M0-4), `Validate` (M0-4), `KindCPF`, `KindCNPJ` (M0-1)
- Produces:
  - `func onlyDigits(s string) string` (shared, unexported; used by `Detect` and all later types)
  - `ValidateDocument(doc string) (docType string, isValid bool)` retained, now `// Deprecated:` and implemented over `Detect`+`Validate`.

> EXISTING `ValidateDocument` (brdoc.go lines 484-501) returns `("CPF", ...)`, `("CNPJ", ...)`, or `("UNKNOWN", false)`. The deprecated wrapper MUST preserve those exact return strings so `TestValidateDocument` (brdoc_test.go lines 367-389) stays green: it asserts `"CPF"`, `"CNPJ"`, and `"UNKNOWN"`.

- [ ] **Step 1: Write failing test for onlyDigits.** Create `helpers_test.go`:
```go
package brdoc

import "testing"

func TestOnlyDigits(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"123.456.789-09", "12345678909"},
		{"12.ABC.345/01DE-35", "1234501"},
		{"  +55 (11) 98888-7777 ", "5511988887777"},
		{"abc", ""},
		{"", ""},
		{"0001-10", "000110"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := onlyDigits(tt.in); got != tt.want {
				t.Fatalf("onlyDigits(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL (compile error).** Run:
```
go test -run TestOnlyDigits ./...
```
Expected FAIL: `undefined: onlyDigits`.

- [ ] **Step 3: Create `helpers.go`.** Write:
```go
package brdoc

import "strings"

// onlyDigits returns s with all non-digit bytes removed. It is the shared
// digit-cleaning helper used by Detect and by every document type's parsing.
func onlyDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= '0' && c <= '9' {
			b.WriteByte(c)
		}
	}
	return b.String()
}
```

- [ ] **Step 4: Run the test, expect PASS.** Run:
```
go test -run TestOnlyDigits ./...
```
Expected PASS: `ok  github.com/inovacc/brdoc`. (This also unblocks `registry.go` `Detect`, which references `onlyDigits`.)

- [ ] **Step 5: Write failing test that ValidateDocument is a Detect-backed wrapper.** Append to `helpers_test.go`:
```go
func TestValidateDocument_DelegatesToDetect(t *testing.T) {
	// Wrapper must agree with Detect+Validate for known kinds.
	if dt, ok := ValidateDocument("123.456.789-09"); dt != "CPF" || !ok {
		t.Fatalf("ValidateDocument(valid CPF) = (%q, %v), want (CPF, true)", dt, ok)
	}
	if dt, ok := ValidateDocument("12.ABC.345/01DE-35"); dt != "CNPJ" || !ok {
		t.Fatalf("ValidateDocument(valid CNPJ) = (%q, %v), want (CNPJ, true)", dt, ok)
	}
	// Invalid-but-CPF-shaped input keeps the CPF label (parity with legacy).
	if dt, ok := ValidateDocument("123.456.789-00"); dt != "CPF" || ok {
		t.Fatalf("ValidateDocument(invalid CPF) = (%q, %v), want (CPF, false)", dt, ok)
	}
	if dt, ok := ValidateDocument("12345"); dt != "UNKNOWN" || ok {
		t.Fatalf("ValidateDocument(garbage) = (%q, %v), want (UNKNOWN, false)", dt, ok)
	}
}
```

> Parity note: legacy `ValidateDocument` labels by LENGTH first (11→"CPF", 14→"CNPJ") and only then validates, so an invalid-but-11-digit string returns `("CPF", false)`. `Detect` returns ok=false for invalid input and gives no kind, so the wrapper cannot rely on `Detect` alone for the label — it must reproduce the length-based labelling. The implementation below does exactly that to preserve behavior.

- [ ] **Step 6: Run the new test, expect FAIL.** Run:
```
go test -run TestValidateDocument_DelegatesToDetect ./...
```
Expected FAIL on the invalid-CPF case if you naively delegate to `Detect` (which would return `"UNKNOWN"` for invalid input). This drives the length-aware implementation in Step 7.

- [ ] **Step 7: Rewrite ValidateDocument as a deprecated, length-labelling wrapper.** In `brdoc.go`, BEFORE (lines 484-501):
```go
// ValidateDocument automatically identifies and validates CPF or CNPJ
func ValidateDocument(doc string) (docType string, isValid bool) {
	cleaned := strings.ReplaceAll(doc, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "/", "")
	cleaned = strings.ToUpper(cleaned)

	// Identifica pelo tamanho
	if len(cleaned) == CpfLength {
		cpf := NewCPF()
		return "CPF", cpf.Validate(doc)
	} else if len(cleaned) == CnpjLength {
		cnpj := NewCNPJ()
		return "CNPJ", cnpj.Validate(doc)
	}

	return "UNKNOWN", false
}
```
AFTER:
```go
// ValidateDocument automatically identifies and validates a CPF or CNPJ.
//
// Deprecated: use Detect to identify the Kind and Validate(kind, value) to
// validate, e.g. k, ok := brdoc.Detect(s); the typed Kind ("cpf"/"cnpj") is
// richer than the legacy "CPF"/"CNPJ"/"UNKNOWN" strings. This wrapper will be
// removed after 2026-07-18.
func ValidateDocument(doc string) (docType string, isValid bool) {
	// Preserve legacy length-based labelling: an 11/14-length value keeps its
	// label even when invalid (parity with the original implementation).
	switch len(onlyDigits(doc)) {
	case CpfLength:
		ok, _ := Validate(KindCPF, doc)
		return "CPF", ok
	case CnpjLength:
		ok, _ := Validate(KindCNPJ, doc)
		return "CNPJ", ok
	default:
		return "UNKNOWN", false
	}
}
```

> The original cleaned alphanumeric CNPJ (`12.ABC.345/01DE-35`) has 7 digits via `onlyDigits` — NOT 14. The legacy code used `strings.ReplaceAll` (only `.`,`-`,`/`) + `ToUpper`, so `12ABC34501DE35` counted as length 14. To preserve EXACT parity for alphanumeric CNPJ, the wrapper must clean the same way the legacy did for the length test. Adjust the switch to use a CNPJ-aware cleaner: reuse the existing private `(&CNPJ{}).digits(doc)` (which keeps 0-9 and A-Z, length 14 for valid CNPJ). Use the corrected body in Step 8.

- [ ] **Step 8: Correct the wrapper to preserve alphanumeric-CNPJ parity.** Replace the wrapper body so length detection matches the legacy alphanumeric behavior:
```go
func ValidateDocument(doc string) (docType string, isValid bool) {
	// CPF length is measured over digits only; CNPJ length over the
	// alphanumeric (0-9, A-Z) cleaned form, matching the legacy semantics.
	if len(onlyDigits(doc)) == CpfLength {
		ok, _ := Validate(KindCPF, doc)
		return "CPF", ok
	}
	if len((&CNPJ{}).digits(doc)) == CnpjLength {
		ok, _ := Validate(KindCNPJ, doc)
		return "CNPJ", ok
	}
	return "UNKNOWN", false
}
```
Keep the `// Deprecated:` doc comment from Step 7 above this function.

> `(&CNPJ{}).digits` is the existing private method (brdoc.go lines 456-478) that uppercases and keeps `[0-9A-Z]`. For `12.ABC.345/01DE-35` it returns the 14-char `12ABC34501DE35`, so the CNPJ branch is reached exactly as before. For a 14-digit numeric CNPJ both `onlyDigits` (=14, not 11) and the CNPJ cleaner agree; the CPF branch (==11) is checked first and won't false-trigger.

- [ ] **Step 9: If `strings` becomes unused in brdoc.go, keep it — it is still used.** `strings` is still referenced by `strings.Repeat` in `init()` and `strings.Builder` in `Generate`. No import change needed. Verify the package builds:
```
go build ./...
```
Expected: no output (success).

- [ ] **Step 10: Run the targeted tests, expect PASS.** Run:
```
go test -run "TestOnlyDigits|TestValidateDocument" ./...
```
Expected PASS: both `TestOnlyDigits`, `TestValidateDocument_DelegatesToDetect`, and the existing `TestValidateDocument` (brdoc_test.go) green. `ok  github.com/inovacc/brdoc`.

- [ ] **Step 11: Run the FULL suite + vet, expect PASS.** Run:
```
go vet ./... && go test ./...
```
Expected PASS: entire suite green; `go vet` reports nothing. (The `// Deprecated:` comment is informational; vet does not fail on self-use within the package, but avoid calling `ValidateDocument` from new code.)

- [ ] **Step 12: Commit.** Run:
```
git add helpers.go helpers_test.go brdoc.go
git commit -m "refactor: add onlyDigits helper; deprecate ValidateDocument over Detect+Validate"
```

---

> **M0 exit check (run before starting M1):**
> ```
> go vet ./... && go test ./...
> ```
> Expected: all tests green, including every original `brdoc_test.go` case (zero behavior change), the registry/UF/errors/meta tests, the CPF/CNPJ migration tests, and the `39591842000010` regression. The frozen contract (`document.go`, `errors.go`, `uf.go`, `registry.go`, `meta.go`, `helpers.go`) is now the reference for all later milestones.


---

## Milestone M1 — CLI engine

Goal: rewrite `cmd/brdoc/main.go` so every Cobra subcommand is **derived by iterating `brdoc.Kinds()`** — adding a type to the registry automatically yields its `<kind>` subcommand with no per-type boilerplate. Per-kind flags: `-g/--generate`, `-v/--validate VALUE`, `--format VALUE`, `--origin VALUE` (only when the type implements `OriginResolver`), `-f/--from FILE|-`, `-n/--count N`, and `--uf` (only for RG / `UFScoped` types). The existing `bufio.Scanner` bulk `--from` streaming (1 MB max line, file or stdin via `-`) is extracted into a shared helper and reused by every kind. Top-level commands: `brdoc detect <value>` and `brdoc version`. Current UX is preserved exactly: `SilenceUsage`/`SilenceErrors`, exit code 1 on invalid input, `valid\t<formatted>` / `invalid\t<value>` output lines.

In M1 the registry contains only CPF and CNPJ (registered in M0). CPF implements `OriginResolver` (so `cpf` gets `--origin`); CNPJ does not (so `cnpj` has no `--origin`). No type implements `UFScoped` yet, so `--uf` is wired but inert until M2 ships RG. The CNPJ `--legacy` flag from the old CLI is dropped from the generic engine; `cnpj --generate` emits formatted alphanumeric CNPJs via `brdoc.Generate(KindCNPJ)` + `brdoc.Format`. This is an intentional UX simplification recorded in M5 docs.

Files touched in this milestone:
- `cmd/brdoc/iohelper.go` (new) — shared bulk `--from` reader + streaming validator.
- `cmd/brdoc/iohelper_test.go` (new).
- `cmd/brdoc/version.go` (new) — `version` subcommand + build-info version string.
- `cmd/brdoc/detect.go` (new) — `detect` subcommand.
- `cmd/brdoc/detect_test.go` (new).
- `cmd/brdoc/kindcmd.go` (new) — registry-driven per-kind command factory.
- `cmd/brdoc/kindcmd_test.go` (new).
- `cmd/brdoc/main.go` (rewrite) — slim root + wiring.
- `cmd/brdoc/main_test.go` (new) — end-to-end command tests.

---

### Task M1-1: Shared bulk `--from` streaming helper

**Files:**
- Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/iohelper.go`
- Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/iohelper_test.go`

**Interfaces:**
- Consumes: nothing from the registry (pure I/O). Standard library only.
- Produces:
  - `const maxLine = 1024 * 1024`
  - `func openReader(path string) (io.Reader, func(), error)` — `"-"` → `os.Stdin` (nil close fn); otherwise `filepath.Abs` then `os.Open`, returning a close fn that ignores the close error.
  - `type lineValidator func(value string) (formatted string, valid bool)`
  - `func streamValidate(r io.Reader, w io.Writer, fn lineValidator) (anyInvalid bool, err error)` — scans lines, trims whitespace, skips blank lines and `#`-prefixed comments, writes `valid\t<formatted>` (or bare `valid` when `formatted == ""`) / `invalid\t<value>`, returns whether any line was invalid.

- [ ] **Step 1: Write failing test for `openReader` and `streamValidate`.** Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/iohelper_test.go`:
```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenReaderStdin(t *testing.T) {
	r, closeFn, err := openReader("-")
	require.NoError(t, err)
	assert.Nil(t, closeFn)
	assert.Equal(t, os.Stdin, r)
}

func TestOpenReaderFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docs.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))

	r, closeFn, err := openReader(path)
	require.NoError(t, err)
	require.NotNil(t, closeFn)
	defer closeFn()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	assert.Equal(t, "hello\n", buf.String())
}

func TestOpenReaderMissingFile(t *testing.T) {
	_, _, err := openReader(filepath.Join(t.TempDir(), "nope.txt"))
	assert.Error(t, err)
}

func TestStreamValidate(t *testing.T) {
	in := strings.NewReader("# comment\n\n111\n222\n   333   \n")
	out := new(bytes.Buffer)

	// Treat "222" as the only invalid line; format doubles the value.
	fn := func(value string) (string, bool) {
		if value == "222" {
			return "", false
		}
		return value + value, true
	}

	anyInvalid, err := streamValidate(in, out, fn)
	require.NoError(t, err)
	assert.True(t, anyInvalid)
	assert.Equal(t, "valid\t111111\ninvalid\t222\nvalid\t333333\n", out.String())
}

func TestStreamValidateBareValid(t *testing.T) {
	in := strings.NewReader("abc\n")
	out := new(bytes.Buffer)
	fn := func(value string) (string, bool) { return "", true }

	anyInvalid, err := streamValidate(in, out, fn)
	require.NoError(t, err)
	assert.False(t, anyInvalid)
	assert.Equal(t, "valid\n", out.String())
}
```
- [ ] **Step 2: Run the test — expect compile failure.** Run `go test ./cmd/brdoc/ -run 'TestOpenReader|TestStreamValidate'`. Expect FAIL: `undefined: openReader` / `undefined: streamValidate`.
- [ ] **Step 3: Implement the helper.** Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/iohelper.go`:
```go
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxLine = 1024 * 1024

// scratch buffer reused by bufio.Scanner for long lines.
var scanBuf = make([]byte, 0, 64*1024)

// lineValidator validates a single trimmed line. It returns the formatted
// representation (empty string when no mask applies) and whether it is valid.
type lineValidator func(value string) (formatted string, valid bool)

// openReader returns an io.Reader for the given path. If path is "-" it returns
// stdin and a nil close function; otherwise it opens the absolute path and
// returns a close function that ignores the close error.
func openReader(path string) (io.Reader, func(), error) {
	if path == "-" {
		return os.Stdin, nil, nil
	}

	fullPath, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, err
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, err
	}

	return f, func() { _ = f.Close() }, nil
}

// streamValidate scans r line by line (max 1 MB per line), trims whitespace,
// skips blank lines and '#'-prefixed comments, and writes one result line per
// input: "valid\t<formatted>" (or bare "valid" when formatted is empty) for
// valid values and "invalid\t<value>" otherwise. It returns whether any line
// was invalid and any scanner error.
func streamValidate(r io.Reader, w io.Writer, fn lineValidator) (bool, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(scanBuf, maxLine)

	bw := bufio.NewWriter(w)
	defer func() { _ = bw.Flush() }()

	anyInvalid := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		formatted, valid := fn(line)
		switch {
		case valid && formatted != "":
			_, _ = fmt.Fprintf(bw, "valid\t%s\n", formatted)
		case valid:
			_, _ = fmt.Fprintln(bw, "valid")
		default:
			anyInvalid = true
			_, _ = fmt.Fprintf(bw, "invalid\t%s\n", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return anyInvalid, err
	}

	return anyInvalid, nil
}
```
- [ ] **Step 4: Run the test — expect PASS.** Run `go test ./cmd/brdoc/ -run 'TestOpenReader|TestStreamValidate'`. Expect `ok  github.com/inovacc/brdoc/cmd/brdoc`.
- [ ] **Step 5: Commit.**
```
git add cmd/brdoc/iohelper.go cmd/brdoc/iohelper_test.go
git commit -m "feat(cli): extract shared bulk --from streaming helper"
```

---

### Task M1-2: Registry-driven per-kind command factory

**Files:**
- Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/kindcmd.go`
- Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/kindcmd_test.go`

**Interfaces:**
- Consumes (frozen contract, root package `brdoc`):
  - `func Kinds() []Kind`
  - `func Get(kind Kind) (Document, bool)`
  - `type Document interface { Kind() Kind; Validate(value string) bool; Generate() string; Format(value string) (string, error) }`
  - `type OriginResolver interface { Origin(value string) (string, error) }`
  - `type UFScoped interface { ValidateUF(value string, uf UF) (bool, error); ImplementedUFs() []UF }`
  - `type Kind string`, `func (k Kind) String() string`
  - `type UF string`
  - `func NewCNPJ() *CNPJ`, `func (c *CNPJ) Generate() string`, `func (c *CNPJ) Format(value string) (string, error)`
  - From M1-1: `func openReader(path string) (io.Reader, func(), error)`, `func streamValidate(r io.Reader, w io.Writer, fn lineValidator) (bool, error)`.
- Produces:
  - `func newKindCmd(kind brdoc.Kind) *cobra.Command` — builds the `<kind>` subcommand with flags wired per the kind's capabilities.
  - `func registerKindCommands(root *cobra.Command)` — iterates `brdoc.Kinds()` and adds one command per kind.

- [ ] **Step 1: Write failing tests for the factory.** Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/kindcmd_test.go`:
```go
package main

import (
	"bytes"
	"strings"
	"testing"

	sdk "github.com/inovacc/brdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestKindCmdGenerateCPF(t *testing.T) {
	out, err := runCmd(t, "cpf", "--generate")
	require.NoError(t, err)
	got := strings.TrimSpace(out)
	assert.True(t, sdk.NewCPF().Validate(got), "generated CPF must validate: %q", got)
}

func TestKindCmdGenerateCount(t *testing.T) {
	out, err := runCmd(t, "cpf", "--generate", "--count", "3")
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 3)
}

func TestKindCmdValidateValidCPF(t *testing.T) {
	// 529.982.247-25 is a well-known valid CPF.
	out, err := runCmd(t, "cpf", "--validate", "529.982.247-25")
	require.NoError(t, err)
	assert.Equal(t, "valid\t529.982.247-25\n", out)
}

func TestKindCmdValidateInvalidCPF(t *testing.T) {
	out, err := runCmd(t, "cpf", "--validate", "123.456.789-00")
	require.NoError(t, err) // exit handled in main(); RunE returns nil
	assert.Equal(t, "invalid\n", out)
}

func TestKindCmdFormatCNPJ(t *testing.T) {
	// 39591842000010 is the paemuri regression sample (valid CNPJ).
	out, err := runCmd(t, "cnpj", "--format", "39591842000010")
	require.NoError(t, err)
	assert.Equal(t, "39.591.842/0001-10\n", out)
}

func TestKindCmdOriginOnlyForResolver(t *testing.T) {
	cpf := newKindCmd(sdk.KindCPF)
	assert.NotNil(t, cpf.Flags().Lookup("origin"), "cpf must expose --origin (OriginResolver)")

	cnpj := newKindCmd(sdk.KindCNPJ)
	assert.Nil(t, cnpj.Flags().Lookup("origin"), "cnpj must NOT expose --origin")
}

func TestKindCmdUFOnlyForUFScoped(t *testing.T) {
	cpf := newKindCmd(sdk.KindCPF)
	assert.Nil(t, cpf.Flags().Lookup("uf"), "cpf must NOT expose --uf")
}

func TestKindCmdOriginCPF(t *testing.T) {
	out, err := runCmd(t, "cpf", "--origin", "529.982.247-25")
	require.NoError(t, err)
	assert.NotEmpty(t, strings.TrimSpace(out))
}

func TestKindCmdNoFlags(t *testing.T) {
	_, err := runCmd(t, "cpf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either")
}

func TestKindCmdGenerateConflictsValidate(t *testing.T) {
	_, err := runCmd(t, "cpf", "--generate", "--validate", "529.982.247-25")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used with")
}

func TestRegisterKindCommands(t *testing.T) {
	root := newRootCmd()
	for _, k := range sdk.Kinds() {
		assert.NotNil(t, findCmd(root, k.String()), "missing subcommand for %s", k)
	}
}

func findCmd(root *cobra.Command, name string) *cobra.Command {
	for _, c := range root.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
```
Note: `findCmd` references `cobra`; add the import in the test file:
```go
import "github.com/spf13/cobra"
```
(Place it in the import block above alongside the others.)
- [ ] **Step 2: Run the test — expect compile failure.** Run `go test ./cmd/brdoc/ -run 'TestKindCmd|TestRegisterKindCommands'`. Expect FAIL: `undefined: newRootCmd` / `undefined: newKindCmd` / `undefined: registerKindCommands`.
- [ ] **Step 3: Implement the per-kind command factory.** Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/kindcmd.go`:
```go
package main

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/inovacc/brdoc"
	"github.com/spf13/cobra"
)

// kindFlags holds the bound flag values for one per-kind subcommand.
type kindFlags struct {
	generate bool
	validate string
	format   string
	origin   string
	from     string
	uf       string
	count    int
}

// newKindCmd builds the Cobra subcommand for a single registered document kind.
// Capability flags (--origin, --uf) are wired only when the underlying type
// implements OriginResolver / UFScoped respectively.
func newKindCmd(kind sdk.Kind) *cobra.Command {
	doc, ok := sdk.Get(kind)
	if !ok {
		// Defensive: registerKindCommands only iterates registered kinds.
		return &cobra.Command{Use: kind.String(), Hidden: true}
	}

	name := kind.String()
	upper := strings.ToUpper(name)
	f := &kindFlags{}

	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Generate, validate, or format %s", upper),
		Example: strings.Join([]string{
			fmt.Sprintf("brdoc %s --generate", name),
			fmt.Sprintf("brdoc %s --generate --count 10", name),
			fmt.Sprintf("brdoc %s --validate <value>", name),
			fmt.Sprintf("brdoc %s --format <value>", name),
			fmt.Sprintf("brdoc %s --from values.txt", name),
			fmt.Sprintf("type values.txt | brdoc %s --from -", name),
		}, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKind(cmd, doc, f)
		},
	}

	cmd.Flags().BoolVarP(&f.generate, "generate", "g", false, "Generate a valid "+upper)
	cmd.Flags().StringVarP(&f.validate, "validate", "v", "", "Validate a single "+upper+" value")
	cmd.Flags().StringVar(&f.format, "format", "", "Format a single "+upper+" value")
	cmd.Flags().StringVarP(&f.from, "from", "f", "", "Validate many values from file or '-' for stdin")
	cmd.Flags().IntVarP(&f.count, "count", "n", 0, "When generating, how many values to output")

	if _, ok := doc.(sdk.OriginResolver); ok {
		cmd.Flags().StringVar(&f.origin, "origin", "", "Resolve origin/region of a single "+upper+" value")
	}

	if _, ok := doc.(sdk.UFScoped); ok {
		cmd.Flags().StringVar(&f.uf, "uf", "", "Federative unit (e.g. SP) — required for "+upper)
	}

	return cmd
}

// registerKindCommands adds one subcommand per registered kind to root.
func registerKindCommands(root *cobra.Command) {
	for _, k := range sdk.Kinds() {
		root.AddCommand(newKindCmd(k))
	}
}

// runKind dispatches a single per-kind invocation based on the bound flags.
func runKind(cmd *cobra.Command, doc sdk.Document, f *kindFlags) error {
	if err := f.validateCombo(); err != nil {
		return err
	}

	switch {
	case f.generate:
		return runGenerate(cmd, doc, f.count)
	case f.from != "":
		return runFrom(cmd, doc, f.from)
	case f.format != "":
		return runFormat(cmd, doc, f.format)
	case f.origin != "":
		return runOrigin(cmd, doc, f.origin)
	default:
		return runValidate(cmd, doc, f.validate, f.uf)
	}
}

// validateCombo enforces mutually exclusive / required flag combinations,
// preserving the original CLI's error messages and UX.
func (f *kindFlags) validateCombo() error {
	actions := 0
	for _, on := range []bool{f.generate, f.validate != "", f.format != "", f.origin != "", f.from != ""} {
		if on {
			actions++
		}
	}

	if f.generate && (f.validate != "" || f.from != "" || f.format != "" || f.origin != "") {
		return errors.New("--generate cannot be used with --validate, --format, --origin, or --from")
	}

	if actions == 0 {
		return errors.New("either --generate, --validate, --format, --origin, or --from must be provided")
	}

	if f.from != "" && f.validate != "" {
		return errors.New("--from and --validate are mutually exclusive")
	}

	return nil
}

func runGenerate(cmd *cobra.Command, doc sdk.Document, count int) error {
	if count <= 0 {
		count = 1
	}

	w := bufio.NewWriter(cmd.OutOrStdout())
	defer func() { _ = w.Flush() }()

	for i := 0; i < count; i++ {
		value := doc.Generate()
		if formatted, err := doc.Format(value); err == nil {
			_, _ = fmt.Fprintln(w, formatted)
		} else {
			_, _ = fmt.Fprintln(w, value)
		}
	}

	return nil
}

func runFrom(cmd *cobra.Command, doc sdk.Document, from string) error {
	r, closeFn, err := openReader(from)
	if err != nil {
		return err
	}

	if closeFn != nil {
		defer closeFn()
	}

	fn := func(value string) (string, bool) {
		if !doc.Validate(value) {
			return "", false
		}

		formatted, ferr := doc.Format(value)
		if ferr != nil {
			return "", true
		}

		return formatted, true
	}

	anyInvalid, err := streamValidate(r, cmd.OutOrStdout(), fn)
	if err != nil {
		return err
	}

	if anyInvalid {
		cmd.SilenceUsage = true
		return errInvalidInput
	}

	return nil
}

func runFormat(cmd *cobra.Command, doc sdk.Document, value string) error {
	formatted, err := doc.Format(value)
	if err != nil {
		cmd.SilenceUsage = true
		return err
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), formatted)

	return nil
}

func runOrigin(cmd *cobra.Command, doc sdk.Document, value string) error {
	r, ok := doc.(sdk.OriginResolver)
	if !ok {
		return fmt.Errorf("--origin is not supported for %s", doc.Kind())
	}

	origin, err := r.Origin(value)
	if err != nil {
		cmd.SilenceUsage = true
		return err
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), origin)

	return nil
}

func runValidate(cmd *cobra.Command, doc sdk.Document, value, uf string) error {
	valid := docValidate(doc, value, uf)
	if !valid {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "invalid")
		cmd.SilenceUsage = true
		return errInvalidInput
	}

	if formatted, err := doc.Format(value); err == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "valid\t%s\n", formatted)
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "valid")
	}

	return nil
}

// docValidate runs UF-scoped validation when a --uf is supplied and the type
// supports it; otherwise it runs plain Validate.
func docValidate(doc sdk.Document, value, uf string) bool {
	if uf != "" {
		if s, ok := doc.(sdk.UFScoped); ok {
			ok2, err := s.ValidateUF(value, sdk.UF(strings.ToUpper(uf)))
			return err == nil && ok2
		}
	}

	return doc.Validate(value)
}
```
Note: `errInvalidInput` and `newRootCmd` are defined in Task M1-4 (`main.go`). To keep this task independently testable, add a temporary stub in `kindcmd.go` only if M1-4 has not landed; the canonical definitions live in `main.go`. In the final tree both come from M1-4 — do NOT duplicate them. For the TDD run in this task, temporarily add at the bottom of `kindcmd_test.go`:
```go
// Stubs satisfied by main.go (Task M1-4) once landed; remove when M1-4 lands.
```
(No code stub needed — implement M1-4 before running this task's tests if executing strictly in order. The recommended order is M1-1 → M1-4 → M1-2 → M1-3, but each task's code is complete.)
- [ ] **Step 4: Implement `main.go` essentials needed here are defined in M1-4.** If running tasks out of order, ensure `errInvalidInput` (`var errInvalidInput = errors.New("invalid input")`) and `func newRootCmd() *cobra.Command` exist from M1-4 before compiling.
- [ ] **Step 5: Run the test — expect PASS.** Run `go test ./cmd/brdoc/ -run 'TestKindCmd|TestRegisterKindCommands'`. Expect `ok`. All capability-flag assertions (`--origin` present for cpf, absent for cnpj; `--uf` absent for cpf) pass.
- [ ] **Step 6: Commit.**
```
git add cmd/brdoc/kindcmd.go cmd/brdoc/kindcmd_test.go
git commit -m "feat(cli): derive per-kind subcommands from registry"
```

---

### Task M1-3: `brdoc detect <value>` subcommand

**Files:**
- Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/detect.go`
- Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/detect_test.go`

**Interfaces:**
- Consumes (frozen contract):
  - `func Detect(value string) (Kind, bool)`
  - `func (k Kind) String() string`
  - From M1-2/M1-4: `var errInvalidInput`.
- Produces:
  - `func newDetectCmd() *cobra.Command` — `detect <value>` positional arg; prints the detected kind or reports unknown.

- [ ] **Step 1: Write failing test.** Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/detect_test.go`:
```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCPF(t *testing.T) {
	out, err := runCmd(t, "detect", "529.982.247-25")
	require.NoError(t, err)
	assert.Equal(t, "cpf\n", out)
}

func TestDetectCNPJ(t *testing.T) {
	out, err := runCmd(t, "detect", "39591842000010")
	require.NoError(t, err)
	assert.Equal(t, "cnpj\n", out)
}

func TestDetectUnknown(t *testing.T) {
	out, err := runCmd(t, "detect", "12345")
	require.Error(t, err)
	assert.Equal(t, "unknown\n", out)
}

func TestDetectRequiresArg(t *testing.T) {
	_, err := runCmd(t, "detect")
	require.Error(t, err)
}
```
- [ ] **Step 2: Run the test — expect failure.** Run `go test ./cmd/brdoc/ -run 'TestDetect'`. Expect FAIL: `unknown command "detect"` (cobra) once `newDetectCmd` is unwired, or compile failure `undefined: newDetectCmd` if referenced.
- [ ] **Step 3: Implement detect.** Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/detect.go`:
```go
package main

import (
	"fmt"

	sdk "github.com/inovacc/brdoc"
	"github.com/spf13/cobra"
)

// newDetectCmd builds the top-level "detect <value>" command, which reports the
// document kind inferred by the registry's length-based Detect.
func newDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detect <value>",
		Short: "Auto-detect the document kind of a value",
		Args:  cobra.ExactArgs(1),
		Example: "brdoc detect 529.982.247-25\nbrdoc detect 39.591.842/0001-10",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := sdk.Detect(args[0])
			if !ok {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "unknown")
				cmd.SilenceUsage = true
				return errInvalidInput
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), kind.String())

			return nil
		},
	}
}
```
- [ ] **Step 4: Wire it in.** Ensure `newRootCmd()` (Task M1-4) calls `root.AddCommand(newDetectCmd())`. If M1-4 already landed, confirm the line is present; otherwise it is added in M1-4 Step 3.
- [ ] **Step 5: Run the test — expect PASS.** Run `go test ./cmd/brdoc/ -run 'TestDetect'`. Expect `ok`.
- [ ] **Step 6: Commit.**
```
git add cmd/brdoc/detect.go cmd/brdoc/detect_test.go
git commit -m "feat(cli): add detect subcommand backed by registry Detect"
```

---

### Task M1-4: `brdoc version` + slim registry-wired root, exit-code preservation

**Files:**
- Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/version.go`
- Rewrite `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/main.go` (replace lines 23-344; keep the license header lines 1-21)
- Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/main_test.go`

**Interfaces:**
- Consumes (frozen contract):
  - `const CLIUse = "brdoc"`, `const CLIShort = "Brazilian documents utilities (CPF/CNPJ and more)"`, `const AppName = "brdoc"` (from `meta.go`, M0-5).
  - `func registerKindCommands(root *cobra.Command)` (M1-2), `func newDetectCmd() *cobra.Command` (M1-3).
- Produces:
  - `var errInvalidInput = errors.New("invalid input")` — sentinel returned by RunE on invalid input so `main()` exits 1 without printing usage.
  - `func newRootCmd() *cobra.Command` — builds the root with `SilenceUsage`/`SilenceErrors`, disabled default completion command, registry-driven subcommands, `detect`, and `version`.
  - `func newVersionCmd() *cobra.Command` — prints `brdoc <version>`.
  - `func version() string` — resolves the build version from `runtime/debug.ReadBuildInfo`, defaulting to `"dev"`.
  - `func main()` — executes the root; on error other than nothing, prints to stderr only when the error is not `errInvalidInput` (invalid input already printed "invalid"/"unknown"), then `os.Exit(1)`.

- [ ] **Step 1: Write failing test for version + root behavior.** Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/main_test.go`:
```go
package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	out, err := runCmd(t, "version")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, "brdoc "), "got %q", out)
	assert.NotEmpty(t, strings.TrimSpace(strings.TrimPrefix(out, "brdoc ")))
}

func TestVersionFuncDefault(t *testing.T) {
	assert.NotEmpty(t, version())
}

func TestRootHasNoCompletionCmd(t *testing.T) {
	root := newRootCmd()
	assert.Nil(t, findCmd(root, "completion"))
}

func TestRootSilences(t *testing.T) {
	root := newRootCmd()
	assert.True(t, root.SilenceUsage)
	assert.True(t, root.SilenceErrors)
}

func TestInvalidInputSentinel(t *testing.T) {
	_, err := runCmd(t, "cpf", "--validate", "000.000.000-00")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errInvalidInput))
}
```
- [ ] **Step 2: Run the test — expect failure.** Run `go test ./cmd/brdoc/ -run 'TestVersion|TestRoot|TestInvalidInputSentinel'`. Expect FAIL: `undefined: version` / `undefined: newRootCmd` / `undefined: errInvalidInput`.
- [ ] **Step 3: Implement version.** Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/version.go`:
```go
package main

import (
	"fmt"
	"runtime/debug"

	sdk "github.com/inovacc/brdoc"
	"github.com/spf13/cobra"
)

// version resolves the build version from the module build info, defaulting to
// "dev" for `go run` / un-versioned builds.
func version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}

	return "dev"
}

// newVersionCmd builds the top-level "version" command.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the " + sdk.AppName + " version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", sdk.AppName, version())
			return nil
		},
	}
}
```
- [ ] **Step 4: Rewrite `main.go`.** Replace the entire file body below the license header (keep lines 1-21 verbatim) so `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/main.go` reads:
```go
package main

import (
	"errors"
	"fmt"
	"os"

	sdk "github.com/inovacc/brdoc"
	"github.com/spf13/cobra"
)

// errInvalidInput is returned by RunE handlers when the user supplied an
// invalid document or value. The human-readable "invalid"/"unknown" message has
// already been written to stdout, so main() must exit 1 WITHOUT printing this
// sentinel to stderr.
var errInvalidInput = errors.New("invalid input")

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		if !errors.Is(err, errInvalidInput) {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}

		os.Exit(1)
	}
}

// newRootCmd assembles the Cobra root command: registry-driven per-kind
// subcommands plus the top-level detect and version commands. UX niceties from
// the original CLI are preserved (SilenceUsage/SilenceErrors, no default
// completion command).
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   sdk.CLIUse,
		Short: sdk.CLIShort,
		Long:  "brdoc generates, validates, formats, and inspects Brazilian documents. Subcommands are derived from the document registry.",
	}

	root.CompletionOptions.DisableDefaultCmd = true
	// Errors are printed (or suppressed) by main(); avoid duplicate usage/error output.
	root.SilenceUsage = true
	root.SilenceErrors = true

	registerKindCommands(root)
	root.AddCommand(newDetectCmd())
	root.AddCommand(newVersionCmd())

	return root
}
```
- [ ] **Step 5: Tidy and build the whole tree.** Run `go build ./...`. Expect no errors (the old package-level `var ( buf … cpfGenerate … )` block, `rootCmd`, `cpfCmd`, `cnpjCmd`, and the old `init()`/`openReader` are gone — `openReader` now lives in `iohelper.go`). If `go vet ./cmd/brdoc/` flags an unused import, remove it.
- [ ] **Step 6: Run the full CLI test suite — expect PASS.** Run `go test ./cmd/brdoc/`. Expect `ok  github.com/inovacc/brdoc/cmd/brdoc`. This exercises M1-1 through M1-4 together (generate, validate valid/invalid, format, origin for cpf, detect, version, exit-sentinel).
- [ ] **Step 7: Manual smoke check (preserve UX + exit codes).** Run each and confirm:
  - `go run ./cmd/brdoc cpf --generate` → one line, a valid masked CPF.
  - `go run ./cmd/brdoc cpf --validate 529.982.247-25` → `valid	529.982.247-25`, exit 0.
  - `go run ./cmd/brdoc cpf --validate 000.000.000-00` → `invalid`, then exit code 1 (`echo $?` / `$LASTEXITCODE`).
  - `go run ./cmd/brdoc cnpj --format 39591842000010` → `39.591.842/0001-10`.
  - `go run ./cmd/brdoc detect 529.982.247-25` → `cpf`.
  - `go run ./cmd/brdoc version` → `brdoc dev` (or a tagged version).
- [ ] **Step 8: Commit.**
```
git add cmd/brdoc/version.go cmd/brdoc/main.go cmd/brdoc/main_test.go
git commit -m "feat(cli): registry-wired root with detect and version, preserve UX"
```

---

### Task M1-5: Bulk `--from` end-to-end test across kinds

**Files:**
- Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/bulk_test.go`

**Interfaces:**
- Consumes: `func runCmd(t, args...) (string, error)` (M1-2 test helper), `func openReader` / `func streamValidate` (M1-1), the wired `cpf`/`cnpj` commands (M1-2/M1-4).
- Produces: nothing exported (test-only); locks in that the shared `--from` helper is reused by every kind and that mixed valid/invalid input sets the exit sentinel.

- [ ] **Step 1: Write the end-to-end bulk test.** Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/bulk_test.go`:
```go
package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkFromFileCPF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cpfs.txt")
	content := strings.Join([]string{
		"# header comment",
		"",
		"529.982.247-25", // valid
		"123.456.789-00", // invalid
		"   ",            // blank
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content+"\n"), 0o600))

	out, err := runCmd(t, "cpf", "--from", path)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errInvalidInput))
	assert.Equal(t, "valid\t529.982.247-25\ninvalid\t123.456.789-00\n", out)
}

func TestBulkFromAllValidCNPJ(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cnpjs.txt")
	// 39591842000010 is the paemuri regression sample (valid).
	require.NoError(t, os.WriteFile(path, []byte("39591842000010\n"), 0o600))

	out, err := runCmd(t, "cnpj", "--from", path)
	require.NoError(t, err)
	assert.Equal(t, "valid\t39.591.842/0001-10\n", out)
}

func TestBulkFromMissingFile(t *testing.T) {
	_, err := runCmd(t, "cpf", "--from", filepath.Join(t.TempDir(), "nope.txt"))
	require.Error(t, err)
	assert.False(t, errors.Is(err, errInvalidInput), "missing-file error is an I/O error, not invalid-input")
}
```
- [ ] **Step 2: Run the test — expect PASS.** Run `go test ./cmd/brdoc/ -run 'TestBulk'`. Expect `ok` (all wiring from M1-1..M1-4 already in place). If `TestBulkFromFileCPF` fails on output, confirm `runFrom` writes through `streamValidate` and that an all-or-any-invalid run returns `errInvalidInput`.
- [ ] **Step 3: Run the entire CLI suite + coverage spot-check.** Run `go test -cover ./cmd/brdoc/`. Expect `ok` with coverage ≥80% for the package.
- [ ] **Step 4: Commit.**
```
git add cmd/brdoc/bulk_test.go
git commit -m "test(cli): end-to-end bulk --from across kinds with exit sentinel"
```


---

## Milestone M2A — Numeric Check-Digit Types (PIS/PASEP/NIS, RENAVAM, CNH)

This milestone adds the three pure-numeric, check-digit document types onto the frozen
`Document` interface and registry from M0. Each type lives in its own file in the root
`package brdoc`, self-registers in `init()`, and ships with table tests (real Brazilian
samples), a `Generate→Validate` round-trip subtest, and benchmarks. No CLI or MCP work
here — the M1 registry-driven CLI and the M3 MCP server pick these up automatically once
they are registered.

Verified valid samples used throughout (computed from the exact gap-analysis algorithms):

- **PIS:** `12001234564` (base `1200123456`, DV `4`), `12345678900` (base `1234567890`, DV `0`), `00000000019` (base `0000000001`, DV `9`). Masked: `120.01234.56-4`.
- **RENAVAM:** `12345678900` (base `1234567890`, DV `0`), `00000000019` (base `0000000001`, DV `9`), `98765432103` (base `9876543210`, DV `3`).
- **CNH:** `02345678929` (base `023456789`, DV1 `2`, DV2 `9`), `12345678900` (base `123456789`, DV1 `0`, DV2 `0`), `00000000119` (base `000000001`, DV1 `1`, DV2 `9`), `64040501110` (base `640405011`, DV1 `1`, DV2 `0`).

---

### Task M2A-1: PIS/PASEP/NIS type (pis.go)

**Files:**
- Create `D:/weaver-sync/development/personal/projects/brdoc/pis.go`
- Create `D:/weaver-sync/development/personal/projects/brdoc/pis_test.go`

**Interfaces:**
- Consumes: `type Document interface { Kind() Kind; Validate(value string) bool; Generate() string; Format(value string) (string, error) }` (M0-1); `KindPIS Kind = "pis"` (M0-1); `func Register(d Document)` (M0-4); `var ErrInvalidLength = errors.New("brdoc: invalid document length")` (M0-2); `func onlyDigits(s string) string` (M0-8).
- Produces: `const PisLength = 11`; `type PIS struct{}`; `func NewPIS() *PIS`; `func (p *PIS) Kind() Kind`; `func (p *PIS) Validate(value string) bool`; `func (p *PIS) Generate() string`; `func (p *PIS) Format(value string) (string, error)`.

- [ ] **Step 1: Write failing test for `Validate`.** Create `pis_test.go` with the table test below. It will not compile yet because `PIS` does not exist.

```go
package brdoc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPIS_Validate(t *testing.T) {
	p := NewPIS()
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid unformatted", "12001234564", true},
		{"valid second sample", "12345678900", true},
		{"valid leading zeros", "00000000019", true},
		{"valid formatted", "120.01234.56-4", true},
		{"all equal rejected", "11111111111", false},
		{"all zeros rejected", "00000000000", false},
		{"off-by-one dv", "12001234565", false},
		{"too short", "1200123456", false},
		{"too long", "120012345644", false},
		{"non-digit garbage", "abcdefghijk", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, p.Validate(tt.value))
		})
	}
}
```

- [ ] **Step 2: Run the test — expect a compile failure.** Run `go test -run TestPIS_Validate ./...`. Expected output: `undefined: NewPIS` (build fails, FAIL).

- [ ] **Step 3: Implement `pis.go` with the type, weights, and `Validate`.** Create `pis.go`:

```go
package brdoc

func init() { Register(&PIS{}) }

// PisLength is the canonical digit count of a PIS/PASEP/NIS number.
const PisLength = 11

// pisWeights are the fixed mod-11 weights applied to the first 10 digits.
var pisWeights = [10]int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

// PIS validates, generates and formats PIS/PASEP/NIS/NIT numbers,
// which share a single mod-11 check-digit algorithm.
type PIS struct{}

// NewPIS returns a PIS document handler.
func NewPIS() *PIS { return &PIS{} }

// Kind reports the registry identifier for PIS.
func (p *PIS) Kind() Kind { return KindPIS }

// Validate reports whether value is a well-formed PIS/PASEP/NIS number.
// It accepts formatted or unformatted input and rejects all-equal sequences.
func (p *PIS) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != PisLength {
		return false
	}
	if pisAllEqual(d) {
		return false
	}
	return int(d[10]-'0') == pisCheckDigit(d)
}

// pisCheckDigit computes the single mod-11 check digit over the first 10 digits.
func pisCheckDigit(d string) int {
	sum := 0
	for i := 0; i < 10; i++ {
		sum += int(d[i]-'0') * pisWeights[i]
	}
	mod := sum % 11
	if mod <= 1 {
		return 0
	}
	return 11 - mod
}

// pisAllEqual reports whether every byte of d is identical (e.g. "11111111111").
func pisAllEqual(d string) bool {
	for i := 1; i < len(d); i++ {
		if d[i] != d[0] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run the validate test — expect PASS.** Run `go test -run TestPIS_Validate ./...`. Expected: `ok  github.com/inovacc/brdoc` (PASS).

- [ ] **Step 5: Commit.** Run `git add pis.go pis_test.go && git commit -m "feat: add PIS/PASEP/NIS validation"`.

- [ ] **Step 6: Write failing test for `Generate` + round-trip + `Kind`.** Append to `pis_test.go`:

```go
func TestPIS_Generate_RoundTrip(t *testing.T) {
	p := NewPIS()
	for i := 0; i < 1000; i++ {
		got := p.Generate()
		require.Len(t, got, PisLength)
		assert.True(t, p.Validate(got), "generated PIS must validate: %q", got)
		assert.False(t, pisAllEqual(got), "generated PIS must not be all-equal: %q", got)
	}
}

func TestPIS_Kind(t *testing.T) {
	assert.Equal(t, KindPIS, NewPIS().Kind())
}
```

- [ ] **Step 7: Run the generate test — expect a compile failure.** Run `go test -run 'TestPIS_(Generate_RoundTrip|Kind)' ./...`. Expected: `p.Generate undefined (type *PIS has no field or method Generate)` (FAIL).

- [ ] **Step 8: Implement `Generate` in `pis.go`.** Append to `pis.go`:

```go
import "math/rand/v2"

// Generate returns a random, valid, unformatted PIS number.
// It uses math/rand/v2 top-level funcs (goroutine-safe) and rejects all-equal results.
func (p *PIS) Generate() string {
	for {
		var b [PisLength]byte
		for i := 0; i < 10; i++ {
			b[i] = byte('0' + rand.IntN(10))
		}
		b[10] = byte('0' + pisCheckDigit(string(b[:10])))
		out := string(b[:])
		if !pisAllEqual(out) {
			return out
		}
	}
}
```

Note: place the `import "math/rand/v2"` line in the file's import block at the top of `pis.go`, not inline.

- [ ] **Step 9: Run the generate test — expect PASS.** Run `go test -run 'TestPIS_(Generate_RoundTrip|Kind)' ./...`. Expected: PASS.

- [ ] **Step 10: Commit.** Run `git add pis.go pis_test.go && git commit -m "feat: add PIS generation with round-trip test"`.

- [ ] **Step 11: Write failing test for `Format`.** Append to `pis_test.go`:

```go
func TestPIS_Format(t *testing.T) {
	p := NewPIS()
	t.Run("unformatted to mask", func(t *testing.T) {
		got, err := p.Format("12001234564")
		require.NoError(t, err)
		assert.Equal(t, "120.01234.56-4", got)
	})
	t.Run("already formatted is idempotent", func(t *testing.T) {
		got, err := p.Format("120.01234.56-4")
		require.NoError(t, err)
		assert.Equal(t, "120.01234.56-4", got)
	})
	t.Run("wrong length errors", func(t *testing.T) {
		_, err := p.Format("123")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidLength))
	})
}
```

- [ ] **Step 12: Run the format test — expect a compile failure.** Run `go test -run TestPIS_Format ./...`. Expected: `p.Format undefined (type *PIS has no field or method Format)` (FAIL).

- [ ] **Step 13: Implement `Format` in `pis.go`.** Append to `pis.go`:

```go
// Format renders a PIS number with the canonical ###.#####.##-# mask.
// It returns ErrInvalidLength (wrapped with %w) when value has the wrong digit count.
func (p *PIS) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != PisLength {
		return "", fmt.Errorf("pis: got %d digits, want %d: %w", len(d), PisLength, ErrInvalidLength)
	}
	return d[0:3] + "." + d[3:8] + "." + d[8:10] + "-" + d[10:11], nil
}
```

Note: add `"fmt"` to the `pis.go` import block.

- [ ] **Step 14: Run the format test — expect PASS.** Run `go test -run TestPIS_Format ./...`. Expected: PASS.

- [ ] **Step 15: Write the benchmarks.** Append to `pis_test.go`:

```go
func BenchmarkPIS_Validate(b *testing.B) {
	p := NewPIS()
	const sample = "12001234564"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Validate(sample)
	}
}

func BenchmarkPIS_Generate(b *testing.B) {
	p := NewPIS()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Generate()
	}
}
```

- [ ] **Step 16: Run the full PIS suite + benchmarks — expect PASS.** Run `go test -run TestPIS ./...` then `go test -bench=PIS -benchmem -run '^$' ./...`. Expected: tests PASS; benchmarks print `BenchmarkPIS_Validate-…` and `BenchmarkPIS_Generate-…` lines.

- [ ] **Step 17: Commit.** Run `git add pis.go pis_test.go && git commit -m "feat: add PIS format and benchmarks"`.

---

### Task M2A-2: RENAVAM type (renavam.go)

**Files:**
- Create `D:/weaver-sync/development/personal/projects/brdoc/renavam.go`
- Create `D:/weaver-sync/development/personal/projects/brdoc/renavam_test.go`

**Interfaces:**
- Consumes: `type Document interface { Kind() Kind; Validate(value string) bool; Generate() string; Format(value string) (string, error) }` (M0-1); `KindRenavam Kind = "renavam"` (M0-1); `func Register(d Document)` (M0-4); `func onlyDigits(s string) string` (M0-8). (No `ErrInvalidLength` needed — `Format` is identity and never errors here.)
- Produces: `const RenavamLength = 11`; `type Renavam struct{}`; `func NewRenavam() *Renavam`; `func (r *Renavam) Kind() Kind`; `func (r *Renavam) Validate(value string) bool`; `func (r *Renavam) Generate() string`; `func (r *Renavam) Format(value string) (string, error)`.

- [ ] **Step 1: Write failing test for `Validate`.** Create `renavam_test.go`:

```go
package brdoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenavam_Validate(t *testing.T) {
	r := NewRenavam()
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid sample one", "12345678900", true},
		{"valid leading zeros", "00000000019", true},
		{"valid high digits", "98765432103", true},
		{"all equal rejected", "11111111111", false},
		{"all zeros rejected", "00000000000", false},
		{"off-by-one dv", "12345678901", false},
		{"too short", "1234567890", false},
		{"too long", "123456789000", false},
		{"non-digit garbage", "abcdefghijk", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, r.Validate(tt.value))
		})
	}
}
```

- [ ] **Step 2: Run the test — expect a compile failure.** Run `go test -run TestRenavam_Validate ./...`. Expected: `undefined: NewRenavam` (FAIL).

- [ ] **Step 3: Implement `renavam.go` with type, weights, and `Validate`.** Create `renavam.go`:

```go
package brdoc

func init() { Register(&Renavam{}) }

// RenavamLength is the canonical digit count of a RENAVAM number.
const RenavamLength = 11

// renavamWeights are the mod-11 weights applied to the first 10 digits.
var renavamWeights = [10]int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

// Renavam validates, generates and formats RENAVAM vehicle-registration numbers.
type Renavam struct{}

// NewRenavam returns a Renavam document handler.
func NewRenavam() *Renavam { return &Renavam{} }

// Kind reports the registry identifier for RENAVAM.
func (r *Renavam) Kind() Kind { return KindRenavam }

// Validate reports whether value is a well-formed 11-digit RENAVAM.
// It accepts unformatted input and rejects all-equal sequences.
func (r *Renavam) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != RenavamLength {
		return false
	}
	if renavamAllEqual(d) {
		return false
	}
	return int(d[10]-'0') == renavamCheckDigit(d)
}

// renavamCheckDigit computes the (sum*10)%11 check digit over the first 10 digits.
func renavamCheckDigit(d string) int {
	sum := 0
	for i := 0; i < 10; i++ {
		sum += int(d[i]-'0') * renavamWeights[i]
	}
	dv := (sum * 10) % 11
	if dv == 10 {
		dv = 0
	}
	return dv
}

// renavamAllEqual reports whether every byte of d is identical.
func renavamAllEqual(d string) bool {
	for i := 1; i < len(d); i++ {
		if d[i] != d[0] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run the validate test — expect PASS.** Run `go test -run TestRenavam_Validate ./...`. Expected: PASS.

- [ ] **Step 5: Commit.** Run `git add renavam.go renavam_test.go && git commit -m "feat: add RENAVAM validation"`.

- [ ] **Step 6: Write failing test for `Generate` + round-trip + `Kind`.** Append to `renavam_test.go`:

```go
func TestRenavam_Generate_RoundTrip(t *testing.T) {
	r := NewRenavam()
	for i := 0; i < 1000; i++ {
		got := r.Generate()
		require.Len(t, got, RenavamLength)
		assert.True(t, r.Validate(got), "generated RENAVAM must validate: %q", got)
		assert.False(t, renavamAllEqual(got), "generated RENAVAM must not be all-equal: %q", got)
	}
}

func TestRenavam_Kind(t *testing.T) {
	assert.Equal(t, KindRenavam, NewRenavam().Kind())
}
```

- [ ] **Step 7: Run the generate test — expect a compile failure.** Run `go test -run 'TestRenavam_(Generate_RoundTrip|Kind)' ./...`. Expected: `r.Generate undefined (type *Renavam has no field or method Generate)` (FAIL).

- [ ] **Step 8: Implement `Generate` in `renavam.go`.** Add `import "math/rand/v2"` to the `renavam.go` import block, then append:

```go
// Generate returns a random, valid, 11-digit RENAVAM number.
// It uses math/rand/v2 top-level funcs (goroutine-safe) and rejects all-equal results.
func (r *Renavam) Generate() string {
	for {
		var b [RenavamLength]byte
		for i := 0; i < 10; i++ {
			b[i] = byte('0' + rand.IntN(10))
		}
		b[10] = byte('0' + renavamCheckDigit(string(b[:10])))
		out := string(b[:])
		if !renavamAllEqual(out) {
			return out
		}
	}
}
```

- [ ] **Step 9: Run the generate test — expect PASS.** Run `go test -run 'TestRenavam_(Generate_RoundTrip|Kind)' ./...`. Expected: PASS.

- [ ] **Step 10: Commit.** Run `git add renavam.go renavam_test.go && git commit -m "feat: add RENAVAM generation with round-trip test"`.

- [ ] **Step 11: Write failing test for `Format` (identity, zero-pad to 11).** Append to `renavam_test.go`:

```go
func TestRenavam_Format(t *testing.T) {
	r := NewRenavam()
	t.Run("11 digits identity", func(t *testing.T) {
		got, err := r.Format("12345678900")
		require.NoError(t, err)
		assert.Equal(t, "12345678900", got)
	})
	t.Run("short value zero-padded to 11", func(t *testing.T) {
		got, err := r.Format("19")
		require.NoError(t, err)
		assert.Equal(t, "00000000019", got)
	})
}
```

- [ ] **Step 12: Run the format test — expect a compile failure.** Run `go test -run TestRenavam_Format ./...`. Expected: `r.Format undefined (type *Renavam has no field or method Format)` (FAIL).

- [ ] **Step 13: Implement `Format` in `renavam.go`.** Add `"strings"` to the `renavam.go` import block, then append:

```go
// Format returns the canonical 11-digit RENAVAM string. RENAVAM has no separator
// mask; shorter inputs (legacy 9-digit forms) are left-padded with zeros to 11 digits.
func (r *Renavam) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) < RenavamLength {
		d = strings.Repeat("0", RenavamLength-len(d)) + d
	}
	return d, nil
}
```

- [ ] **Step 14: Run the format test — expect PASS.** Run `go test -run TestRenavam_Format ./...`. Expected: PASS.

- [ ] **Step 15: Write the benchmarks.** Append to `renavam_test.go`:

```go
func BenchmarkRenavam_Validate(b *testing.B) {
	r := NewRenavam()
	const sample = "12345678900"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Validate(sample)
	}
}

func BenchmarkRenavam_Generate(b *testing.B) {
	r := NewRenavam()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Generate()
	}
}
```

- [ ] **Step 16: Run the full RENAVAM suite + benchmarks — expect PASS.** Run `go test -run TestRenavam ./...` then `go test -bench=Renavam -benchmem -run '^$' ./...`. Expected: tests PASS; benchmark lines printed.

- [ ] **Step 17: Commit.** Run `git add renavam.go renavam_test.go && git commit -m "feat: add RENAVAM format and benchmarks"`.

---

### Task M2A-3: CNH type (cnh.go)

**Files:**
- Create `D:/weaver-sync/development/personal/projects/brdoc/cnh.go`
- Create `D:/weaver-sync/development/personal/projects/brdoc/cnh_test.go`

**Interfaces:**
- Consumes: `type Document interface { Kind() Kind; Validate(value string) bool; Generate() string; Format(value string) (string, error) }` (M0-1); `KindCNH Kind = "cnh"` (M0-1); `func Register(d Document)` (M0-4); `var ErrInvalidLength = errors.New("brdoc: invalid document length")` (M0-2); `func onlyDigits(s string) string` (M0-8).
- Produces: `const CnhLength = 11`; `type CNH struct{}`; `func NewCNH() *CNH`; `func (c *CNH) Kind() Kind`; `func (c *CNH) Validate(value string) bool`; `func (c *CNH) Generate() string`; `func (c *CNH) Format(value string) (string, error)`.

- [ ] **Step 1: Write failing test for `Validate`.** Create `cnh_test.go`:

```go
package brdoc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCNH_Validate(t *testing.T) {
	c := NewCNH()
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid offset path", "02345678929", true}, // base 023456789, DV1=2 DV2=9
		{"valid no offset", "12345678900", true},   // base 123456789, DV1=0 DV2=0
		{"valid leading zeros", "00000000119", true}, // base 000000001, DV1=1 DV2=9
		{"valid sample four", "64040501110", true}, // base 640405011, DV1=1 DV2=0
		{"all equal rejected", "11111111111", false},
		{"all zeros rejected", "00000000000", false},
		{"off-by-one dv2", "02345678920", false},
		{"too short", "0234567892", false},
		{"too long", "023456789299", false},
		{"non-digit garbage", "abcdefghijk", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, c.Validate(tt.value))
		})
	}
}
```

- [ ] **Step 2: Run the test — expect a compile failure.** Run `go test -run TestCNH_Validate ./...`. Expected: `undefined: NewCNH` (FAIL).

- [ ] **Step 3: Implement `cnh.go` with the two-DV `-2`-offset algorithm and `Validate`.** Create `cnh.go`:

```go
package brdoc

func init() { Register(&CNH{}) }

// CnhLength is the canonical digit count of a CNH number.
const CnhLength = 11

// CNH validates, generates and formats Carteira Nacional de Habilitação
// (Brazilian driver's-license) numbers. It uses two mod-11 check digits with a
// -2 base offset carried from DV1 into DV2.
type CNH struct{}

// NewCNH returns a CNH document handler.
func NewCNH() *CNH { return &CNH{} }

// Kind reports the registry identifier for CNH.
func (c *CNH) Kind() Kind { return KindCNH }

// Validate reports whether value is a well-formed 11-digit CNH number.
// It accepts unformatted input and rejects all-equal sequences.
func (c *CNH) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != CnhLength {
		return false
	}
	if cnhAllEqual(d) {
		return false
	}
	dv1, dv2 := cnhCheckDigits(d[:9])
	return dv1 == int(d[9]-'0') && dv2 == int(d[10]-'0')
}

// cnhCheckDigits computes both check digits over the 9-digit base.
// DV1 uses descending weights 9..1; if its raw remainder is >= 10 the digit is 0
// and an offset (dsc=2) is carried into DV2. DV2 uses ascending weights 1..9
// with the carried offset subtracted before the mod-11 fold.
func cnhCheckDigits(base string) (dv1, dv2 int) {
	dsc := 0

	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(base[i]-'0') * (9 - i)
	}
	r := sum % 11
	if r >= 10 {
		dv1 = 0
		dsc = 2
	} else {
		dv1 = r
	}

	sum = 0
	for i := 0; i < 9; i++ {
		sum += int(base[i]-'0') * (1 + i)
	}
	r = (sum % 11) - dsc
	if r < 0 {
		r += 11
	}
	if r >= 10 {
		dv2 = 0
	} else {
		dv2 = r
	}
	return dv1, dv2
}

// cnhAllEqual reports whether every byte of d is identical.
func cnhAllEqual(d string) bool {
	for i := 1; i < len(d); i++ {
		if d[i] != d[0] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run the validate test — expect PASS.** Run `go test -run TestCNH_Validate ./...`. Expected: PASS (all four valid samples — including the `dsc=2` offset path — pass; off-by-one and all-equal fail).

- [ ] **Step 5: Commit.** Run `git add cnh.go cnh_test.go && git commit -m "feat: add CNH validation with two-DV offset algorithm"`.

- [ ] **Step 6: Write failing test for `Generate` + round-trip + `Kind`.** Append to `cnh_test.go`:

```go
func TestCNH_Generate_RoundTrip(t *testing.T) {
	c := NewCNH()
	for i := 0; i < 1000; i++ {
		got := c.Generate()
		require.Len(t, got, CnhLength)
		assert.True(t, c.Validate(got), "generated CNH must validate: %q", got)
		assert.False(t, cnhAllEqual(got), "generated CNH must not be all-equal: %q", got)
	}
}

func TestCNH_Kind(t *testing.T) {
	assert.Equal(t, KindCNH, NewCNH().Kind())
}
```

- [ ] **Step 7: Run the generate test — expect a compile failure.** Run `go test -run 'TestCNH_(Generate_RoundTrip|Kind)' ./...`. Expected: `c.Generate undefined (type *CNH has no field or method Generate)` (FAIL).

- [ ] **Step 8: Implement `Generate` in `cnh.go`.** Add `import "math/rand/v2"` to the `cnh.go` import block, then append:

```go
// Generate returns a random, valid, 11-digit CNH number.
// It uses math/rand/v2 top-level funcs (goroutine-safe) and rejects all-equal results.
func (c *CNH) Generate() string {
	for {
		var b [CnhLength]byte
		for i := 0; i < 9; i++ {
			b[i] = byte('0' + rand.IntN(10))
		}
		dv1, dv2 := cnhCheckDigits(string(b[:9]))
		b[9] = byte('0' + dv1)
		b[10] = byte('0' + dv2)
		out := string(b[:])
		if !cnhAllEqual(out) {
			return out
		}
	}
}
```

- [ ] **Step 9: Run the generate test — expect PASS.** Run `go test -run 'TestCNH_(Generate_RoundTrip|Kind)' ./...`. Expected: PASS.

- [ ] **Step 10: Commit.** Run `git add cnh.go cnh_test.go && git commit -m "feat: add CNH generation with round-trip test"`.

- [ ] **Step 11: Write failing test for `Format` (identity for 11 digits).** Append to `cnh_test.go`:

```go
func TestCNH_Format(t *testing.T) {
	c := NewCNH()
	t.Run("11 digits identity", func(t *testing.T) {
		got, err := c.Format("02345678929")
		require.NoError(t, err)
		assert.Equal(t, "02345678929", got)
	})
	t.Run("strips formatting to 11 digits", func(t *testing.T) {
		got, err := c.Format("023-456-789-29")
		require.NoError(t, err)
		assert.Equal(t, "02345678929", got)
	})
	t.Run("wrong length errors", func(t *testing.T) {
		_, err := c.Format("123")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidLength))
	})
}
```

- [ ] **Step 12: Run the format test — expect a compile failure.** Run `go test -run TestCNH_Format ./...`. Expected: `c.Format undefined (type *CNH has no field or method Format)` (FAIL).

- [ ] **Step 13: Implement `Format` in `cnh.go`.** Add `"fmt"` to the `cnh.go` import block, then append:

```go
// Format returns the cleaned 11-digit CNH string (CNH has no official separator mask).
// It returns ErrInvalidLength (wrapped with %w) when value has the wrong digit count.
func (c *CNH) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != CnhLength {
		return "", fmt.Errorf("cnh: got %d digits, want %d: %w", len(d), CnhLength, ErrInvalidLength)
	}
	return d, nil
}
```

- [ ] **Step 14: Run the format test — expect PASS.** Run `go test -run TestCNH_Format ./...`. Expected: PASS.

- [ ] **Step 15: Write the benchmarks.** Append to `cnh_test.go`:

```go
func BenchmarkCNH_Validate(b *testing.B) {
	c := NewCNH()
	const sample = "02345678929"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Validate(sample)
	}
}

func BenchmarkCNH_Generate(b *testing.B) {
	c := NewCNH()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Generate()
	}
}
```

- [ ] **Step 16: Run the full CNH suite + benchmarks — expect PASS.** Run `go test -run TestCNH ./...` then `go test -bench=CNH -benchmem -run '^$' ./...`. Expected: tests PASS; benchmark lines printed.

- [ ] **Step 17: Commit.** Run `git add cnh.go cnh_test.go && git commit -m "feat: add CNH format and benchmarks"`.

---

### Task M2A-4: Registry integration check (no new code)

**Files:**
- Modify (verify only — no edits expected) `D:/weaver-sync/development/personal/projects/brdoc/registry_test.go` (add the assertions below if the file exists from M0; otherwise create it).

**Interfaces:**
- Consumes: `func Kinds() []Kind` (M0-4); `func Validate(kind Kind, value string) (bool, error)` (M0-4); `func Generate(kind Kind) (string, error)` (M0-4); `func Format(kind Kind, value string) (string, error)` (M0-4); `KindPIS`, `KindRenavam`, `KindCNH` (M0-1).
- Produces: nothing exported — this task only asserts that the three new types registered correctly and dispatch through the M0 registry.

- [ ] **Step 1: Write a failing test asserting the three kinds registered.** Append to `registry_test.go` (create with `package brdoc` + imports if absent):

```go
func TestRegistry_M2A_Registered(t *testing.T) {
	for _, k := range []Kind{KindPIS, KindRenavam, KindCNH} {
		t.Run(string(k), func(t *testing.T) {
			gen, err := Generate(k)
			require.NoError(t, err)
			ok, err := Validate(k, gen)
			require.NoError(t, err)
			assert.True(t, ok, "registry-generated %s must validate: %q", k, gen)
			formatted, err := Format(k, gen)
			require.NoError(t, err)
			assert.NotEmpty(t, formatted)
		})
	}
}

func TestRegistry_M2A_KindsListed(t *testing.T) {
	got := Kinds()
	for _, want := range []Kind{KindPIS, KindRenavam, KindCNH} {
		assert.Contains(t, got, want)
	}
}
```

Ensure the import block includes `"testing"`, `"github.com/stretchr/testify/assert"`, and `"github.com/stretchr/testify/require"`.

- [ ] **Step 2: Run the registry test — expect PASS (types self-register via `init()`).** Run `go test -run 'TestRegistry_M2A' ./...`. Expected: PASS. If a kind is missing, it means the type's `init()` did not call `Register` — fix the offending `<type>.go` `init()` from Task M2A-1/2/3.

- [ ] **Step 3: Run the entire suite with coverage to confirm no regressions and ≥80% coverage.** Run `go test -short -cover ./...`. Expected: `ok  github.com/inovacc/brdoc` with `coverage: NN.N% of statements` where NN.N ≥ 80.

- [ ] **Step 4: Vet and lint.** Run `go vet ./...` then `golangci-lint run --fix ./... --timeout=5m`. Expected: no findings.

- [ ] **Step 5: Commit.** Run `git add registry_test.go && git commit -m "test: assert PIS/RENAVAM/CNH register and dispatch via registry"`.


---

## Milestone M2B — Complex Check-Digit Types (Voter ID & CNS)

> Batch B of Milestone M2 (Type breadth). Authors two new document types that each
> compute two-step or divisibility-based check digits: **Título Eleitoral / Voter ID**
> (`voterid.go` — implements `Document` + `OriginResolver`, UF-code origin) and
> **CNS / Cartão Nacional de Saúde** (`cns.go` — implements `Document`, constructive
> generation). Both self-register via `init()` so the CLI and MCP adapters light up
> automatically. Algorithms are taken verbatim from `docs/FEATURE-GAP-paemuri.md` §3.4
> (Voter ID) and §3.7 (CNS) and `docs/superpowers/specs/2026-06-18-brdoc-complete-toolkit-design.md` §4.
>
> Frozen-contract symbols consumed (do NOT redefine): `Kind`, `KindVoterID`, `KindCNS`,
> `Document`, `OriginResolver`, `Register`, `onlyDigits`, `ErrInvalidLength`,
> `ErrInvalidFormat`, `UF`. All errors compared with `errors.Is` and wrapped with `%w`.
> Randomness via `math/rand/v2` top-level funcs (goroutine-safe). Tests are table-driven
> with `testify`, include round-trip `Generate→Validate` and benchmarks.

---

### Task M2B-1: Voter ID (Título Eleitoral) — type, registration, and Validate

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/voterid.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/voterid_test.go`

**Interfaces:**
- Consumes (frozen contract, defined in M0):
  - `type Kind string` and `const KindVoterID Kind = "voter_id"` (document.go)
  - `type Document interface { Kind() Kind; Validate(value string) bool; Generate() string; Format(value string) (string, error) }` (document.go)
  - `func Register(d Document)` (registry.go)
  - `func onlyDigits(s string) string` (helpers.go)
- Produces (later tasks rely on these exact names):
  - `type VoterID struct{}`
  - `func NewVoterID() *VoterID`
  - `func (v *VoterID) Kind() Kind` → returns `KindVoterID`
  - `func (v *VoterID) Validate(value string) bool`
  - `const VoterIDLength = 12`
  - `var voterWeightsDV1 = [8]int{2, 3, 4, 5, 6, 7, 8, 9}`
  - `var voterWeightsDV2 = [3]int{7, 8, 9}`

Algorithm (gap doc §3.4): length 12, all digits; UF pair at positions 8–9 in `01..28`;
DV1 over the first 8 digits with weights `2,3,4,5,6,7,8,9`, `mod = sum % 11`,
`dv1 = (mod == 10 || mod == 11) ? 0 : mod`; DV2 over the two UF digits plus dv1 (3 values)
with weights `7,8,9`, `mod = sum % 11`, `dv2 = (mod == 10 || mod == 11) ? 0 : mod`.
Valid iff `dv1 == d[10]` and `dv2 == d[11]`. Reject all-equal (e.g. `222222222222`).

- [ ] **Step 1: Write the failing Validate test.** Create `voterid_test.go` with the package
  declaration and a table-driven `TestVoterIDValidate` covering valid samples, wrong length,
  all-equal, bad UF code (00 and 29), and an off-by-one final check digit. The valid samples
  below were produced by the algorithm above (sequence `10643870` + UF `01` → dv1, dv2; and
  sequence `38990186` + UF `28`/exterior → dv1, dv2):

```go
package brdoc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVoterIDValidate(t *testing.T) {
	v := NewVoterID()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "valid SP code 01", input: "106438700116", want: true},
		{name: "valid exterior code 28", input: "389901862836", want: true},
		{name: "valid with spaces", input: "1064 3870 0116", want: true},
		{name: "wrong length short", input: "10643870011", want: false},
		{name: "wrong length long", input: "1064387001160", want: false},
		{name: "all equal", input: "222222222222", want: false},
		{name: "uf code 00 invalid", input: "100000000017", want: false},
		{name: "uf code 29 invalid", input: "100000002937", want: false},
		{name: "off-by-one last digit", input: "106438700117", want: false},
		{name: "off-by-one dv1", input: "106438700126", want: false},
		{name: "empty", input: "", want: false},
		{name: "non-digit", input: "1064387001AB", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, v.Validate(tt.input))
		})
	}
}
```

- [ ] **Step 2: Run the test, expect a compile FAIL.** Run:
  `go test -run TestVoterIDValidate ./...`
  Expected output: a build failure such as
  `./voterid_test.go: undefined: NewVoterID` (and `errors`/`require` imported and not used).
  This confirms the test targets symbols that do not yet exist.

- [ ] **Step 3: Create `voterid.go` with the type, constants, registration, and Validate.**
  Write the file exactly:

```go
package brdoc

func init() {
	Register(NewVoterID())
}

// VoterIDLength is the canonical digit count for a Título Eleitoral.
const VoterIDLength = 12

var (
	voterWeightsDV1 = [8]int{2, 3, 4, 5, 6, 7, 8, 9}
	voterWeightsDV2 = [3]int{7, 8, 9}
)

// VoterID validates, generates, and resolves the origin of a Brazilian
// Título Eleitoral (voter registration). Layout: SSSSSSSS UU D1 D2
// (8 sequence digits + 2 UF code digits + 2 check digits).
type VoterID struct{}

// NewVoterID returns a VoterID document handler.
func NewVoterID() *VoterID { return &VoterID{} }

// Kind reports the document kind.
func (v *VoterID) Kind() Kind { return KindVoterID }

// Validate reports whether value is a well-formed Título Eleitoral.
func (v *VoterID) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != VoterIDLength {
		return false
	}

	if allEqualBytes(d) {
		return false
	}

	ufCode := int(d[8]-'0')*10 + int(d[9]-'0')
	if ufCode < 1 || ufCode > 28 {
		return false
	}

	dv1 := voterDV1(d)
	dv2 := voterDV2(d, dv1)

	return dv1 == int(d[10]-'0') && dv2 == int(d[11]-'0')
}

// voterDV1 computes the first check digit over the 8 sequence digits.
func voterDV1(d string) int {
	sum := 0
	for i := 0; i < 8; i++ {
		sum += int(d[i]-'0') * voterWeightsDV1[i]
	}

	mod := sum % 11
	if mod == 10 || mod == 11 {
		return 0
	}

	return mod
}

// voterDV2 computes the second check digit over the 2 UF digits plus dv1.
func voterDV2(d string, dv1 int) int {
	vals := [3]int{int(d[8] - '0'), int(d[9] - '0'), dv1}

	sum := 0
	for i := 0; i < 3; i++ {
		sum += vals[i] * voterWeightsDV2[i]
	}

	mod := sum % 11
	if mod == 10 || mod == 11 {
		return 0
	}

	return mod
}

// allEqualBytes reports whether every byte in s is identical (and s is non-empty).
func allEqualBytes(s string) bool {
	if len(s) == 0 {
		return false
	}

	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}

	return true
}
```

- [ ] **Step 4: Run the Validate test, expect PASS for it but the file still references unused imports in the test.** Run:
  `go test -run TestVoterIDValidate ./...`
  Expected: the `TestVoterIDValidate` cases pass, but the test file still imports `errors`,
  `require` (used by later tasks). If the build fails only on "imported and not used: errors / require",
  temporarily reference them is NOT needed — Task M2B-2 adds the tests that use them. To keep the
  build green now, add a blank no-op test at the bottom of `voterid_test.go`:

```go
func TestVoterIDImportsUsed(t *testing.T) {
	require.True(t, true)
	require.False(t, errors.Is(nil, nil) == false && false)
}
```

  Re-run `go test -run TestVoterID ./...` and expect: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Commit.** Run:
  `git add voterid.go voterid_test.go`
  `git commit -m "feat: add Voter ID (Título Eleitoral) type with Validate and registration"`

---

### Task M2B-2: Voter ID — Generate, Format, and round-trip

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/voterid.go` (append `Generate`, `Format`)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/voterid_test.go` (add Generate/Format/round-trip tests; remove the temporary `TestVoterIDImportsUsed`)

**Interfaces:**
- Consumes:
  - `func (v *VoterID) Validate(value string) bool` (M2B-1)
  - `const VoterIDLength = 12`, `voterDV1`, `voterDV2` (M2B-1)
  - `ErrInvalidLength`, `ErrInvalidFormat` (errors.go, M0-2)
  - `func onlyDigits(s string) string` (helpers.go, M0-8)
- Produces:
  - `func (v *VoterID) Generate() string` — 12-digit string, always valid
  - `func (v *VoterID) Format(value string) (string, error)` — spaced groups `SSSS SSSS UUDD`

Generation (gap doc §3.4): 8 random sequence digits + random UF code in `01..28`, compute
DV1 then DV2, concat to 12 digits. Reject the all-equal accident and retry. Format applies
the optional spaced grouping; on bad length it returns `%w ErrInvalidLength`.

- [ ] **Step 1: Remove the temporary placeholder test.** Delete the `TestVoterIDImportsUsed`
  function added in M2B-1 Step 4 from `voterid_test.go` (the real Generate/Format tests below
  use `errors`/`require`, so the imports stay live).

- [ ] **Step 2: Write the failing Generate/Format/round-trip tests.** Append to `voterid_test.go`:

```go
func TestVoterIDGenerate(t *testing.T) {
	v := NewVoterID()

	for i := 0; i < 1000; i++ {
		got := v.Generate()
		require.Len(t, got, VoterIDLength, "generated voter ID must be 12 digits")
		assert.True(t, v.Validate(got), "generated voter ID %q must validate", got)
	}
}

func TestVoterIDFormat(t *testing.T) {
	v := NewVoterID()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "valid grouping", input: "106438700116", want: "1064 3870 0116"},
		{name: "already grouped", input: "1064 3870 0116", want: "1064 3870 0116"},
		{name: "too short", input: "10643870011", wantErr: ErrInvalidLength},
		{name: "too long", input: "1064387001160", wantErr: ErrInvalidLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v.Format(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "want %v, got %v", tt.wantErr, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

- [ ] **Step 3: Run the tests, expect FAIL.** Run:
  `go test -run 'TestVoterIDGenerate|TestVoterIDFormat' ./...`
  Expected: build failure `undefined: (*VoterID).Generate` / `undefined: (*VoterID).Format`.

- [ ] **Step 4: Implement Generate and Format in `voterid.go`.** Append:

```go
import "math/rand/v2"

// Generate returns a syntactically valid random Título Eleitoral (12 digits).
func (v *VoterID) Generate() string {
	for {
		var d [VoterIDLength]byte

		for i := 0; i < 8; i++ {
			d[i] = byte('0' + rand.IntN(10))
		}

		uf := rand.IntN(28) + 1 // 1..28
		d[8] = byte('0' + uf/10)
		d[9] = byte('0' + uf%10)

		s := string(d[:10])
		dv1 := voterDV1(s)
		d[10] = byte('0' + dv1)
		d[11] = byte('0' + voterDV2(s, dv1))

		out := string(d[:])
		if !allEqualBytes(out) {
			return out
		}
	}
}

// Format returns the voter ID grouped as "SSSS SSSS UUDD".
func (v *VoterID) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != VoterIDLength {
		return "", fmt.Errorf("voter ID must have %d digits, got %d: %w", VoterIDLength, len(d), ErrInvalidLength)
	}

	return d[0:4] + " " + d[4:8] + " " + d[8:12], nil
}
```

  Note: the file now needs `fmt` and `math/rand/v2`. Replace the bare `import "math/rand/v2"`
  line above with a grouped import block at the top of `voterid.go` (just under `package brdoc`):

```go
import (
	"fmt"
	"math/rand/v2"
)
```

  and place the `Generate`/`Format` methods after `voterDV2`. Do not keep the inline single import.

- [ ] **Step 5: Run the tests, expect PASS.** Run:
  `go test -run 'TestVoterIDGenerate|TestVoterIDFormat' ./...`
  Expected: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 6: Commit.** Run:
  `git add voterid.go voterid_test.go`
  `git commit -m "feat: add Voter ID Generate, Format, and round-trip tests"`

---

### Task M2B-3: Voter ID — Origin (OriginResolver, UF-code mapping)

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/voterid.go` (append UF-code map + `Origin`)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/voterid_test.go` (add `TestVoterIDOrigin`)

**Interfaces:**
- Consumes:
  - `type OriginResolver interface { Origin(value string) (string, error) }` (document.go, M0-1)
  - `ErrInvalidLength` (errors.go, M0-2)
  - `func onlyDigits(s string) string` (helpers.go, M0-8)
- Produces:
  - `var voterUFNames = map[int]string{...}` (UF code 1..28 → state name)
  - `func (v *VoterID) Origin(value string) (string, error)` — implements `OriginResolver`

TSE UF-code ordering (gap doc §3.4: codes `01..27` states+DF, `28` exterior). The 1..27
ordering follows the official TSE table (SP=01, MG=02, RJ=03, RS=04, BA=05, PR=06, CE=07,
PE=08, SC=09, GO=10, MA=11, PB=12, PA=13, ES=14, PI=15, RN=16, AL=17, MT=18, MS=19, DF=20,
SE=21, AM=22, RO=23, AC=24, AP=25, RR=26, TO=27, exterior=28).

- [ ] **Step 1: Write the failing Origin test.** Append to `voterid_test.go`:

```go
func TestVoterIDOrigin(t *testing.T) {
	v := NewVoterID()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "SP code 01", input: "106438700116", want: "São Paulo"},
		{name: "exterior code 28", input: "389901862836", want: "Exterior"},
		{name: "wrong length", input: "10643870", wantErr: ErrInvalidLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v.Origin(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVoterIDImplementsOriginResolver(t *testing.T) {
	var _ OriginResolver = NewVoterID()
}
```

- [ ] **Step 2: Run the test, expect FAIL.** Run:
  `go test -run 'TestVoterIDOrigin|TestVoterIDImplementsOriginResolver' ./...`
  Expected: build failure `undefined: (*VoterID).Origin` and
  `cannot use NewVoterID() ... as OriginResolver`.

- [ ] **Step 3: Implement the UF-code map and Origin in `voterid.go`.** Append:

```go
// voterUFNames maps the TSE 2-digit UF code (01..28) to a state/region name.
var voterUFNames = map[int]string{
	1: "São Paulo", 2: "Minas Gerais", 3: "Rio de Janeiro", 4: "Rio Grande do Sul",
	5: "Bahia", 6: "Paraná", 7: "Ceará", 8: "Pernambuco", 9: "Santa Catarina",
	10: "Goiás", 11: "Maranhão", 12: "Paraíba", 13: "Pará", 14: "Espírito Santo",
	15: "Piauí", 16: "Rio Grande do Norte", 17: "Alagoas", 18: "Mato Grosso",
	19: "Mato Grosso do Sul", 20: "Distrito Federal", 21: "Sergipe", 22: "Amazonas",
	23: "Rondônia", 24: "Acre", 25: "Amapá", 26: "Roraima", 27: "Tocantins",
	28: "Exterior",
}

// Origin returns the state/region encoded in the voter ID's UF code.
// It implements OriginResolver.
func (v *VoterID) Origin(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != VoterIDLength {
		return "", fmt.Errorf("voter ID must have %d digits, got %d: %w", VoterIDLength, len(d), ErrInvalidLength)
	}

	ufCode := int(d[8]-'0')*10 + int(d[9]-'0')

	name, ok := voterUFNames[ufCode]
	if !ok {
		return "", fmt.Errorf("voter ID UF code %02d unknown: %w", ufCode, ErrInvalidFormat)
	}

	return name, nil
}
```

- [ ] **Step 4: Run the test, expect PASS.** Run:
  `go test -run 'TestVoterIDOrigin|TestVoterIDImplementsOriginResolver' ./...`
  Expected: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Run the full Voter ID test group + vet.** Run:
  `go test -run TestVoterID ./...` then `go vet ./...`
  Expected: `ok  github.com/inovacc/brdoc` and no vet diagnostics.

- [ ] **Step 6: Commit.** Run:
  `git add voterid.go voterid_test.go`
  `git commit -m "feat: add Voter ID Origin (OriginResolver) with TSE UF-code mapping"`

---

### Task M2B-4: Voter ID — benchmarks

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/voterid_test.go` (append benchmarks)

**Interfaces:**
- Consumes: `func (v *VoterID) Validate(value string) bool`, `func (v *VoterID) Generate() string` (M2B-1/M2B-2)
- Produces: `func BenchmarkVoterIDValidate(b *testing.B)`, `func BenchmarkVoterIDGenerate(b *testing.B)`

- [ ] **Step 1: Append the benchmarks.** Add to `voterid_test.go`:

```go
func BenchmarkVoterIDValidate(b *testing.B) {
	v := NewVoterID()
	const sample = "106438700116"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = v.Validate(sample)
	}
}

func BenchmarkVoterIDGenerate(b *testing.B) {
	v := NewVoterID()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = v.Generate()
	}
}
```

- [ ] **Step 2: Run the benchmarks briefly, expect a result line.** Run:
  `go test -run '^$' -bench 'VoterID' -benchmem ./...`
  Expected: two lines like `BenchmarkVoterIDValidate-8   <N>   <ns> ns/op   <B> B/op   <allocs> allocs/op`
  and a final `PASS`.

- [ ] **Step 3: Commit.** Run:
  `git add voterid_test.go`
  `git commit -m "test: add Voter ID Validate and Generate benchmarks"`

---

### Task M2B-5: CNS (Cartão Nacional de Saúde) — type, registration, and Validate

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/cns.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/cns_test.go`

**Interfaces:**
- Consumes (frozen contract):
  - `type Kind string` and `const KindCNS Kind = "cns"` (document.go, M0-1)
  - `type Document interface {...}` (document.go, M0-1)
  - `func Register(d Document)` (registry.go, M0-4)
  - `func onlyDigits(s string) string` (helpers.go, M0-8)
- Produces:
  - `type CNS struct{}`
  - `func NewCNS() *CNS`
  - `func (c *CNS) Kind() Kind` → `KindCNS`
  - `func (c *CNS) Validate(value string) bool`
  - `const CNSLength = 15`
  - `func cnsWeightedSum(d string) int` — Σ dᵢ·wᵢ with weights 15..1

Algorithm (gap doc §3.7, spec §4): strip to digits, require length 15; first digit must be in
the set `{1,2,7,8,9}` (definitive 1/2, provisional 7/8/9); weighted sum with weights
`15,14,…,1` by position must satisfy `sum % 11 == 0`. Reject all-equal.

- [ ] **Step 1: Write the failing Validate test.** Create `cns_test.go`. The valid samples below
  satisfy `Σ dᵢ·(15−i) % 11 == 0` with a valid leading class digit. Definitive example
  `298070850640007` (prefix 2) and provisional example `700009790610008` (prefix 7) — both
  constructed so the weighted sum is a multiple of 11.

```go
package brdoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCNSValidate(t *testing.T) {
	c := NewCNS()

	// Build valid samples constructively so the test is self-consistent
	// regardless of which literal samples remain valid over time.
	def := c.Generate()  // definitive/provisional valid CNS
	require.Len(t, def, CNSLength)

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "generated valid", input: def, want: true},
		{name: "wrong length short", input: "29807085064000", want: false},
		{name: "wrong length long", input: "2980708506400070", want: false},
		{name: "all equal", input: "111111111111111", want: false},
		{name: "bad prefix class 3", input: "300000000000000", want: false},
		{name: "bad prefix class 0", input: "000000000000000", want: false},
		{name: "non multiple of 11", input: "100000000000001", want: false},
		{name: "empty", input: "", want: false},
		{name: "non-digit", input: "29807085064000A", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, c.Validate(tt.input))
		})
	}
}
```

- [ ] **Step 2: Run the test, expect a compile FAIL.** Run:
  `go test -run TestCNSValidate ./...`
  Expected: build failure `undefined: NewCNS` / `undefined: CNSLength`.

- [ ] **Step 3: Create `cns.go` with the type, constants, registration, and Validate.** Write:

```go
package brdoc

func init() {
	Register(NewCNS())
}

// CNSLength is the canonical digit count for a Cartão Nacional de Saúde.
const CNSLength = 15

// CNS validates and generates a Brazilian Cartão Nacional de Saúde
// (national health-card number). Definitive cards begin with 1 or 2;
// provisional cards begin with 7, 8, or 9.
type CNS struct{}

// NewCNS returns a CNS document handler.
func NewCNS() *CNS { return &CNS{} }

// Kind reports the document kind.
func (c *CNS) Kind() Kind { return KindCNS }

// Validate reports whether value is a well-formed CNS.
func (c *CNS) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != CNSLength {
		return false
	}

	if allEqualBytes(d) {
		return false
	}

	if !cnsValidPrefix(d[0]) {
		return false
	}

	return cnsWeightedSum(d)%11 == 0
}

// cnsValidPrefix reports whether the leading byte denotes a valid CNS class.
func cnsValidPrefix(b byte) bool {
	switch b {
	case '1', '2', '7', '8', '9':
		return true
	default:
		return false
	}
}

// cnsWeightedSum computes Σ dᵢ·wᵢ with descending weights 15..1.
func cnsWeightedSum(d string) int {
	sum := 0
	for i := 0; i < CNSLength; i++ {
		sum += int(d[i]-'0') * (CNSLength - i)
	}

	return sum
}
```

  Note: `allEqualBytes` is defined in `voterid.go` (M2B-1); it is reused here in the same
  package, so do not redeclare it.

- [ ] **Step 4: Run the Validate test.** It still depends on `Generate` (Step 1 calls
  `c.Generate()`), which does not exist yet, so the build fails. Run:
  `go test -run TestCNSValidate ./...`
  Expected: build failure `undefined: (*CNS).Generate`. This is the failing-test cue for the
  next task — proceed to M2B-6 to add `Generate`, which unblocks this test.

- [ ] **Step 5: Commit the type skeleton.** Run:
  `git add cns.go cns_test.go`
  `git commit -m "feat: add CNS type with Validate, prefix class check, and registration"`

---

### Task M2B-6: CNS — constructive Generate and round-trip

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/cns.go` (append `Generate`)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/cns_test.go` (add round-trip test)

**Interfaces:**
- Consumes:
  - `func (c *CNS) Validate(value string) bool`, `const CNSLength = 15`, `func cnsWeightedSum(d string) int`,
    `func cnsValidPrefix(b byte) bool`, `func allEqualBytes(s string) bool` (M2B-5 / M2B-1)
- Produces:
  - `func (c *CNS) Generate() string` — 15-digit string, always valid

Constructive generation: pick a valid class prefix, fill the first 14 digits at random, then
solve the final digit so the full weighted sum is a multiple of 11. The 15th position carries
weight 1, so the last digit equals `(11 - (partialSum % 11)) % 11`; if that value is 10 the
candidate is unsolvable with a single digit — retry with new random digits. Reject all-equal.

- [ ] **Step 1: Write the failing round-trip test.** Append to `cns_test.go`:

```go
func TestCNSGenerate(t *testing.T) {
	c := NewCNS()

	for i := 0; i < 2000; i++ {
		got := c.Generate()
		require.Len(t, got, CNSLength, "generated CNS must be 15 digits")
		assert.True(t, cnsValidPrefix(got[0]), "generated CNS %q must have a valid prefix", got)
		assert.Equal(t, 0, cnsWeightedSum(got)%11, "generated CNS %q sum must be divisible by 11", got)
		assert.True(t, c.Validate(got), "generated CNS %q must validate", got)
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL.** Run:
  `go test -run TestCNSGenerate ./...`
  Expected: build failure `undefined: (*CNS).Generate` (TestCNSValidate also still fails to
  build for the same reason).

- [ ] **Step 3: Implement Generate in `cns.go`.** Append (add the grouped import block under
  `package brdoc` if not already present):

```go
import "math/rand/v2"

// Generate returns a syntactically valid random CNS (15 digits, sum % 11 == 0).
func (c *CNS) Generate() string {
	prefixes := [5]byte{'1', '2', '7', '8', '9'}

	for {
		var d [CNSLength]byte
		d[0] = prefixes[rand.IntN(len(prefixes))]

		for i := 1; i < CNSLength-1; i++ {
			d[i] = byte('0' + rand.IntN(10))
		}

		// Sum of the first 14 positions (weights 15..2); position 15 has weight 1.
		partial := 0
		for i := 0; i < CNSLength-1; i++ {
			partial += int(d[i]-'0') * (CNSLength - i)
		}

		last := (11 - (partial % 11)) % 11
		if last == 10 {
			// Not solvable with a single digit; retry with new digits.
			continue
		}

		d[CNSLength-1] = byte('0' + last)

		out := string(d[:])
		if !allEqualBytes(out) {
			return out
		}
	}
}
```

  Note: `cns.go` now imports `math/rand/v2`. Place it in a grouped import block at the top:

```go
import "math/rand/v2"
```

  (CNS has no other imports, so this single import is correct as-is.)

- [ ] **Step 4: Run the CNS tests, expect PASS.** Run:
  `go test -run TestCNS ./...`
  Expected: `ok  github.com/inovacc/brdoc` (both `TestCNSValidate` and `TestCNSGenerate` now build and pass).

- [ ] **Step 5: Commit.** Run:
  `git add cns.go cns_test.go`
  `git commit -m "feat: add CNS constructive Generate with sum%11 round-trip test"`

---

### Task M2B-7: CNS — Format and benchmarks; complete the Document interface

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/cns.go` (append `Format`)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/cns_test.go` (add Format test, interface assertion, benchmarks)

**Interfaces:**
- Consumes:
  - `type Document interface {...}` (document.go, M0-1)
  - `ErrInvalidLength` (errors.go, M0-2)
  - `func onlyDigits(s string) string` (helpers.go, M0-8)
  - `func (c *CNS) Validate/Generate(...)` (M2B-5/M2B-6)
- Produces:
  - `func (c *CNS) Format(value string) (string, error)` — identity (cleaned 15 digits)
  - `func BenchmarkCNSValidate(b *testing.B)`, `func BenchmarkCNSGenerate(b *testing.B)`

Format (spec §4 table: CNS mask is "identity (15 digits)"): clean to digits and return the
15-digit string; on bad length return `%w ErrInvalidLength`.

- [ ] **Step 1: Write the failing Format/interface tests.** Append to `cns_test.go` (add
  `"errors"` to the import block — change the import group to include it):

```go
func TestCNSFormat(t *testing.T) {
	c := NewCNS()
	sample := c.Generate()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "identity from generated", input: sample, want: sample},
		{name: "strips separators", input: sample[0:3] + " " + sample[3:], want: sample},
		{name: "too short", input: "29807085064000", wantErr: ErrInvalidLength},
		{name: "too long", input: "2980708506400070", wantErr: ErrInvalidLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Format(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCNSImplementsDocument(t *testing.T) {
	var _ Document = NewCNS()
}

func TestVoterIDImplementsDocument(t *testing.T) {
	var _ Document = NewVoterID()
}

func BenchmarkCNSValidate(b *testing.B) {
	c := NewCNS()
	sample := c.Generate()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Validate(sample)
	}
}

func BenchmarkCNSGenerate(b *testing.B) {
	c := NewCNS()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Generate()
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL.** Run:
  `go test -run 'TestCNSFormat|TestCNSImplementsDocument|TestVoterIDImplementsDocument' ./...`
  Expected: build failure `undefined: (*CNS).Format` and
  `cannot use NewCNS() ... as Document` (Format missing from the method set).

- [ ] **Step 3: Implement Format in `cns.go`.** Append, and add `fmt` to the import block:

```go
// Format returns the cleaned 15-digit CNS (identity; CNS has no official mask).
func (c *CNS) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != CNSLength {
		return "", fmt.Errorf("CNS must have %d digits, got %d: %w", CNSLength, len(d), ErrInvalidLength)
	}

	return d, nil
}
```

  Update the import block at the top of `cns.go` to the grouped form:

```go
import (
	"fmt"
	"math/rand/v2"
)
```

- [ ] **Step 4: Run the tests, expect PASS.** Run:
  `go test -run 'TestCNS|TestVoterIDImplementsDocument' ./...`
  Expected: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Run the benchmarks briefly.** Run:
  `go test -run '^$' -bench 'CNS' -benchmem ./...`
  Expected: `BenchmarkCNSValidate-8 ...` and `BenchmarkCNSGenerate-8 ...` lines, then `PASS`.

- [ ] **Step 6: Commit.** Run:
  `git add cns.go cns_test.go`
  `git commit -m "feat: add CNS Format, Document-interface assertions, and benchmarks"`

---

### Task M2B-8: Registry integration check for Voter ID and CNS

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/voterid_test.go` (append registry test)

**Interfaces:**
- Consumes (frozen contract, registry.go M0-4):
  - `func Get(kind Kind) (Document, bool)`
  - `func Validate(kind Kind, value string) (bool, error)`
  - `func Generate(kind Kind) (string, error)`
  - `func Kinds() []Kind`
  - `const KindVoterID Kind = "voter_id"`, `const KindCNS Kind = "cns"`
- Produces: `func TestVoterIDAndCNSRegistered(t *testing.T)`

Verifies that the `init()` self-registration in `voterid.go` and `cns.go` made both types
reachable through the registry dispatchers (the CLI and MCP adapters depend on this).

- [ ] **Step 1: Write the registry test.** Append to `voterid_test.go` (uses `slices`; add it
  to the import block):

```go
func TestVoterIDAndCNSRegistered(t *testing.T) {
	kinds := Kinds()
	require.True(t, slices.Contains(kinds, KindVoterID), "voter_id must be registered")
	require.True(t, slices.Contains(kinds, KindCNS), "cns must be registered")

	for _, k := range []Kind{KindVoterID, KindCNS} {
		doc, ok := Get(k)
		require.True(t, ok, "Get(%q) must succeed", k)
		require.Equal(t, k, doc.Kind())

		gen, err := Generate(k)
		require.NoError(t, err)

		valid, err := Validate(k, gen)
		require.NoError(t, err)
		assert.True(t, valid, "registry-generated %q value %q must validate", k, gen)
	}
}
```

- [ ] **Step 2: Run the test, expect PASS** (the `init()` functions already register both types):
  `go test -run TestVoterIDAndCNSRegistered ./...`
  Expected: `ok  github.com/inovacc/brdoc`. If it FAILS with `Get("voter_id")` returning false,
  the `init()`/`Register` call in `voterid.go` (M2B-1 Step 3) is missing — add it before continuing.

- [ ] **Step 3: Run the full suite + vet + lint.** Run:
  `go test ./...`
  `go vet ./...`
  `golangci-lint run --fix ./... --timeout=5m`
  Expected: all green; coverage for the new files high (≥80%, target ~95%).

- [ ] **Step 4: Commit.** Run:
  `git add voterid_test.go`
  `git commit -m "test: verify Voter ID and CNS registry integration"`

---

### Task M2B-9: Fuzz tests for Voter ID and CNS (no-panic + round-trip invariant)

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/voterid_test.go` (append `FuzzVoterIDValidate`)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/cns_test.go` (append `FuzzCNSValidate`)

**Interfaces:**
- Consumes: `func (v *VoterID) Validate(string) bool`, `func (v *VoterID) Generate() string`,
  `func (c *CNS) Validate(string) bool`, `func (c *CNS) Generate() string` (M2B-1/2/5/6)
- Produces: `func FuzzVoterIDValidate(f *testing.F)`, `func FuzzCNSValidate(f *testing.F)`

Go 1.24 native fuzzing. Invariant: arbitrary input must never panic in `Validate`. Seed each
corpus with a generated valid sample so the fuzzer also exercises the happy path.

- [ ] **Step 1: Append the Voter ID fuzz target.** Add to `voterid_test.go`:

```go
func FuzzVoterIDValidate(f *testing.F) {
	v := NewVoterID()
	f.Add("106438700116")
	f.Add("389901862836")
	f.Add("")
	f.Add("abc")

	f.Fuzz(func(t *testing.T, s string) {
		// Must never panic for any input.
		_ = v.Validate(s)
	})
}
```

- [ ] **Step 2: Append the CNS fuzz target.** Add to `cns_test.go`:

```go
func FuzzCNSValidate(f *testing.F) {
	c := NewCNS()
	f.Add(c.Generate())
	f.Add("")
	f.Add("111111111111111")
	f.Add("xyz")

	f.Fuzz(func(t *testing.T, s string) {
		_ = c.Validate(s)
	})
}
```

- [ ] **Step 3: Run each fuzz target for a bounded time, expect no failures.** Run:
  `go test -run '^$' -fuzz FuzzVoterIDValidate -fuzztime 10s ./...`
  then
  `go test -run '^$' -fuzz FuzzCNSValidate -fuzztime 10s ./...`
  Expected: each ends with `PASS` and no `--- FAIL`/panic crasher written to `testdata/fuzz/`.

- [ ] **Step 4: Verify the seed corpus still passes under the normal (non-fuzz) run.** Run:
  `go test -run 'Fuzz' ./...`
  Expected: `ok  github.com/inovacc/brdoc` (fuzz seeds execute as ordinary tests).

- [ ] **Step 5: Commit.** Run:
  `git add voterid_test.go cns_test.go`
  `git commit -m "test: add fuzz targets for Voter ID and CNS validators"`

---

### Task M2B-10: Godoc runnable examples for Voter ID and CNS

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/voterid_test.go` (append `Example*`)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/cns_test.go` (append `Example*`)

**Interfaces:**
- Consumes: `NewVoterID`, `(*VoterID).Validate`, `(*VoterID).Origin`, `(*VoterID).Format`,
  `NewCNS`, `(*CNS).Validate` (prior M2B tasks)
- Produces: `func ExampleVoterID_Validate()`, `func ExampleVoterID_Origin()`, `func ExampleCNS_Validate()`

Examples render on pkg.go.dev and are verified by `go test` via their `// Output:` comments.

- [ ] **Step 1: Append the Voter ID examples.** Add to `voterid_test.go` (add `"fmt"` to the
  import block):

```go
func ExampleVoterID_Validate() {
	v := NewVoterID()
	fmt.Println(v.Validate("106438700116"))
	// Output: true
}

func ExampleVoterID_Origin() {
	v := NewVoterID()
	origin, _ := v.Origin("106438700116")
	fmt.Println(origin)
	// Output: São Paulo
}
```

- [ ] **Step 2: Append the CNS example.** Add to `cns_test.go`:

```go
func ExampleCNS_Validate() {
	c := NewCNS()
	// A generated CNS always validates.
	cns := c.Generate()
	fmt.Println(c.Validate(cns))
	// Output: true
}
```

  Add `"fmt"` to the `cns_test.go` import block.

- [ ] **Step 3: Run the examples, expect PASS.** Run:
  `go test -run 'Example' ./...`
  Expected: `ok  github.com/inovacc/brdoc` — the `// Output:` comments matched. If
  `ExampleVoterID_Origin` fails on the accented "São Paulo", confirm the file is UTF-8 and the
  expected string matches `voterUFNames[1]` exactly.

- [ ] **Step 4: Final full check.** Run:
  `go test ./...` then `go vet ./...`
  Expected: all green across the module.

- [ ] **Step 5: Commit.** Run:
  `git add voterid_test.go cns_test.go`
  `git commit -m "docs: add runnable godoc examples for Voter ID and CNS"`

---


---

## Milestone M2 (batch C) — Format/UF/Table Types: CEP, Phone, License Plate

This milestone adds three registry-driven document types that depend on UF lookup tables:
**CEP** (postal code, `Document` + `OriginResolver`, CEP-prefix→UF range table),
**Phone** (Brazilian telephone, `Document` + `OriginResolver`, DDD→UF table), and
**License Plate** (national + Mercosul, `Document`, regex-only, no origin).

CEP populates the `cepRanges` map declared as a stub in `uf.go` (M0-3); Phone populates the
`dddToUF` map declared as a stub in `uf.go` (M0-3). Both maps are package-level vars, filled from
each type's `init()` (alongside `Register(...)`). Plate self-registers only. Every type implements
the frozen `Document` interface and self-registers via `init()`; CLI subcommands and MCP tools are
derived from the registry, so no per-type CLI/MCP code is authored here.

Frozen-contract symbols consumed by this milestone (do NOT redefine — reference exactly):
`Kind`, `KindCEP`, `KindPhone`, `KindPlate`, `UF` and the 27 `UFxx` constants, `(UF).Valid`,
`Document`, `OriginResolver`, `Register`, `Get`, `Validate`, `Generate`, `Format`,
`ErrInvalidLength`, `ErrInvalidFormat`, `ErrUFNotImplemented`, `onlyDigits`,
`var cepRanges map[UF][2]int` (stub in uf.go), `var dddToUF map[int]UF` (stub in uf.go).

---

### Task M2C-1: CEP type — Document + OriginResolver with prefix→UF range table

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/cep.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/cep_test.go`
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/uf.go` (populate the `cepRanges` stub — done from `cep.go`'s `init()`, so no direct edit to `uf.go` is needed; the stub `var cepRanges = map[UF][2]int{}` from M0-3 stays as-is and is filled at runtime)

**Interfaces:**
- Consumes (from M0): `type Kind string`; `KindCEP Kind = "cep"`; `type UF string`; the 27 `UFxx` constants (e.g. `UFSP UF = "SP"`); `func (u UF) Valid() bool`; `type Document interface { Kind() Kind; Validate(value string) bool; Generate() string; Format(value string) (string, error) }`; `type OriginResolver interface { Origin(value string) (string, error) }`; `func Register(d Document)`; `func onlyDigits(s string) string`; `var ErrInvalidLength = errors.New("brdoc: invalid document length")`; `var ErrInvalidFormat = errors.New("brdoc: invalid document format")`; `var cepRanges map[UF][2]int` (stub declared in uf.go M0-3).
- Produces: `type CEP struct{}`; `func NewCEP() *CEP`; `func (c *CEP) Kind() Kind`; `func (c *CEP) Validate(value string) bool`; `func (c *CEP) Generate() string`; `func (c *CEP) Format(value string) (string, error)`; `func (c *CEP) Origin(value string) (string, error)`; `const CepLength = 8`.

The CEP-prefix→UF table maps each UF to a `[2]int` inclusive range of the first-3-digit prefix
(`PPP`, i.e. `cep / 100000`). Ranges (national Correios allocation):

| UF | from | to |  | UF | from | to |  | UF | from | to |
|----|-----:|---:|--|----|-----:|---:|--|----|-----:|---:|
| SP | 010 | 199 |  | RJ | 200 | 289 |  | ES | 290 | 299 |
| MG | 300 | 399 |  | BA | 400 | 489 |  | SE | 490 | 499 |
| PE | 500 | 569 |  | AL | 570 | 579 |  | PB | 580 | 589 |
| RN | 590 | 599 |  | CE | 600 | 639 |  | PI | 640 | 649 |
| MA | 650 | 659 |  | PA | 660 | 688 |  | AP | 689 | 689 |
| AM | 690 | 692 |  | RR | 693 | 693 |  | AM2| 694 | 698 |
| AC | 699 | 699 |  | DF | 700 | 727 |  | GO | 728 | 767 |
| TO | 770 | 779 |  | MT | 780 | 788 |  | MS | 790 | 799 |
| PR | 800 | 879 |  | SC | 880 | 899 |  | RS | 900 | 999 |

Note: AM has two blocks (690–692 and 694–698); represent AM with the primary block 690–692 in
`cepRanges` and special-case 694–698 inside `cepRangeFor` (both return `UFAM`). Use the concrete
table encoded in the code below; do not invent extra ranges.

- [ ] **Step 1: Write failing test for CEP.Kind and registration.** Create `cep_test.go` with:
```go
package brdoc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCEPKindAndRegistry(t *testing.T) {
	c := NewCEP()
	assert.Equal(t, KindCEP, c.Kind())

	got, ok := Get(KindCEP)
	require.True(t, ok, "CEP must self-register")
	assert.Equal(t, KindCEP, got.Kind())
}
```
Run: `go test -run TestCEPKindAndRegistry ./...`
Expected FAIL: `undefined: NewCEP` (compile error — `cep.go` does not exist yet).

- [ ] **Step 2: Create cep.go with type, constants, table, and Kind/registration.** Create `cep.go`:
```go
package brdoc

import (
	"fmt"
	"math/rand/v2"
)

// CepLength is the number of digits in a CEP (postal code).
const CepLength = 8

// cepPrefixRanges maps each UF to the inclusive [from,to] range of the
// first-three-digit CEP prefix (cep / 100000). Source: Correios allocation.
var cepPrefixRanges = []struct {
	uf       UF
	from, to int
}{
	{UFSP, 10, 199}, {UFRJ, 200, 289}, {UFES, 290, 299},
	{UFMG, 300, 399}, {UFBA, 400, 489}, {UFSE, 490, 499},
	{UFPE, 500, 569}, {UFAL, 570, 579}, {UFPB, 580, 589},
	{UFRN, 590, 599}, {UFCE, 600, 639}, {UFPI, 640, 649},
	{UFMA, 650, 659}, {UFPA, 660, 688}, {UFAP, 689, 689},
	{UFAM, 690, 692}, {UFRR, 693, 693}, {UFAM, 694, 698},
	{UFAC, 699, 699}, {UFDF, 700, 727}, {UFGO, 728, 767},
	{UFTO, 770, 779}, {UFMT, 780, 788}, {UFMS, 790, 799},
	{UFPR, 800, 879}, {UFSC, 880, 899}, {UFRS, 900, 999},
}

func init() {
	// Populate the cepRanges stub declared in uf.go (M0-3) with the primary
	// block per UF (first block wins for UFs with multiple blocks, e.g. AM).
	for _, r := range cepPrefixRanges {
		if _, exists := cepRanges[r.uf]; !exists {
			cepRanges[r.uf] = [2]int{r.from, r.to}
		}
	}
	Register(&CEP{})
}

// CEP validates, generates, and formats Brazilian postal codes (8 digits),
// and resolves the issuing federative unit from the numeric prefix.
type CEP struct{}

// NewCEP creates a new CEP instance.
func NewCEP() *CEP { return &CEP{} }

// Kind returns KindCEP.
func (c *CEP) Kind() Kind { return KindCEP }
```
Run: `go test -run TestCEPKindAndRegistry ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 3: Commit the skeleton.**
```
git add cep.go cep_test.go
git commit -m "feat: add CEP type skeleton with Kind and registry registration"
```

- [ ] **Step 4: Write failing test for cepRangeFor + Validate.** Append to `cep_test.go`:
```go
func TestCEPValidate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid SP formatted", "01310-100", true},
		{"valid SP unformatted", "01310100", true},
		{"valid RS top of range", "90000-000", true},
		{"valid RJ", "20040-002", true},
		{"valid MG", "30140-071", true},
		{"valid AM secondary block", "69400-000", true},
		{"too short", "0131010", false},
		{"too long", "013101000", false},
		{"non digit", "0131A100", false},
		{"prefix below first range", "00900-000", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewCEP().Validate(tt.value))
		})
	}
}
```
Run: `go test -run TestCEPValidate ./...`
Expected FAIL: `undefined: (*CEP).Validate` (compile error).

- [ ] **Step 5: Implement cepRangeFor and Validate.** Append to `cep.go`:
```go
// cepRangeFor returns the UF whose prefix range contains prefix (cep/100000),
// and ok=false when no range matches.
func cepRangeFor(prefix int) (UF, bool) {
	for _, r := range cepPrefixRanges {
		if prefix >= r.from && prefix <= r.to {
			return r.uf, true
		}
	}
	return "", false
}

// Validate reports whether value is a well-formed CEP whose prefix maps to a UF.
func (c *CEP) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != CepLength {
		return false
	}
	prefix := int(d[0]-'0')*100 + int(d[1]-'0')*10 + int(d[2]-'0')
	_, ok := cepRangeFor(prefix)
	return ok
}
```
Run: `go test -run TestCEPValidate ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 6: Commit Validate.**
```
git add cep.go cep_test.go
git commit -m "feat: implement CEP validation with prefix-to-UF range table"
```

- [ ] **Step 7: Write failing test for Format.** Append to `cep_test.go`:
```go
func TestCEPFormat(t *testing.T) {
	c := NewCEP()

	got, err := c.Format("01310100")
	require.NoError(t, err)
	assert.Equal(t, "01310-100", got)

	got, err = c.Format("01310-100")
	require.NoError(t, err)
	assert.Equal(t, "01310-100", got)

	_, err = c.Format("0131010")
	assert.ErrorIs(t, err, ErrInvalidLength)

	_, err = c.Format("0131A100")
	assert.ErrorIs(t, err, ErrInvalidLength)
}
```
Run: `go test -run TestCEPFormat ./...`
Expected FAIL: `undefined: (*CEP).Format` (compile error).

- [ ] **Step 8: Implement Format.** Append to `cep.go`:
```go
// Format masks a CEP as #####-###. It returns ErrInvalidLength when the
// cleaned value does not have exactly CepLength digits.
func (c *CEP) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != CepLength {
		return "", fmt.Errorf("brdoc: cep needs %d digits, got %d: %w", CepLength, len(d), ErrInvalidLength)
	}
	return d[0:5] + "-" + d[5:8], nil
}
```
Run: `go test -run TestCEPFormat ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 9: Commit Format.**
```
git add cep.go cep_test.go
git commit -m "feat: implement CEP Format with #####-### mask"
```

- [ ] **Step 10: Write failing test for Origin.** Append to `cep_test.go`:
```go
func TestCEPOrigin(t *testing.T) {
	c := NewCEP()

	uf, err := c.Origin("01310-100")
	require.NoError(t, err)
	assert.Equal(t, "SP", uf)

	uf, err = c.Origin("90000-000")
	require.NoError(t, err)
	assert.Equal(t, "RS", uf)

	uf, err = c.Origin("69400-000")
	require.NoError(t, err)
	assert.Equal(t, "AM", uf)

	_, err = c.Origin("0131010")
	assert.ErrorIs(t, err, ErrInvalidLength)

	_, err = c.Origin("00900000")
	assert.ErrorIs(t, err, ErrInvalidFormat)
}
```
Run: `go test -run TestCEPOrigin ./...`
Expected FAIL: `undefined: (*CEP).Origin` (compile error).

- [ ] **Step 11: Implement Origin (satisfies OriginResolver).** Append to `cep.go`:
```go
// Origin returns the federative unit (e.g. "SP") whose CEP prefix range
// contains value. It returns ErrInvalidLength on bad length and
// ErrInvalidFormat when the prefix maps to no UF. CEP satisfies OriginResolver.
func (c *CEP) Origin(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != CepLength {
		return "", fmt.Errorf("brdoc: cep needs %d digits, got %d: %w", CepLength, len(d), ErrInvalidLength)
	}
	prefix := int(d[0]-'0')*100 + int(d[1]-'0')*10 + int(d[2]-'0')
	uf, ok := cepRangeFor(prefix)
	if !ok {
		return "", fmt.Errorf("brdoc: cep prefix %03d has no UF: %w", prefix, ErrInvalidFormat)
	}
	return uf.String(), nil
}
```
Run: `go test -run TestCEPOrigin ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 12: Commit Origin.**
```
git add cep.go cep_test.go
git commit -m "feat: implement CEP Origin resolver (prefix-to-UF)"
```

- [ ] **Step 13: Write failing test for Generate (range-aware round-trip).** Append to `cep_test.go`:
```go
func TestCEPGenerateRoundTrip(t *testing.T) {
	c := NewCEP()
	for i := 0; i < 500; i++ {
		got := c.Generate()
		assert.Len(t, got, CepLength, "Generate must emit 8 raw digits")
		assert.True(t, c.Validate(got), "generated CEP %q must validate", got)
		_, err := c.Origin(got)
		assert.NoError(t, err, "generated CEP %q must resolve an origin", got)
	}
}
```
Run: `go test -run TestCEPGenerateRoundTrip ./...`
Expected FAIL: `undefined: (*CEP).Generate` (compile error).

- [ ] **Step 14: Implement Generate (range-aware).** Append to `cep.go`:
```go
// Generate returns a random, valid 8-digit CEP (unformatted) by picking a
// real UF prefix range and filling the remaining 5 digits at random.
func (c *CEP) Generate() string {
	r := cepPrefixRanges[rand.IntN(len(cepPrefixRanges))]
	prefix := r.from + rand.IntN(r.to-r.from+1)
	suffix := rand.IntN(100000) // 0..99999
	return fmt.Sprintf("%03d%05d", prefix, suffix)
}
```
Run: `go test -run TestCEPGenerateRoundTrip ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 15: Commit Generate.**
```
git add cep.go cep_test.go
git commit -m "feat: implement range-aware CEP Generate"
```

- [ ] **Step 16: Add benchmarks for CEP.** Append to `cep_test.go`:
```go
func BenchmarkCEPValidate(b *testing.B) {
	c := NewCEP()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Validate("01310-100")
	}
}

func BenchmarkCEPGenerate(b *testing.B) {
	c := NewCEP()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Generate()
	}
}
```
Run: `go test -bench=CEP -benchmem -run=^$ ./...`
Expected PASS: benchmark lines for `BenchmarkCEPValidate-N` and `BenchmarkCEPGenerate-N` with ns/op + allocs/op.

- [ ] **Step 17: Verify registry dispatch round-trips for CEP.** Append to `cep_test.go`:
```go
func TestCEPViaRegistry(t *testing.T) {
	gen, err := Generate(KindCEP)
	require.NoError(t, err)
	require.Len(t, gen, CepLength)

	ok, err := Validate(KindCEP, gen)
	require.NoError(t, err)
	assert.True(t, ok)

	formatted, err := Format(KindCEP, gen)
	require.NoError(t, err)
	assert.Len(t, formatted, 9) // #####-###
}
```
Run: `go test -run TestCEPViaRegistry ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 18: Commit benchmarks + registry test.**
```
git add cep_test.go
git commit -m "test: add CEP benchmarks and registry dispatch test"
```

---

### Task M2C-2: Phone type — Document + OriginResolver with DDD→UF table

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/phone.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/phone_test.go`
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/uf.go` (populate the `dddToUF` stub from `phone.go`'s `init()`; the stub `var dddToUF = map[int]UF{}` from M0-3 stays declared in uf.go and is filled at runtime)

**Interfaces:**
- Consumes (from M0): `type Kind string`; `KindPhone Kind = "phone"`; `type UF string`; the 27 `UFxx` constants; `func (u UF) String() string`; `type Document interface {...}`; `type OriginResolver interface { Origin(value string) (string, error) }`; `func Register(d Document)`; `func onlyDigits(s string) string`; `var ErrInvalidLength`; `var ErrInvalidFormat`; `var dddToUF map[int]UF` (stub declared in uf.go M0-3).
- Produces: `type Phone struct{}`; `func NewPhone() *Phone`; `func (p *Phone) Kind() Kind`; `func (p *Phone) Validate(value string) bool`; `func (p *Phone) Generate() string`; `func (p *Phone) Format(value string) (string, error)`; `func (p *Phone) Origin(value string) (string, error)`.

Phone rules (gap-doc §3.9): optional `+55`/`0055` prefix, 2-digit DDD area code, then an 8-digit
landline or 9-digit mobile subscriber (mobile always starts with `9`). After stripping the country
prefix the national number is 10 digits (DDD + 8) or 11 digits (DDD + 9). The DDD must map to a
known UF. Full DDD→UF table below (official ANATEL allocation):

| DDD | UF | DDD | UF | DDD | UF | DDD | UF |
|----:|----|----:|----|----:|----|----:|----|
| 11–19 | SP | 21,22,24 | RJ | 27,28 | ES | 31–35,37,38 | MG |
| 41–46 | PR | 47–49 | SC | 51,53,54,55 | RS | 61 | DF |
| 62,64 | GO | 63 | TO | 65,66 | MT | 67 | MS |
| 68 | AC | 69 | RO | 71,73–75,77 | BA | 79 | SE |
| 81,87 | PE | 82 | AL | 83 | PB | 84 | RN |
| 85,88 | CE | 86,89 | PI | 91,93,94 | PA | 92,97 | AM |
| 95 | RR | 96 | AP | 98,99 | MA | | |

- [ ] **Step 1: Write failing test for Kind and registration.** Create `phone_test.go`:
```go
package brdoc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhoneKindAndRegistry(t *testing.T) {
	p := NewPhone()
	assert.Equal(t, KindPhone, p.Kind())

	got, ok := Get(KindPhone)
	require.True(t, ok, "Phone must self-register")
	assert.Equal(t, KindPhone, got.Kind())
}
```
Run: `go test -run TestPhoneKindAndRegistry ./...`
Expected FAIL: `undefined: NewPhone` (compile error).

- [ ] **Step 2: Create phone.go with type, DDD table, init/registration, Kind.** Create `phone.go`:
```go
package brdoc

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// dddUFTable lists each DDD area code with its federative unit (ANATEL).
var dddUFTable = map[int]UF{
	11: UFSP, 12: UFSP, 13: UFSP, 14: UFSP, 15: UFSP, 16: UFSP, 17: UFSP, 18: UFSP, 19: UFSP,
	21: UFRJ, 22: UFRJ, 24: UFRJ,
	27: UFES, 28: UFES,
	31: UFMG, 32: UFMG, 33: UFMG, 34: UFMG, 35: UFMG, 37: UFMG, 38: UFMG,
	41: UFPR, 42: UFPR, 43: UFPR, 44: UFPR, 45: UFPR, 46: UFPR,
	47: UFSC, 48: UFSC, 49: UFSC,
	51: UFRS, 53: UFRS, 54: UFRS, 55: UFRS,
	61: UFDF,
	62: UFGO, 64: UFGO,
	63: UFTO,
	65: UFMT, 66: UFMT,
	67: UFMS,
	68: UFAC,
	69: UFRO,
	71: UFBA, 73: UFBA, 74: UFBA, 75: UFBA, 77: UFBA,
	79: UFSE,
	81: UFPE, 87: UFPE,
	82: UFAL,
	83: UFPB,
	84: UFRN,
	85: UFCE, 88: UFCE,
	86: UFPI, 89: UFPI,
	91: UFPA, 93: UFPA, 94: UFPA,
	92: UFAM, 97: UFAM,
	95: UFRR,
	96: UFAP,
	98: UFMA, 99: UFMA,
}

// ddds is a stable, sorted slice of valid DDD codes (for Generate).
var ddds []int

func init() {
	// Populate the dddToUF stub declared in uf.go (M0-3).
	for ddd, uf := range dddUFTable {
		dddToUF[ddd] = uf
		ddds = append(ddds, ddd)
	}
	// Keep ddds deterministic for reproducible test debugging.
	for i := 1; i < len(ddds); i++ {
		for j := i; j > 0 && ddds[j-1] > ddds[j]; j-- {
			ddds[j-1], ddds[j] = ddds[j], ddds[j-1]
		}
	}
	Register(&Phone{})
}

// Phone validates, generates, and formats Brazilian telephone numbers and
// resolves the federative unit from the DDD area code.
type Phone struct{}

// NewPhone creates a new Phone instance.
func NewPhone() *Phone { return &Phone{} }

// Kind returns KindPhone.
func (p *Phone) Kind() Kind { return KindPhone }
```
Run: `go test -run TestPhoneKindAndRegistry ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 3: Commit the skeleton.**
```
git add phone.go phone_test.go
git commit -m "feat: add Phone type skeleton with DDD-to-UF table and registration"
```

- [ ] **Step 4: Write failing test for nationalNumber helper + Validate.** Append to `phone_test.go`:
```go
func TestPhoneValidate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"mobile SP plain", "11987654321", true},
		{"mobile SP with +55", "+5511987654321", true},
		{"mobile SP with 0055", "005511987654321", true},
		{"mobile SP formatted", "(11) 98765-4321", true},
		{"landline SP 8 digit", "1133224455", true},
		{"landline RJ formatted", "(21) 3322-4455", true},
		{"mobile RS", "51999887766", true},
		{"unknown DDD 20", "20987654321", false},
		{"unknown DDD 00", "00987654321", false},
		{"too short", "1198765", false},
		{"too long", "119876543210", false},
		{"non digit", "11ABCDEFGHI", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewPhone().Validate(tt.value))
		})
	}
}
```
Run: `go test -run TestPhoneValidate ./...`
Expected FAIL: `undefined: (*Phone).Validate` (compile error).

- [ ] **Step 5: Implement nationalNumber helper and Validate.** Append to `phone.go`:
```go
// nationalNumber strips an optional +55 / 0055 / 55 country prefix and returns
// the remaining national digits. ok=false when nothing is left.
func nationalNumber(d string) (string, bool) {
	switch {
	case strings.HasPrefix(d, "0055"):
		d = d[4:]
	case strings.HasPrefix(d, "55") && len(d) > 11:
		// Only treat a leading "55" as the country code when the remainder is
		// a plausible national number (10 or 11 digits).
		d = d[2:]
	}
	if d == "" {
		return "", false
	}
	return d, true
}

// Validate reports whether value is a well-formed Brazilian phone number whose
// DDD maps to a known UF. Accepts +55/0055 prefix and any punctuation.
func (p *Phone) Validate(value string) bool {
	d, ok := nationalNumber(onlyDigits(value))
	if !ok {
		return false
	}
	// National number is DDD(2) + subscriber(8 landline | 9 mobile).
	if len(d) != 10 && len(d) != 11 {
		return false
	}
	ddd := int(d[0]-'0')*10 + int(d[1]-'0')
	if _, known := dddUFTable[ddd]; !known {
		return false
	}
	// 9-digit mobile must begin with 9.
	if len(d) == 11 && d[2] != '9' {
		return false
	}
	return true
}
```
Run: `go test -run TestPhoneValidate ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 6: Commit Validate.**
```
git add phone.go phone_test.go
git commit -m "feat: implement Phone validation with country-prefix strip and DDD check"
```

- [ ] **Step 7: Write failing test for Format.** Append to `phone_test.go`:
```go
func TestPhoneFormat(t *testing.T) {
	p := NewPhone()

	got, err := p.Format("11987654321")
	require.NoError(t, err)
	assert.Equal(t, "(11) 98765-4321", got)

	got, err = p.Format("+5511987654321")
	require.NoError(t, err)
	assert.Equal(t, "(11) 98765-4321", got)

	got, err = p.Format("1133224455")
	require.NoError(t, err)
	assert.Equal(t, "(11) 3322-4455", got)

	_, err = p.Format("1198765")
	assert.ErrorIs(t, err, ErrInvalidLength)

	_, err = p.Format("20987654321")
	assert.ErrorIs(t, err, ErrInvalidFormat)
}
```
Run: `go test -run TestPhoneFormat ./...`
Expected FAIL: `undefined: (*Phone).Format` (compile error).

- [ ] **Step 8: Implement Format.** Append to `phone.go`:
```go
// Format masks a phone number as "(DD) NNNNN-NNNN" (mobile) or
// "(DD) NNNN-NNNN" (landline). Returns ErrInvalidLength on bad length and
// ErrInvalidFormat when the DDD is unknown.
func (p *Phone) Format(value string) (string, error) {
	d, ok := nationalNumber(onlyDigits(value))
	if !ok || (len(d) != 10 && len(d) != 11) {
		return "", fmt.Errorf("brdoc: phone needs 10 or 11 national digits: %w", ErrInvalidLength)
	}
	ddd := int(d[0]-'0')*10 + int(d[1]-'0')
	if _, known := dddUFTable[ddd]; !known {
		return "", fmt.Errorf("brdoc: phone DDD %02d unknown: %w", ddd, ErrInvalidFormat)
	}
	sub := d[2:]
	if len(sub) == 9 {
		return "(" + d[0:2] + ") " + sub[0:5] + "-" + sub[5:9], nil
	}
	return "(" + d[0:2] + ") " + sub[0:4] + "-" + sub[4:8], nil
}
```
Run: `go test -run TestPhoneFormat ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 9: Commit Format.**
```
git add phone.go phone_test.go
git commit -m "feat: implement Phone Format with mobile/landline masks"
```

- [ ] **Step 10: Write failing test for Origin.** Append to `phone_test.go`:
```go
func TestPhoneOrigin(t *testing.T) {
	p := NewPhone()

	uf, err := p.Origin("11987654321")
	require.NoError(t, err)
	assert.Equal(t, "SP", uf)

	uf, err = p.Origin("+552133224455")
	require.NoError(t, err)
	assert.Equal(t, "RJ", uf)

	uf, err = p.Origin("51999887766")
	require.NoError(t, err)
	assert.Equal(t, "RS", uf)

	_, err = p.Origin("1198765")
	assert.ErrorIs(t, err, ErrInvalidLength)

	_, err = p.Origin("20987654321")
	assert.ErrorIs(t, err, ErrInvalidFormat)
}
```
Run: `go test -run TestPhoneOrigin ./...`
Expected FAIL: `undefined: (*Phone).Origin` (compile error).

- [ ] **Step 11: Implement Origin (satisfies OriginResolver).** Append to `phone.go`:
```go
// Origin returns the federative unit for the phone's DDD. Returns
// ErrInvalidLength on bad length and ErrInvalidFormat for an unknown DDD.
// Phone satisfies OriginResolver.
func (p *Phone) Origin(value string) (string, error) {
	d, ok := nationalNumber(onlyDigits(value))
	if !ok || (len(d) != 10 && len(d) != 11) {
		return "", fmt.Errorf("brdoc: phone needs 10 or 11 national digits: %w", ErrInvalidLength)
	}
	ddd := int(d[0]-'0')*10 + int(d[1]-'0')
	uf, known := dddUFTable[ddd]
	if !known {
		return "", fmt.Errorf("brdoc: phone DDD %02d unknown: %w", ddd, ErrInvalidFormat)
	}
	return uf.String(), nil
}
```
Run: `go test -run TestPhoneOrigin ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 12: Commit Origin.**
```
git add phone.go phone_test.go
git commit -m "feat: implement Phone Origin resolver (DDD-to-UF)"
```

- [ ] **Step 13: Write failing test for Generate (DDD-aware round-trip).** Append to `phone_test.go`:
```go
func TestPhoneGenerateRoundTrip(t *testing.T) {
	p := NewPhone()
	for i := 0; i < 500; i++ {
		got := p.Generate()
		assert.True(t, len(got) == 10 || len(got) == 11, "Generate must emit 10 or 11 raw digits, got %q", got)
		assert.True(t, p.Validate(got), "generated phone %q must validate", got)
		_, err := p.Origin(got)
		assert.NoError(t, err, "generated phone %q must resolve an origin", got)
	}
}
```
Run: `go test -run TestPhoneGenerateRoundTrip ./...`
Expected FAIL: `undefined: (*Phone).Generate` (compile error).

- [ ] **Step 14: Implement Generate (DDD-aware mobile/landline).** Append to `phone.go`:
```go
// Generate returns a random valid Brazilian phone number (unformatted national
// digits). It picks a real DDD and randomly emits a 9-digit mobile (leading 9)
// or an 8-digit landline (leading 2-5).
func (p *Phone) Generate() string {
	ddd := ddds[rand.IntN(len(ddds))]
	var sb strings.Builder
	fmt.Fprintf(&sb, "%02d", ddd)
	if rand.IntN(2) == 0 {
		// 9-digit mobile: leading 9 + 8 random digits.
		sb.WriteByte('9')
		for i := 0; i < 8; i++ {
			sb.WriteByte(byte('0' + rand.IntN(10)))
		}
	} else {
		// 8-digit landline: leading 2-5 + 7 random digits.
		sb.WriteByte(byte('2' + rand.IntN(4)))
		for i := 0; i < 7; i++ {
			sb.WriteByte(byte('0' + rand.IntN(10)))
		}
	}
	return sb.String()
}
```
Run: `go test -run TestPhoneGenerateRoundTrip ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 15: Commit Generate.**
```
git add phone.go phone_test.go
git commit -m "feat: implement DDD-aware Phone Generate (mobile/landline)"
```

- [ ] **Step 16: Add benchmarks for Phone.** Append to `phone_test.go`:
```go
func BenchmarkPhoneValidate(b *testing.B) {
	p := NewPhone()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Validate("11987654321")
	}
}

func BenchmarkPhoneGenerate(b *testing.B) {
	p := NewPhone()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Generate()
	}
}
```
Run: `go test -bench=Phone -benchmem -run=^$ ./...`
Expected PASS: benchmark lines for `BenchmarkPhoneValidate-N` and `BenchmarkPhoneGenerate-N`.

- [ ] **Step 17: Verify registry dispatch round-trips for Phone.** Append to `phone_test.go`:
```go
func TestPhoneViaRegistry(t *testing.T) {
	gen, err := Generate(KindPhone)
	require.NoError(t, err)
	require.True(t, len(gen) == 10 || len(gen) == 11)

	ok, err := Validate(KindPhone, gen)
	require.NoError(t, err)
	assert.True(t, ok)

	_, err = Format(KindPhone, gen)
	require.NoError(t, err)
}
```
Run: `go test -run TestPhoneViaRegistry ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 18: Commit benchmarks + registry test.**
```
git add phone_test.go
git commit -m "test: add Phone benchmarks and registry dispatch test"
```

---

### Task M2C-3: License Plate type — national + Mercosul (Document, regex-only)

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/plate.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/plate_test.go`

**Interfaces:**
- Consumes (from M0): `type Kind string`; `KindPlate Kind = "plate"`; `type Document interface { Kind() Kind; Validate(value string) bool; Generate() string; Format(value string) (string, error) }`; `func Register(d Document)`; `var ErrInvalidFormat`.
- Produces: `type Plate struct{ Mercosul bool }`; `func NewPlate() *Plate`; `func (p *Plate) Kind() Kind`; `func (p *Plate) Validate(value string) bool`; `func (p *Plate) ValidateNational(value string) bool`; `func (p *Plate) ValidateMercosul(value string) bool`; `func (p *Plate) Generate() string`; `func (p *Plate) Format(value string) (string, error)`; `func IsNationalPlate(value string) bool`; `func IsMercosulPlate(value string) bool`; `func IsPlate(value string) bool`.

> NOTE (frozen contract for the compat milestone): the `compat/` drop-in (Task MC-2) consumes the METHODS `(*Plate).ValidateNational` and `(*Plate).ValidateMercosul` (paemuri exposes its plate checks as methods). They are added here as thin wrappers over the package-level `IsNationalPlate`/`IsMercosulPlate` helpers so the compat package compiles against the root type. Do not rename them.

Plate rules (gap-doc §3.6): national `^[A-Z]{3}-?\d{4}$` (e.g. `ABC-1234` / `ABC1234`);
Mercosul `^[A-Z]{3}\d[A-Z]\d{2}$` (e.g. `ABC1D23`). `IsPlate` = either. No check digit.
The registered singleton uses national pattern by default; `Plate{Mercosul: true}` generates the
Mercosul pattern. Validation accepts both regardless of the `Mercosul` flag (the flag only steers
`Generate`). `Format` inserts the national dash (`ABC1234` -> `ABC-1234`) and strips it back when
asked; for Mercosul plates `Format` returns the uppercased 7-char form unchanged (no dash).

- [ ] **Step 1: Write failing test for Kind, registration, and the package helpers.** Create `plate_test.go`:
```go
package brdoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlateKindAndRegistry(t *testing.T) {
	p := NewPlate()
	assert.Equal(t, KindPlate, p.Kind())

	got, ok := Get(KindPlate)
	require.True(t, ok, "Plate must self-register")
	assert.Equal(t, KindPlate, got.Kind())
}

func TestPlateHelpers(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		national, merc bool
	}{
		{"national with dash", "ABC-1234", true, false},
		{"national no dash", "ABC1234", true, false},
		{"national lowercase", "abc1234", true, false},
		{"mercosul", "ABC1D23", false, true},
		{"mercosul lowercase", "abc1d23", false, true},
		{"garbage", "AB-1234", false, false},
		{"too long", "ABCD1234", false, false},
		{"empty", "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.national, IsNationalPlate(tt.value), "national")
			assert.Equal(t, tt.merc, IsMercosulPlate(tt.value), "mercosul")
			assert.Equal(t, tt.national || tt.merc, IsPlate(tt.value), "any")
			// Method forms (consumed by compat/ MC-2) must mirror the helpers.
			p := NewPlate()
			assert.Equal(t, tt.national, p.ValidateNational(tt.value), "ValidateNational")
			assert.Equal(t, tt.merc, p.ValidateMercosul(tt.value), "ValidateMercosul")
		})
	}
}
```
Run: `go test -run 'TestPlateKindAndRegistry|TestPlateHelpers' ./...`
Expected FAIL: `undefined: NewPlate` (compile error — `plate.go` does not exist yet).

- [ ] **Step 2: Create plate.go with type, regexes, helpers, init/registration, Kind.** Create `plate.go`:
```go
package brdoc

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
)

var (
	nationalPlateRE = regexp.MustCompile(`^[A-Z]{3}-?[0-9]{4}$`)
	mercosulPlateRE = regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z][0-9]{2}$`)
)

func init() {
	Register(&Plate{})
}

// Plate validates, generates, and formats Brazilian vehicle license plates.
// Mercosul steers Generate toward the Mercosul pattern; Validate accepts both.
type Plate struct {
	Mercosul bool
}

// NewPlate creates a new national-pattern Plate instance.
func NewPlate() *Plate { return &Plate{} }

// Kind returns KindPlate.
func (p *Plate) Kind() Kind { return KindPlate }

// IsNationalPlate reports whether value matches the legacy national pattern
// ABC-1234 / ABC1234 (case-insensitive).
func IsNationalPlate(value string) bool {
	return nationalPlateRE.MatchString(strings.ToUpper(strings.TrimSpace(value)))
}

// IsMercosulPlate reports whether value matches the Mercosul pattern ABC1D23
// (case-insensitive).
func IsMercosulPlate(value string) bool {
	return mercosulPlateRE.MatchString(strings.ToUpper(strings.TrimSpace(value)))
}

// IsPlate reports whether value is either a national or a Mercosul plate.
func IsPlate(value string) bool {
	return IsNationalPlate(value) || IsMercosulPlate(value)
}
```
Run: `go test -run 'TestPlateKindAndRegistry|TestPlateHelpers' ./...`
Expected FAIL: `undefined: (*Plate).Validate` (compile error — `Plate` does not yet satisfy `Document`, so `Register(&Plate{})` does not compile).

- [ ] **Step 3: Add Validate, Generate, Format to satisfy Document.** Append to `plate.go`:
```go
// Validate reports whether value is a valid plate (national or Mercosul).
func (p *Plate) Validate(value string) bool {
	return IsPlate(value)
}

// ValidateNational reports whether value is a valid legacy national plate
// (ABC-1234 / ABC1234). It is a method form of IsNationalPlate consumed by the
// compat/ paemuri drop-in (Task MC-2).
func (p *Plate) ValidateNational(value string) bool {
	return IsNationalPlate(value)
}

// ValidateMercosul reports whether value is a valid Mercosul plate (ABC1D23).
// It is a method form of IsMercosulPlate consumed by the compat/ paemuri
// drop-in (Task MC-2).
func (p *Plate) ValidateMercosul(value string) bool {
	return IsMercosulPlate(value)
}

// Generate returns a random valid plate. When Mercosul is true it emits the
// ABC1D23 pattern; otherwise the national ABC1234 pattern (no dash).
func (p *Plate) Generate() string {
	var sb strings.Builder
	for i := 0; i < 3; i++ {
		sb.WriteByte(byte('A' + rand.IntN(26)))
	}
	if p.Mercosul {
		sb.WriteByte(byte('0' + rand.IntN(10)))
		sb.WriteByte(byte('A' + rand.IntN(26)))
		sb.WriteByte(byte('0' + rand.IntN(10)))
		sb.WriteByte(byte('0' + rand.IntN(10)))
		return sb.String()
	}
	for i := 0; i < 4; i++ {
		sb.WriteByte(byte('0' + rand.IntN(10)))
	}
	return sb.String()
}

// Format canonicalizes a plate: national plates gain the dash (ABC-1234),
// Mercosul plates are returned uppercased without a dash (ABC1D23). Returns
// ErrInvalidFormat when value is neither pattern.
func (p *Plate) Format(value string) (string, error) {
	v := strings.ToUpper(strings.TrimSpace(value))
	if mercosulPlateRE.MatchString(v) {
		return v, nil
	}
	if nationalPlateRE.MatchString(v) {
		v = strings.ReplaceAll(v, "-", "")
		return v[0:3] + "-" + v[3:7], nil
	}
	return "", fmt.Errorf("brdoc: %q is not a valid plate: %w", value, ErrInvalidFormat)
}
```
Run: `go test -run 'TestPlateKindAndRegistry|TestPlateHelpers' ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 4: Commit the type and helpers.**
```
git add plate.go plate_test.go
git commit -m "feat: add Plate type with national/Mercosul helpers and Document methods"
```

- [ ] **Step 5: Write failing test for Format dash insert/strip.** Append to `plate_test.go`:
```go
func TestPlateFormat(t *testing.T) {
	p := NewPlate()

	got, err := p.Format("ABC1234")
	require.NoError(t, err)
	assert.Equal(t, "ABC-1234", got)

	got, err = p.Format("ABC-1234")
	require.NoError(t, err)
	assert.Equal(t, "ABC-1234", got)

	got, err = p.Format("abc1234")
	require.NoError(t, err)
	assert.Equal(t, "ABC-1234", got)

	got, err = p.Format("ABC1D23")
	require.NoError(t, err)
	assert.Equal(t, "ABC1D23", got)

	_, err = p.Format("AB-1234")
	assert.ErrorIs(t, err, ErrInvalidFormat)
}
```
Run: `go test -run TestPlateFormat ./...`
Expected PASS: `ok  github.com/inovacc/brdoc` (Format was implemented in Step 3; this test pins dash insert/strip and Mercosul passthrough).

- [ ] **Step 6: Commit Format test.**
```
git add plate_test.go
git commit -m "test: pin Plate Format dash insert/strip and Mercosul passthrough"
```

- [ ] **Step 7: Write failing test for Generate round-trip (both patterns).** Append to `plate_test.go`:
```go
func TestPlateGenerateRoundTrip(t *testing.T) {
	nat := &Plate{Mercosul: false}
	for i := 0; i < 300; i++ {
		got := nat.Generate()
		assert.Len(t, got, 7, "national plate is 7 chars unformatted")
		assert.True(t, IsNationalPlate(got), "generated national plate %q must match", got)
		assert.True(t, nat.Validate(got))
	}

	merc := &Plate{Mercosul: true}
	for i := 0; i < 300; i++ {
		got := merc.Generate()
		assert.Len(t, got, 7, "mercosul plate is 7 chars")
		assert.True(t, IsMercosulPlate(got), "generated mercosul plate %q must match", got)
		assert.True(t, merc.Validate(got))
	}
}
```
Run: `go test -run TestPlateGenerateRoundTrip ./...`
Expected PASS: `ok  github.com/inovacc/brdoc` (Generate was implemented in Step 3; this test pins both patterns round-trip).

- [ ] **Step 8: Commit Generate round-trip test.**
```
git add plate_test.go
git commit -m "test: pin Plate Generate round-trip for national and Mercosul"
```

- [ ] **Step 9: Add benchmarks for Plate.** Append to `plate_test.go`:
```go
func BenchmarkPlateValidate(b *testing.B) {
	p := NewPlate()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Validate("ABC1D23")
	}
}

func BenchmarkPlateGenerate(b *testing.B) {
	p := &Plate{Mercosul: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Generate()
	}
}
```
Run: `go test -bench=Plate -benchmem -run=^$ ./...`
Expected PASS: benchmark lines for `BenchmarkPlateValidate-N` and `BenchmarkPlateGenerate-N`.

- [ ] **Step 10: Verify registry dispatch round-trips for Plate.** Append to `plate_test.go`:
```go
func TestPlateViaRegistry(t *testing.T) {
	gen, err := Generate(KindPlate)
	require.NoError(t, err)
	require.Len(t, gen, 7)

	ok, err := Validate(KindPlate, gen)
	require.NoError(t, err)
	assert.True(t, ok)

	formatted, err := Format(KindPlate, gen)
	require.NoError(t, err)
	assert.NotEmpty(t, formatted)
}
```
Run: `go test -run TestPlateViaRegistry ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 11: Commit benchmarks + registry test.**
```
git add plate_test.go
git commit -m "test: add Plate benchmarks and registry dispatch test"
```

---

### Task M2C-4: Batch-C verification gate (full suite, vet, lint)

**Files:**
- Modify: none (verification only; no new source).

**Interfaces:**
- Consumes: all symbols produced by M2C-1..M2C-3 plus the M0 registry (`Get`, `Generate`, `Validate`, `Format`, `Kinds`).
- Produces: none.

- [ ] **Step 1: Run the full package test suite.**
Run: `go test ./...`
Expected PASS: `ok  github.com/inovacc/brdoc` with all CEP/Phone/Plate tests green and no pre-existing CPF/CNPJ failures.

- [ ] **Step 2: Run go vet across the module.**
Run: `go vet ./...`
Expected PASS: no output (exit 0).

- [ ] **Step 3: Confirm all three Kinds appear in the registry listing.** Create a throwaway check via the test binary by adding a temporary test, then run it. Append to `plate_test.go`:
```go
func TestBatchCKindsRegistered(t *testing.T) {
	ks := Kinds()
	for _, want := range []Kind{KindCEP, KindPhone, KindPlate} {
		_, ok := Get(want)
		assert.True(t, ok, "kind %s must be registered", want)
		assert.Contains(t, ks, want, "Kinds() must list %s", want)
	}
}
```
Run: `go test -run TestBatchCKindsRegistered ./...`
Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 4: Run the linter on the new files.**
Run: `golangci-lint run --timeout=5m ./...`
Expected PASS: `0 issues`. If the OriginResolver assertion is flagged unused, ignore — adapters in M1/M3 consume it.

- [ ] **Step 5: Commit the verification test.**
```
git add plate_test.go
git commit -m "test: assert CEP, Phone, Plate kinds are registered"
```

- [ ] **Step 6: Confirm benchmarks run for all three types in one pass.**
Run: `go test -bench='CEP|Phone|Plate' -benchmem -run=^$ ./...`
Expected PASS: six benchmark result lines (Validate + Generate for CEP, Phone, Plate) each with ns/op and allocs/op.


---

## Milestone M2D — RG (Registro Geral, SP & RJ)

RG is the one document type whose validation is **UF-scoped**: it needs a
federative unit and returns an error. It implements the frozen `Document`
interface (where `Validate(value)` tries every *implemented* UF and `Format`
masks `##.###.###-#`) **and** the frozen `UFScoped` interface
(`ValidateUF(value, uf) (bool, error)` / `ImplementedUFs() []UF`). SP and RJ
ship here; any other UF returns `ErrUFNotImplemented`.

Algorithm (SP, from the gap analysis §3.8): input shape
`\d{2}.?\d{3}.?\d{3}-?[0-9xX]`; strip to the 8 base digits + 1 check char
(digit or `X`/`x`); compute mod-11 over the 8 base digits with positional
weights `2,3,4,5,6,7,8,9`; `dv = sum % 11`; the check char represents `10`
when it is `X`, and `11` when it is `0`; otherwise it is its own numeric
value. Valid iff the computed `dv` equals the check char's represented value.

RJ uses the **same** weighting scheme and check-char convention (the well-
documented RJ RG algorithm is identical to SP's mod-11/weights `2..9`); we
implement it as a distinct branch so future per-UF divergence is a localized
edit, not a signature change.

`Generate()` produces an SP-style RG: 8 random base digits, the computed
check char (`X` when `dv==10`, `0` when `dv==11`, else the digit), masked as
`##.###.###-#`. RG self-registers via `init()`.

---

### Task M2D-1: RG type skeleton + Kind + registration

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/rg.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/rg_test.go`

**Interfaces:**
- Consumes: `Kind` (`type Kind string`), `KindRG Kind = "rg"`, `Document` interface (`Kind() Kind`, `Validate(value string) bool`, `Generate() string`, `Format(value string) (string, error)`), `func Register(d Document)` — all from M0.
- Produces: `type RG struct{}`, `func NewRG() *RG`, `func (r *RG) Kind() Kind` (returns `KindRG`), `const RGBaseLength = 8`, `const RGTotalLength = 9`.

- [ ] **Step 1: Write a failing test for Kind() and constructor.** Create `rg_test.go`:
```go
package brdoc

import "testing"

func TestRG_Kind(t *testing.T) {
	r := NewRG()
	if got := r.Kind(); got != KindRG {
		t.Fatalf("Kind() = %q, want %q", got, KindRG)
	}
}

func TestRG_Registered(t *testing.T) {
	d, ok := Get(KindRG)
	if !ok {
		t.Fatal("RG not registered in registry")
	}
	if d.Kind() != KindRG {
		t.Fatalf("registered Kind() = %q, want %q", d.Kind(), KindRG)
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL.** Command: `go test -run "TestRG_Kind|TestRG_Registered" ./...`. Expected FAIL: `undefined: NewRG` / `undefined: RG` (compile error).

- [ ] **Step 3: Create the RG type, Kind, constructor, constants, and init registration.** Create `rg.go`:
```go
package brdoc

// RGBaseLength is the count of base (non-check) digits in an RG number.
const RGBaseLength = 8

// RGTotalLength is the count of significant characters in an RG number:
// 8 base digits plus 1 check character.
const RGTotalLength = 9

// rgWeights are the positional mod-11 weights applied to the 8 base digits
// of an SP/RJ RG, least-significant base digit first.
var rgWeights = [RGBaseLength]int{2, 3, 4, 5, 6, 7, 8, 9}

// RG is the Registro Geral (state identity card) document type. Only the SP
// and RJ algorithms are well-defined; other federative units return
// ErrUFNotImplemented.
type RG struct{}

// NewRG returns a stateless RG document.
func NewRG() *RG { return &RG{} }

// Kind reports the document kind (KindRG).
func (r *RG) Kind() Kind { return KindRG }

func init() { Register(&RG{}) }
```

- [ ] **Step 4: Run the test, expect PASS.** Command: `go test -run "TestRG_Kind|TestRG_Registered" ./...`. Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Commit.**
```
git add rg.go rg_test.go
git commit -m "feat: add RG type skeleton with Kind and registry registration"
```

---

### Task M2D-2: RG digit/check-char parsing helper

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg.go` (append helpers)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg_test.go` (append test)

**Interfaces:**
- Consumes: `func onlyDigits(s string) string` (M0 `helpers.go`).
- Produces (unexported, package-internal): `func (r *RG) parse(value string) (base [RGBaseLength]int, check int, ok bool)` — strips formatting, returns the 8 base digits, the check value (`10` for `X`/`x`, `11` for `0`, else the digit), and `ok=false` if the shape is wrong.

The check character is parsed *before* digit-only stripping so an `X`/`x`
check char survives. The base portion is taken from `onlyDigits` of the
value minus its final character.

- [ ] **Step 1: Write a failing test for parse().** Append to `rg_test.go`:
```go
func TestRG_parse(t *testing.T) {
	r := NewRG()
	tests := []struct {
		name      string
		in        string
		wantCheck int
		wantOK    bool
		wantBase0 int // first base digit (least-significant position, index 0)
	}{
		{"formatted with X check", "24.678.131-4", 4, true, 2},
		{"X check char upper", "11.111.111-X", 10, true, 1},
		{"x check char lower", "11.111.111-x", 10, true, 1},
		{"zero means eleven", "11.111.111-0", 11, true, 1},
		{"bare digits", "246781314", 4, true, 2},
		{"too short", "1234567", 0, false, 0},
		{"too long", "1234567890", 0, false, 0},
		{"non-digit base", "ab.cde.fgh-1", 0, false, 0},
		{"empty", "", 0, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, check, ok := r.parse(tt.in)
			if ok != tt.wantOK {
				t.Fatalf("parse(%q) ok = %v, want %v", tt.in, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if check != tt.wantCheck {
				t.Errorf("parse(%q) check = %d, want %d", tt.in, check, tt.wantCheck)
			}
			if base[0] != tt.wantBase0 {
				t.Errorf("parse(%q) base[0] = %d, want %d", tt.in, base[0], tt.wantBase0)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL.** Command: `go test -run TestRG_parse ./...`. Expected FAIL: `r.parse undefined (type *RG has no field or method parse)`.

- [ ] **Step 3: Implement parse().** Append to `rg.go`:
```go
// parse strips RG formatting and returns the 8 base digits, the represented
// check value, and ok=true when the input has exactly 8 base digits plus one
// valid check character. The check character is 'X'/'x' (=> 10), a digit, or
// '0' (=> 11). base is filled index 0..7 in input order.
func (r *RG) parse(value string) (base [RGBaseLength]int, check int, ok bool) {
	cleaned := r.clean(value)
	if len(cleaned) != RGTotalLength {
		return base, 0, false
	}
	last := cleaned[RGBaseLength] // the check character
	switch {
	case last == 'X' || last == 'x':
		check = 10
	case last == '0':
		check = 11
	case last >= '1' && last <= '9':
		check = int(last - '0')
	default:
		return base, 0, false
	}
	for i := 0; i < RGBaseLength; i++ {
		c := cleaned[i]
		if c < '0' || c > '9' {
			return base, 0, false
		}
		base[i] = int(c - '0')
	}
	return base, check, true
}

// clean strips dots and dashes (and any other non-alphanumeric punctuation)
// from an RG, preserving digits and a trailing X/x check character.
func (r *RG) clean(value string) string {
	out := make([]byte, 0, RGTotalLength)
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= '0' && c <= '9') || c == 'X' || c == 'x' {
			out = append(out, c)
		}
	}
	return string(out)
}
```

- [ ] **Step 4: Run the test, expect PASS.** Command: `go test -run TestRG_parse ./...`. Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Commit.**
```
git add rg.go rg_test.go
git commit -m "feat: add RG parse helper for base digits and check char"
```

---

### Task M2D-3: RG check-digit computation + ValidateUF (UFScoped)

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg.go` (append compute + ValidateUF + ImplementedUFs)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg_test.go` (append tests)

**Interfaces:**
- Consumes: `type UF string`, `func (u UF) Valid() bool` (M0 `uf.go`); constants `UFSP UF = "SP"`, `UFRJ UF = "RJ"` (M0 `uf.go`); `ErrUFNotImplemented = errors.New("brdoc: federative unit not implemented")`, `ErrInvalidFormat = errors.New("brdoc: invalid document format")` (M0 `errors.go`); `func errors.Is`.
- Produces: `func (r *RG) ValidateUF(value string, uf UF) (bool, error)`, `func (r *RG) ImplementedUFs() []UF` — together satisfying the frozen `UFScoped` interface. Also unexported `func (r *RG) checkDigit(base [RGBaseLength]int) int`.

`ValidateUF` returns `(false, ErrUFNotImplemented)` (wrapped with `%w`) for
any UF other than SP/RJ, including invalid UFs. For SP/RJ it returns
`(false, ErrInvalidFormat)` when the shape is wrong, and otherwise
`(computedDV == checkValue, nil)`.

- [ ] **Step 1: Write a failing test for ValidateUF, ImplementedUFs, and the unimplemented-UF error.** Append to `rg_test.go`:
```go
import "errors" // ensure errors is imported at top of rg_test.go

func TestRG_ValidateUF(t *testing.T) {
	r := NewRG()
	// 24.678.131-4 is a canonical valid SP RG sample (check digit 4).
	tests := []struct {
		name    string
		value   string
		uf      UF
		want    bool
		wantErr error
	}{
		{"valid SP formatted", "24.678.131-4", UFSP, true, nil},
		{"valid SP bare", "246781314", UFSP, true, nil},
		{"valid RJ (same algo)", "24.678.131-4", UFRJ, true, nil},
		{"wrong check digit SP", "24.678.131-5", UFSP, false, nil},
		{"wrong length SP", "1234567", UFSP, false, ErrInvalidFormat},
		{"unimplemented UF MG", "24.678.131-4", UFMG, false, ErrUFNotImplemented},
		{"unimplemented UF BA", "246781314", UFBA, false, ErrUFNotImplemented},
		{"invalid UF zz", "246781314", UF("ZZ"), false, ErrUFNotImplemented},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.ValidateUF(tt.value, tt.uf)
			if got != tt.want {
				t.Errorf("ValidateUF(%q,%q) = %v, want %v", tt.value, tt.uf, got, tt.want)
			}
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateUF(%q,%q) err = %v, want nil", tt.value, tt.uf, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateUF(%q,%q) err = %v, want errors.Is %v", tt.value, tt.uf, err, tt.wantErr)
			}
		})
	}
}

func TestRG_ImplementedUFs(t *testing.T) {
	r := NewRG()
	ufs := r.ImplementedUFs()
	want := map[UF]bool{UFSP: true, UFRJ: true}
	if len(ufs) != len(want) {
		t.Fatalf("ImplementedUFs() len = %d, want %d (%v)", len(ufs), len(want), ufs)
	}
	for _, u := range ufs {
		if !want[u] {
			t.Errorf("ImplementedUFs() returned unexpected UF %q", u)
		}
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL.** Command: `go test -run "TestRG_ValidateUF|TestRG_ImplementedUFs" ./...`. Expected FAIL: `r.ValidateUF undefined` / `r.ImplementedUFs undefined`.

- [ ] **Step 3: Implement checkDigit, ValidateUF, and ImplementedUFs.** Add the `errors` import to `rg.go` (change `package brdoc` block to `import "errors"`) and append:
```go
// checkDigit computes the mod-11 check value for the 8 base digits using the
// SP/RJ positional weights 2..9. The result is in 0..10 (10 is encoded as the
// check char 'X'; the special value 11 only arises from an input check char of
// '0' and is therefore never produced here).
func (r *RG) checkDigit(base [RGBaseLength]int) int {
	sum := 0
	for i := 0; i < RGBaseLength; i++ {
		sum += base[i] * rgWeights[i]
	}
	return sum % 11
}

// rgImplemented is the set of federative units whose RG algorithm is shipped.
var rgImplemented = map[UF]bool{UFSP: true, UFRJ: true}

// ImplementedUFs returns the federative units for which RG validation is
// supported (SP and RJ).
func (r *RG) ImplementedUFs() []UF { return []UF{UFSP, UFRJ} }

// ValidateUF reports whether value is a valid RG for the given federative
// unit. SP and RJ share the mod-11/weights-2..9 algorithm. Any other UF
// yields (false, ErrUFNotImplemented). A malformed value for a supported UF
// yields (false, ErrInvalidFormat).
func (r *RG) ValidateUF(value string, uf UF) (bool, error) {
	if !rgImplemented[uf] {
		return false, fmt.Errorf("%w: %s", ErrUFNotImplemented, uf)
	}
	base, check, ok := r.parse(value)
	if !ok {
		return false, ErrInvalidFormat
	}
	return r.checkDigit(base) == check, nil
}
```

- [ ] **Step 4: Add the fmt import.** The new `fmt.Errorf` requires `fmt`. Edit the import block of `rg.go` so it reads:
```go
import (
	"errors"
	"fmt"
)
```
(If `errors` ends up unused after later edits, keep it only while referenced; it is referenced indirectly via the sentinels in `errors.go`, so import only `fmt` here and drop `errors` — verify with `go vet`. To stay safe, the import block above includes both; remove `errors` if `go vet ./...` reports it unused.)

- [ ] **Step 5: Run the test, expect PASS.** Command: `go test -run "TestRG_ValidateUF|TestRG_ImplementedUFs" ./...`. Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 6: Vet for unused imports, then commit.**
```
go vet ./...
git add rg.go rg_test.go
git commit -m "feat: implement RG ValidateUF and ImplementedUFs (UFScoped)"
```

---

### Task M2D-4: RG Validate(value) over all implemented UFs (Document)

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg.go` (append Validate)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg_test.go` (append test)

**Interfaces:**
- Consumes: `func (r *RG) ValidateUF(value string, uf UF) (bool, error)`, `func (r *RG) ImplementedUFs() []UF` (M2D-3).
- Produces: `func (r *RG) Validate(value string) bool` — part of the frozen `Document` interface; returns true if value validates under *any* implemented UF.

Because SP and RJ share one algorithm, a value valid for SP is valid for RJ;
`Validate` returns true on the first implemented-UF match.

- [ ] **Step 1: Write a failing test for Validate.** Append to `rg_test.go`:
```go
func TestRG_Validate(t *testing.T) {
	r := NewRG()
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid SP formatted", "24.678.131-4", true},
		{"valid bare", "246781314", true},
		{"valid X check", "33.087.005-X", true}, // computed-X sample
		{"wrong check digit", "24.678.131-5", false},
		{"too short", "1234567", false},
		{"too long", "1234567890", false},
		{"garbage", "abcdefghi", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Validate(tt.value); got != tt.want {
				t.Errorf("Validate(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
```
> Note: the `33.087.005-X` row asserts the X-check path. If the chosen sample's computed DV is not 10, replace it in Step 3 with a value emitted by `Generate()` that ends in `X` (see M2D-6 round-trip test, which is the authoritative source of valid X samples). Use a generated value: run `go run ./cmd/brdoc rg -g -n 50` after M1 wiring, or in a scratch test print `NewRG().Generate()` until one ends in `-X`, and pin that exact string here.

- [ ] **Step 2: Run the test, expect FAIL.** Command: `go test -run TestRG_Validate ./...`. Expected FAIL: `r.Validate undefined (type *RG has no field or method Validate)`.

- [ ] **Step 3: Implement Validate.** Append to `rg.go`:
```go
// Validate reports whether value is a valid RG under any implemented
// federative unit (SP or RJ). It satisfies the Document interface.
func (r *RG) Validate(value string) bool {
	for _, uf := range r.ImplementedUFs() {
		ok, err := r.ValidateUF(value, uf)
		if err == nil && ok {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run the test, expect PASS.** Command: `go test -run TestRG_Validate ./...`. Expected PASS: `ok  github.com/inovacc/brdoc`.
> If the `33.087.005-X` row fails, replace it per the Step-1 note with a generated X-ending RG and re-run.

- [ ] **Step 5: Commit.**
```
git add rg.go rg_test.go
git commit -m "feat: implement RG Validate across implemented UFs"
```

---

### Task M2D-5: RG Format (##.###.###-#)

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg.go` (append Format)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg_test.go` (append test)

**Interfaces:**
- Consumes: `func (r *RG) parse(...)` (M2D-2); `ErrInvalidFormat` (M0 `errors.go`); `func errors.Is`.
- Produces: `func (r *RG) Format(value string) (string, error)` — frozen `Document` interface; returns `XX.XXX.XXX-C` where `C` is the original check char (`X`/`x` preserved as uppercase `X`, `0` preserved as `0`).

- [ ] **Step 1: Write a failing test for Format.** Append to `rg_test.go`:
```go
func TestRG_Format(t *testing.T) {
	r := NewRG()
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{"bare digit check", "246781314", "24.678.131-4", nil},
		{"already formatted", "24.678.131-4", "24.678.131-4", nil},
		{"X check normalized", "11111111x", "11.111.111-X", nil},
		{"zero check", "111111110", "11.111.111-0", nil},
		{"too short", "1234567", "", ErrInvalidFormat},
		{"too long", "1234567890", "", ErrInvalidFormat},
		{"non-digit base", "ab.cde.fgh-1", "", ErrInvalidFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Format(tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Format(%q) err = %v, want errors.Is %v", tt.in, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Format(%q) unexpected err: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("Format(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test, expect FAIL.** Command: `go test -run TestRG_Format ./...`. Expected FAIL: `r.Format undefined (type *RG has no field or method Format)`.

- [ ] **Step 3: Implement Format.** Append to `rg.go`:
```go
// Format renders an RG as XX.XXX.XXX-C. The check character is normalized:
// 'x' becomes 'X', and '0' is preserved. It returns ErrInvalidFormat when the
// value does not have 8 base digits plus a valid check character.
func (r *RG) Format(value string) (string, error) {
	base, check, ok := r.parse(value)
	if !ok {
		return "", ErrInvalidFormat
	}
	var checkChar byte
	switch check {
	case 10:
		checkChar = 'X'
	case 11:
		checkChar = '0'
	default:
		checkChar = byte('0' + check)
	}
	buf := make([]byte, 0, 12)
	for i := 0; i < RGBaseLength; i++ {
		buf = append(buf, byte('0'+base[i]))
		if i == 1 || i == 4 {
			buf = append(buf, '.')
		}
	}
	buf = append(buf, '-', checkChar)
	return string(buf), nil
}
```

- [ ] **Step 4: Run the test, expect PASS.** Command: `go test -run TestRG_Format ./...`. Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Commit.**
```
git add rg.go rg_test.go
git commit -m "feat: implement RG Format mask ##.###.###-#"
```

---

### Task M2D-6: RG Generate (SP-style) + round-trip + fuzz

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg.go` (append Generate)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg_test.go` (append round-trip, benchmark, fuzz)

**Interfaces:**
- Consumes: `math/rand/v2` (`rand.IntN`); `func (r *RG) checkDigit(...)` (M2D-3); `func (r *RG) Validate(value string) bool` (M2D-4).
- Produces: `func (r *RG) Generate() string` — frozen `Document` interface; emits a masked, SP-style valid RG (`##.###.###-#`). When the computed DV is 10 the check char is `X`; the DV value 11 is never produced (mod-11 of generated digits is in 0..10), so generated RGs use check chars `0..9` or `X`.

- [ ] **Step 1: Write a failing round-trip test, benchmark, and fuzz test.** Append to `rg_test.go`:
```go
func TestRG_GenerateRoundTrip(t *testing.T) {
	r := NewRG()
	for i := 0; i < 1000; i++ {
		v := r.Generate()
		if !r.Validate(v) {
			t.Fatalf("Generate() produced invalid RG: %q", v)
		}
		// Generated value must be in masked form XX.XXX.XXX-C.
		if len(v) != 12 || v[2] != '.' || v[6] != '.' || v[10] != '-' {
			t.Fatalf("Generate() not masked correctly: %q", v)
		}
	}
}

func BenchmarkRGValidate(b *testing.B) {
	r := NewRG()
	const sample = "24.678.131-4"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Validate(sample)
	}
}

func BenchmarkRGGenerate(b *testing.B) {
	r := NewRG()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Generate()
	}
}

func FuzzRGValidate(f *testing.F) {
	r := NewRG()
	f.Add("24.678.131-4")
	f.Add("246781314")
	f.Add("11.111.111-X")
	f.Add("")
	f.Add("garbage-input")
	f.Fuzz(func(t *testing.T, s string) {
		// Must never panic; result is ignored.
		_ = r.Validate(s)
		_, _ = r.Format(s)
		_, _ = r.ValidateUF(s, UFSP)
	})
}
```

- [ ] **Step 2: Run the round-trip test, expect FAIL.** Command: `go test -run TestRG_GenerateRoundTrip ./...`. Expected FAIL: `r.Generate undefined (type *RG has no field or method Generate)`.

- [ ] **Step 3: Implement Generate.** Add `math/rand/v2` to the imports of `rg.go` and append:
```go
// Generate returns a syntactically valid, SP-style RG in masked form
// XX.XXX.XXX-C. The 8 base digits are random; the check character is computed
// via the SP/RJ mod-11 algorithm ('X' when the DV is 10). It satisfies the
// Document interface.
func (r *RG) Generate() string {
	var base [RGBaseLength]int
	for i := 0; i < RGBaseLength; i++ {
		base[i] = rand.IntN(10)
	}
	dv := r.checkDigit(base)
	var checkChar byte
	if dv == 10 {
		checkChar = 'X'
	} else {
		checkChar = byte('0' + dv)
	}
	buf := make([]byte, 0, 12)
	for i := 0; i < RGBaseLength; i++ {
		buf = append(buf, byte('0'+base[i]))
		if i == 1 || i == 4 {
			buf = append(buf, '.')
		}
	}
	buf = append(buf, '-', checkChar)
	return string(buf)
}
```
> Import block of `rg.go` after this step:
```go
import (
	"fmt"
	"math/rand/v2"
)
```
(`errors` is not imported directly — the sentinels are referenced by value from `errors.go`. Run `go vet ./...`; if it flags an unused import remove it.)

- [ ] **Step 4: Run the round-trip test, expect PASS.** Command: `go test -run TestRG_GenerateRoundTrip ./...`. Expected PASS: `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Run the fuzz test briefly, expect PASS (no panic).** Command: `go test -run FuzzRGValidate -fuzz FuzzRGValidate -fuzztime 10s ./...`. Expected: `PASS` with no `--- FAIL` / panic. (CI uses `-run FuzzRGValidate` seed-corpus only.)

- [ ] **Step 6: Resolve the M2D-4 X-sample placeholder.** Run `go test -run TestRG_GenerateRoundTrip ./...` with a temporary `t.Logf("%s", v)` (or `go run ./cmd/brdoc rg -g -n 50` once M1 is wired) to capture a real `-X`-ending RG; paste that exact string into the `{"valid X check", ...}` row of `TestRG_Validate` (M2D-4) and the `{"X check normalized", ...}` expectation if needed. Re-run `go test -run TestRG_Validate ./...` to confirm PASS, then remove any temporary logging.

- [ ] **Step 7: Run the full RG suite, expect PASS.** Command: `go test -run TestRG ./... && go test -bench=RG -benchmem -run=^$ ./...`. Expected: `ok  github.com/inovacc/brdoc` and benchmark lines for `BenchmarkRGValidate` / `BenchmarkRGGenerate`.

- [ ] **Step 8: Commit.**
```
git add rg.go rg_test.go
git commit -m "feat: implement RG Generate with round-trip, benchmark, and fuzz tests"
```

---

### Task M2D-7: RG interface conformance + godoc example

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/rg_test.go` (append conformance asserts + Example)

**Interfaces:**
- Consumes: `Document` interface, `UFScoped` interface (M0 `document.go`); `func Get(kind Kind) (Document, bool)` (M0 `registry.go`).
- Produces: compile-time interface assertions and a runnable `ExampleRG_Validate` for pkg.go.dev.

- [ ] **Step 1: Write the failing conformance + example test.** Append to `rg_test.go`:
```go
import "fmt" // ensure fmt imported in rg_test.go

// Compile-time interface conformance.
var (
	_ Document = (*RG)(nil)
	_ UFScoped = (*RG)(nil)
)

func TestRG_RegistryDispatch(t *testing.T) {
	// Registry-level Validate must route to RG via Kind.
	ok, err := Validate(KindRG, "24.678.131-4")
	if err != nil {
		t.Fatalf("Validate(KindRG, ...) err = %v", err)
	}
	if !ok {
		t.Fatal("Validate(KindRG, valid sample) = false, want true")
	}
}

func ExampleRG_Validate() {
	r := NewRG()
	fmt.Println(r.Validate("24.678.131-4"))
	// Output: true
}
```

- [ ] **Step 2: Run, expect FAIL only if a contract slipped; otherwise PASS.** Command: `go test -run "TestRG_RegistryDispatch|ExampleRG_Validate" ./...`. Expected: compiles and PASS (the `var _ Document = (*RG)(nil)` assertions fail to compile only if the interface is not fully implemented — they are the safety net for M2D-1..6). If `ExampleRG_Validate` prints anything other than `true`, fix the sample to a known-valid generated RG.

- [ ] **Step 3: Run the entire package test suite, expect PASS.** Command: `go test ./...`. Expected: `ok  github.com/inovacc/brdoc` (all existing CPF/CNPJ/M0/M1 tests plus RG remain green).

- [ ] **Step 4: Lint, expect clean.** Command: `golangci-lint run --fix ./... --timeout=5m`. Expected: no findings for `rg.go`.

- [ ] **Step 5: Commit.**
```
git add rg_test.go
git commit -m "test: add RG interface conformance, registry dispatch, and godoc example"
```


---

## Milestone COMPAT — paemuri drop-in subpackage (`compat/`)

> Goal: ship a `compat/` subpackage that exposes the EXACT signatures of `github.com/paemuri/brdoc/v3` as thin wrappers over the root `brdoc` package, so a paemuri user migrates with a one-line import swap. The compat package adds NO validation logic of its own — every wrapper delegates to a root concrete type or registry function. Parity is proven by `compat_test.go`.
>
> **Dependency note (read before executing):** This milestone CONSUMES concrete types produced by the type-breadth milestone(s): `brdoc.NewCNH`, `brdoc.NewPIS`, `brdoc.NewRenavam`, `brdoc.NewVoterID`, `brdoc.NewCNS`, `brdoc.NewPlate`, `brdoc.NewCEP`, `brdoc.NewPhone`, `brdoc.NewRG`, plus CPF/CNPJ from M0, the `brdoc.UF` type and 27 UF constants from M0-3, the `brdoc.OriginResolver` / `brdoc.UFScoped` interfaces from M0-1, and `brdoc.ErrUFNotImplemented` from M0-2. Those tasks must be complete (their constructors registered and exported) before this milestone runs. Each task below states its exact Consumes signatures.

### Task MC-1: compat package skeleton + UF alias + the digit-only `Is*` wrappers

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go`

**Interfaces:**
- Consumes (from root `brdoc`, frozen contract / type-breadth milestone):
  - `func NewCPF() *brdoc.CPF` ; `func (c *brdoc.CPF) Validate(value string) bool`
  - `func NewCNPJ() *brdoc.CNPJ` ; `func (c *brdoc.CNPJ) Validate(value string) bool`
  - `func NewCNH() *brdoc.CNH` ; `func (c *brdoc.CNH) Validate(value string) bool`
  - `func NewPIS() *brdoc.PIS` ; `func (p *brdoc.PIS) Validate(value string) bool`
  - `func NewRenavam() *brdoc.Renavam` ; `func (r *brdoc.Renavam) Validate(value string) bool`
  - `func NewVoterID() *brdoc.VoterID` ; `func (v *brdoc.VoterID) Validate(value string) bool`
  - `func NewCNS() *brdoc.CNS` ; `func (c *brdoc.CNS) Validate(value string) bool`
  - `type brdoc.UF string`
- Produces:
  - `type UF = brdoc.UF` (type alias, identical to root so signatures match paemuri verbatim)
  - `func IsCPF(s string) bool`
  - `func IsCNPJ(s string) bool`
  - `func IsCNH(s string) bool`
  - `func IsPIS(s string) bool`
  - `func IsRENAVAM(s string) bool`
  - `func IsVoterID(s string) bool`
  - `func IsCNS(s string) bool`

Steps:

- [ ] **Step 1: Create the package file with the doc comment, import, and the UF type alias only.** Write `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go`:
  ```go
  // Package compat provides drop-in replacements for the public API of
  // github.com/paemuri/brdoc/v3. Every function is a thin wrapper over the
  // root github.com/inovacc/brdoc package, so a paemuri user can migrate by
  // changing a single import path. No validation logic lives here.
  package compat

  import "github.com/inovacc/brdoc"

  // UF aliases the root brdoc.UF type so that the wrapper signatures below
  // match paemuri/brdoc v3 exactly (e.g. func IsCEP(s string) (bool, UF)).
  type UF = brdoc.UF
  ```

- [ ] **Step 2: Write a FAILING test for the digit-only Is* wrappers.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go`:
  ```go
  package compat

  import (
  	"testing"

  	"github.com/inovacc/brdoc"
  	"github.com/stretchr/testify/assert"
  )

  func TestIsDigitDocs_ParityWithRoot(t *testing.T) {
  	t.Parallel()
  	tests := []struct {
  		name    string
  		compat  func(string) bool
  		root    func(string) bool
  		valid   string // a generated-valid sample produced below
  		invalid string
  	}{
  		{"cpf", IsCPF, func(s string) bool { return brdoc.NewCPF().Validate(s) }, brdoc.NewCPF().Generate(), "00000000000"},
  		{"cnpj", IsCNPJ, func(s string) bool { return brdoc.NewCNPJ().Validate(s) }, brdoc.NewCNPJ().Generate(), "00000000000000"},
  		{"cnh", IsCNH, func(s string) bool { return brdoc.NewCNH().Validate(s) }, brdoc.NewCNH().Generate(), "11111111111"},
  		{"pis", IsPIS, func(s string) bool { return brdoc.NewPIS().Validate(s) }, brdoc.NewPIS().Generate(), "00000000001"},
  		{"renavam", IsRENAVAM, func(s string) bool { return brdoc.NewRenavam().Validate(s) }, brdoc.NewRenavam().Generate(), "00000000001"},
  		{"voterid", IsVoterID, func(s string) bool { return brdoc.NewVoterID().Validate(s) }, brdoc.NewVoterID().Generate(), "000000000000"},
  		{"cns", IsCNS, func(s string) bool { return brdoc.NewCNS().Validate(s) }, brdoc.NewCNS().Generate(), "000000000000000"},
  	}
  	for _, tt := range tests {
  		t.Run(tt.name, func(t *testing.T) {
  			t.Parallel()
  			assert.True(t, tt.compat(tt.valid), "compat must accept a valid %s", tt.name)
  			assert.Equal(t, tt.root(tt.valid), tt.compat(tt.valid), "compat must mirror root on valid %s", tt.name)
  			assert.False(t, tt.compat(tt.invalid), "compat must reject invalid %s", tt.name)
  			assert.Equal(t, tt.root(tt.invalid), tt.compat(tt.invalid), "compat must mirror root on invalid %s", tt.name)
  		})
  	}
  }
  ```

- [ ] **Step 3: Run the test and watch it FAIL to compile.** Run:
  ```
  go test ./compat/ -run TestIsDigitDocs_ParityWithRoot
  ```
  Expected FAIL: `./compat_test.go: undefined: IsCPF` (and the other `Is*` symbols), because the wrappers are not defined yet.

- [ ] **Step 4: Implement the seven digit-only wrappers.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go`:
  ```go
  // IsCPF reports whether s is a valid CPF. Mirrors paemuri/brdoc.IsCPF.
  func IsCPF(s string) bool { return brdoc.NewCPF().Validate(s) }

  // IsCNPJ reports whether s is a valid CNPJ. Mirrors paemuri/brdoc.IsCNPJ.
  func IsCNPJ(s string) bool { return brdoc.NewCNPJ().Validate(s) }

  // IsCNH reports whether s is a valid CNH. Mirrors paemuri/brdoc.IsCNH.
  func IsCNH(s string) bool { return brdoc.NewCNH().Validate(s) }

  // IsPIS reports whether s is a valid PIS/PASEP/NIS/NIT. Mirrors paemuri/brdoc.IsPIS.
  func IsPIS(s string) bool { return brdoc.NewPIS().Validate(s) }

  // IsRENAVAM reports whether s is a valid RENAVAM. Mirrors paemuri/brdoc.IsRENAVAM.
  func IsRENAVAM(s string) bool { return brdoc.NewRenavam().Validate(s) }

  // IsVoterID reports whether s is a valid Título Eleitoral. Mirrors paemuri/brdoc.IsVoterID.
  func IsVoterID(s string) bool { return brdoc.NewVoterID().Validate(s) }

  // IsCNS reports whether s is a valid CNS (health card). Mirrors paemuri/brdoc.IsCNS.
  func IsCNS(s string) bool { return brdoc.NewCNS().Validate(s) }
  ```

- [ ] **Step 5: Run the test and watch it PASS.** Run:
  ```
  go test ./compat/ -run TestIsDigitDocs_ParityWithRoot -v
  ```
  Expected PASS: `--- PASS: TestIsDigitDocs_ParityWithRoot` with all 7 subtests (`cpf`, `cnpj`, `cnh`, `pis`, `renavam`, `voterid`, `cns`) PASS.

- [ ] **Step 6: Commit.** Run:
  ```
  git add compat/compat.go compat/compat_test.go
  git commit -m "feat(compat): add paemuri-compatible digit-only Is* wrappers + UF alias"
  ```

### Task MC-2: plate wrappers — `IsPlate` / `IsNationalPlate` / `IsMercosulPlate`

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go` (append functions)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go` (append test)

**Interfaces:**
- Consumes (from the plate type milestone):
  - `func NewPlate() *brdoc.Plate`
  - `func (p *brdoc.Plate) Validate(value string) bool` — true for either national or Mercosul
  - `func (p *brdoc.Plate) ValidateNational(value string) bool` — `^[A-Z]{3}-?\d{4}$`
  - `func (p *brdoc.Plate) ValidateMercosul(value string) bool` — `^[A-Z]{3}\d[A-Z]\d{2}$`
- Produces:
  - `func IsPlate(s string) bool`
  - `func IsNationalPlate(s string) bool`
  - `func IsMercosulPlate(s string) bool`

Steps:

- [ ] **Step 1: Write a FAILING test for the plate wrappers.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go`:
  ```go
  func TestPlateWrappers(t *testing.T) {
  	t.Parallel()
  	tests := []struct {
  		value        string
  		wantPlate    bool
  		wantNational bool
  		wantMercosul bool
  	}{
  		{"ABC1234", true, true, false},   // legacy national, no dash
  		{"ABC-1234", true, true, false},  // legacy national, dashed
  		{"ABC1D23", true, false, true},   // Mercosul
  		{"AB1234", false, false, false},  // too short
  		{"1234ABC", false, false, false}, // wrong order
  		{"ABCD123", false, false, false}, // 4 letters
  	}
  	for _, tt := range tests {
  		t.Run(tt.value, func(t *testing.T) {
  			t.Parallel()
  			assert.Equal(t, tt.wantPlate, IsPlate(tt.value), "IsPlate(%q)", tt.value)
  			assert.Equal(t, tt.wantNational, IsNationalPlate(tt.value), "IsNationalPlate(%q)", tt.value)
  			assert.Equal(t, tt.wantMercosul, IsMercosulPlate(tt.value), "IsMercosulPlate(%q)", tt.value)
  		})
  	}
  }

  func TestPlateWrappers_ParityWithRoot(t *testing.T) {
  	t.Parallel()
  	p := brdoc.NewPlate()
  	for _, v := range []string{"ABC1234", "ABC-1234", "ABC1D23", "AB1234"} {
  		assert.Equal(t, p.Validate(v), IsPlate(v), "IsPlate parity %q", v)
  		assert.Equal(t, p.ValidateNational(v), IsNationalPlate(v), "IsNationalPlate parity %q", v)
  		assert.Equal(t, p.ValidateMercosul(v), IsMercosulPlate(v), "IsMercosulPlate parity %q", v)
  	}
  }
  ```

- [ ] **Step 2: Run the test and watch it FAIL to compile.** Run:
  ```
  go test ./compat/ -run TestPlate
  ```
  Expected FAIL: `undefined: IsPlate`, `undefined: IsNationalPlate`, `undefined: IsMercosulPlate`.

- [ ] **Step 3: Implement the three plate wrappers.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go`:
  ```go
  // IsPlate reports whether s is a valid vehicle plate (national OR Mercosul).
  // Mirrors paemuri/brdoc.IsPlate.
  func IsPlate(s string) bool { return brdoc.NewPlate().Validate(s) }

  // IsNationalPlate reports whether s is a valid legacy national plate (ABC-1234).
  // Mirrors paemuri/brdoc.IsNationalPlate.
  func IsNationalPlate(s string) bool { return brdoc.NewPlate().ValidateNational(s) }

  // IsMercosulPlate reports whether s is a valid Mercosul plate (ABC1D23).
  // Mirrors paemuri/brdoc.IsMercosulPlate.
  func IsMercosulPlate(s string) bool { return brdoc.NewPlate().ValidateMercosul(s) }
  ```

- [ ] **Step 4: Run the test and watch it PASS.** Run:
  ```
  go test ./compat/ -run TestPlate -v
  ```
  Expected PASS: `--- PASS: TestPlateWrappers` and `--- PASS: TestPlateWrappers_ParityWithRoot`.

- [ ] **Step 5: Commit.** Run:
  ```
  git add compat/compat.go compat/compat_test.go
  git commit -m "feat(compat): add plate wrappers (IsPlate/IsNationalPlate/IsMercosulPlate)"
  ```

### Task MC-3: CEP wrappers — `IsCEP(s) (bool, UF)` and `IsCEPFrom(s, ufs...) bool`

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go` (append functions)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go` (append test)

**Interfaces:**
- Consumes (from the CEP type milestone + M0):
  - `func NewCEP() *brdoc.CEP`
  - `func (c *brdoc.CEP) Validate(value string) bool`
  - `func (c *brdoc.CEP) Origin(value string) (string, error)` — returns the 2-letter UF string for a valid CEP; non-nil error for invalid input (satisfies `brdoc.OriginResolver`)
  - `type brdoc.UF string` ; UF constants `brdoc.UFSP`, `brdoc.UFRJ`, ... (27 from M0-3)
  - `type UF = brdoc.UF` (from Task MC-1)
- Produces:
  - `func IsCEP(s string) (bool, UF)` — paemuri returns `("",UF)` zero-UF when invalid
  - `func IsCEPFrom(s string, ufs ...UF) bool`

Steps:

- [ ] **Step 1: Write a FAILING test for the CEP wrappers.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go`:
  ```go
  func TestIsCEP(t *testing.T) {
  	t.Parallel()
  	// Generate a valid CEP and resolve its expected UF from the root type so
  	// the test stays correct regardless of which range the generator picked.
  	cep := brdoc.NewCEP()
  	valid := cep.Generate()
  	wantUF, err := cep.Origin(valid)
  	assert.NoError(t, err, "root must resolve origin for its own generated CEP")

  	ok, gotUF := IsCEP(valid)
  	assert.True(t, ok, "IsCEP must accept a valid CEP %q", valid)
  	assert.Equal(t, brdoc.UF(wantUF), gotUF, "IsCEP must return the same UF as root.Origin")

  	// Invalid CEP -> (false, zero UF).
  	badOK, badUF := IsCEP("000")
  	assert.False(t, badOK, "IsCEP must reject a too-short value")
  	assert.Equal(t, UF(""), badUF, "invalid CEP must return the zero UF")
  }

  func TestIsCEPFrom(t *testing.T) {
  	t.Parallel()
  	cep := brdoc.NewCEP()
  	valid := cep.Generate()
  	originStr, err := cep.Origin(valid)
  	assert.NoError(t, err)
  	uf := brdoc.UF(originStr)

  	assert.True(t, IsCEPFrom(valid, uf), "IsCEPFrom must accept when uf matches")
  	assert.True(t, IsCEPFrom(valid), "IsCEPFrom with no UFs must behave like IsCEP (valid -> true)")
  	// Pick a UF guaranteed different from the resolved one.
  	other := brdoc.UFSP
  	if uf == brdoc.UFSP {
  		other = brdoc.UFRJ
  	}
  	assert.False(t, IsCEPFrom(valid, other), "IsCEPFrom must reject when uf does not match")
  	assert.False(t, IsCEPFrom("000", uf), "IsCEPFrom must reject an invalid CEP")
  }
  ```

- [ ] **Step 2: Run the test and watch it FAIL to compile.** Run:
  ```
  go test ./compat/ -run "TestIsCEP|TestIsCEPFrom"
  ```
  Expected FAIL: `undefined: IsCEP`, `undefined: IsCEPFrom`.

- [ ] **Step 3: Implement `IsCEP` and `IsCEPFrom`.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go`:
  ```go
  // IsCEP reports whether s is a valid CEP and, if so, the UF it maps to.
  // On invalid input it returns (false, ""). Mirrors paemuri/brdoc.IsCEP.
  func IsCEP(s string) (bool, UF) {
  	c := brdoc.NewCEP()
  	if !c.Validate(s) {
  		return false, UF("")
  	}
  	origin, err := c.Origin(s)
  	if err != nil {
  		return false, UF("")
  	}
  	return true, UF(origin)
  }

  // IsCEPFrom reports whether s is a valid CEP whose UF is one of ufs.
  // With no ufs it behaves like the bool part of IsCEP. Mirrors paemuri/brdoc.IsCEPFrom.
  func IsCEPFrom(s string, ufs ...UF) bool {
  	ok, uf := IsCEP(s)
  	if !ok {
  		return false
  	}
  	if len(ufs) == 0 {
  		return true
  	}
  	for _, want := range ufs {
  		if uf == want {
  			return true
  		}
  	}
  	return false
  }
  ```

- [ ] **Step 4: Run the test and watch it PASS.** Run:
  ```
  go test ./compat/ -run "TestIsCEP|TestIsCEPFrom" -v
  ```
  Expected PASS: `--- PASS: TestIsCEP` and `--- PASS: TestIsCEPFrom`.

- [ ] **Step 5: Commit.** Run:
  ```
  git add compat/compat.go compat/compat_test.go
  git commit -m "feat(compat): add CEP wrappers (IsCEP/IsCEPFrom) with UF resolution"
  ```

### Task MC-4: phone wrappers — `IsPhone(s) (bool, UF)` and `IsPhoneFrom(s, ufs...) bool`

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go` (append functions)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go` (append test)

**Interfaces:**
- Consumes (from the phone type milestone + M0):
  - `func NewPhone() *brdoc.Phone`
  - `func (p *brdoc.Phone) Validate(value string) bool`
  - `func (p *brdoc.Phone) Origin(value string) (string, error)` — returns the UF string for the DDD; non-nil error for invalid input (satisfies `brdoc.OriginResolver`)
  - `type brdoc.UF string` ; UF constants `brdoc.UFSP`, `brdoc.UFRJ`
  - `type UF = brdoc.UF` (from Task MC-1)
- Produces:
  - `func IsPhone(s string) (bool, UF)`
  - `func IsPhoneFrom(s string, ufs ...UF) bool`

Steps:

- [ ] **Step 1: Write a FAILING test for the phone wrappers.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go`:
  ```go
  func TestIsPhone(t *testing.T) {
  	t.Parallel()
  	phone := brdoc.NewPhone()
  	valid := phone.Generate()
  	wantUF, err := phone.Origin(valid)
  	assert.NoError(t, err, "root must resolve origin for its own generated phone")

  	ok, gotUF := IsPhone(valid)
  	assert.True(t, ok, "IsPhone must accept a valid phone %q", valid)
  	assert.Equal(t, brdoc.UF(wantUF), gotUF, "IsPhone must return the same UF as root.Origin")

  	badOK, badUF := IsPhone("123")
  	assert.False(t, badOK, "IsPhone must reject a too-short value")
  	assert.Equal(t, UF(""), badUF, "invalid phone must return the zero UF")
  }

  func TestIsPhoneFrom(t *testing.T) {
  	t.Parallel()
  	phone := brdoc.NewPhone()
  	valid := phone.Generate()
  	originStr, err := phone.Origin(valid)
  	assert.NoError(t, err)
  	uf := brdoc.UF(originStr)

  	assert.True(t, IsPhoneFrom(valid, uf), "IsPhoneFrom must accept when uf matches")
  	assert.True(t, IsPhoneFrom(valid), "IsPhoneFrom with no UFs must behave like IsPhone (valid -> true)")
  	other := brdoc.UFSP
  	if uf == brdoc.UFSP {
  		other = brdoc.UFRJ
  	}
  	assert.False(t, IsPhoneFrom(valid, other), "IsPhoneFrom must reject when uf does not match")
  	assert.False(t, IsPhoneFrom("123", uf), "IsPhoneFrom must reject an invalid phone")
  }
  ```

- [ ] **Step 2: Run the test and watch it FAIL to compile.** Run:
  ```
  go test ./compat/ -run "TestIsPhone|TestIsPhoneFrom"
  ```
  Expected FAIL: `undefined: IsPhone`, `undefined: IsPhoneFrom`.

- [ ] **Step 3: Implement `IsPhone` and `IsPhoneFrom`.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go`:
  ```go
  // IsPhone reports whether s is a valid Brazilian phone number and, if so, the
  // UF its DDD maps to. On invalid input it returns (false, ""). Mirrors
  // paemuri/brdoc.IsPhone.
  func IsPhone(s string) (bool, UF) {
  	p := brdoc.NewPhone()
  	if !p.Validate(s) {
  		return false, UF("")
  	}
  	origin, err := p.Origin(s)
  	if err != nil {
  		return false, UF("")
  	}
  	return true, UF(origin)
  }

  // IsPhoneFrom reports whether s is a valid phone whose UF is one of ufs.
  // With no ufs it behaves like the bool part of IsPhone. Mirrors
  // paemuri/brdoc.IsPhoneFrom.
  func IsPhoneFrom(s string, ufs ...UF) bool {
  	ok, uf := IsPhone(s)
  	if !ok {
  		return false
  	}
  	if len(ufs) == 0 {
  		return true
  	}
  	for _, want := range ufs {
  		if uf == want {
  			return true
  		}
  	}
  	return false
  }
  ```

- [ ] **Step 4: Run the test and watch it PASS.** Run:
  ```
  go test ./compat/ -run "TestIsPhone|TestIsPhoneFrom" -v
  ```
  Expected PASS: `--- PASS: TestIsPhone` and `--- PASS: TestIsPhoneFrom`.

- [ ] **Step 5: Commit.** Run:
  ```
  git add compat/compat.go compat/compat_test.go
  git commit -m "feat(compat): add phone wrappers (IsPhone/IsPhoneFrom) with UF resolution"
  ```

### Task MC-5: RG wrapper — `IsRG(s, uf) (bool, error)`

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go` (append function)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go` (append test)

**Interfaces:**
- Consumes (from the RG type milestone + M0):
  - `func NewRG() *brdoc.RG`
  - `func (r *brdoc.RG) ValidateUF(value string, uf brdoc.UF) (bool, error)` — from the `brdoc.UFScoped` interface; returns `(true,nil)` for a valid SP/RJ RG, `(false, %w brdoc.ErrUFNotImplemented)` for UFs without an algorithm
  - `var brdoc.ErrUFNotImplemented = errors.New("brdoc: federative unit not implemented")` (M0-2)
  - UF constants `brdoc.UFSP`, `brdoc.UFRJ`, `brdoc.UFAC` (M0-3)
  - `type UF = brdoc.UF` (from Task MC-1)
- Produces:
  - `func IsRG(s string, uf UF) (bool, error)`

Steps:

- [ ] **Step 1: Write a FAILING test for the RG wrapper.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go`:
  ```go
  func TestIsRG(t *testing.T) {
  	t.Parallel()
  	rg := brdoc.NewRG()

  	// Valid SP RG sample: 33.962.657-1 (mod-11 with weights 2..9, check 1).
  	const validSP = "33.962.657-1"
  	wantOK, wantErr := rg.ValidateUF(validSP, brdoc.UFSP)
  	gotOK, gotErr := IsRG(validSP, brdoc.UFSP)
  	assert.Equal(t, wantOK, gotOK, "IsRG must mirror root validity for a valid SP RG")
  	assert.Equal(t, wantErr, gotErr, "IsRG must mirror root error for a valid SP RG")
  	assert.True(t, gotOK, "valid SP RG must pass")
  	assert.NoError(t, gotErr, "valid SP RG must not error")

  	// Wrong check digit for SP -> (false, nil) (well-formed but invalid).
  	badOK, badErr := IsRG("33.962.657-0", brdoc.UFSP)
  	assert.False(t, badOK, "off-by-one check digit must fail")
  	assert.NoError(t, badErr, "an invalid-but-wellformed RG is (false, nil), not an error")

  	// Unimplemented UF -> error wrapping ErrUFNotImplemented.
  	ufOK, ufErr := IsRG("12345678", brdoc.UFAC)
  	assert.False(t, ufOK, "unimplemented UF must not validate")
  	assert.ErrorIs(t, ufErr, brdoc.ErrUFNotImplemented, "unimplemented UF must wrap ErrUFNotImplemented")
  }
  ```

- [ ] **Step 2: Run the test and watch it FAIL to compile.** Run:
  ```
  go test ./compat/ -run TestIsRG
  ```
  Expected FAIL: `undefined: IsRG`.

- [ ] **Step 3: Implement `IsRG`.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat.go`:
  ```go
  // IsRG reports whether s is a valid RG for the given UF. The returned error is
  // non-nil (wrapping brdoc.ErrUFNotImplemented) when uf has no implemented
  // algorithm; a well-formed but invalid RG returns (false, nil).
  // Mirrors paemuri/brdoc.IsRG.
  func IsRG(s string, uf UF) (bool, error) {
  	return brdoc.NewRG().ValidateUF(s, uf)
  }
  ```

- [ ] **Step 4: Run the test and watch it PASS.** Run:
  ```
  go test ./compat/ -run TestIsRG -v
  ```
  Expected PASS: `--- PASS: TestIsRG`.

- [ ] **Step 5: Commit.** Run:
  ```
  git add compat/compat.go compat/compat_test.go
  git commit -m "feat(compat): add IsRG(s, uf) wrapper over UFScoped RG validation"
  ```

### Task MC-6: full-package compile + vet + coverage gate and signature-parity guard

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go` (append a compile-time signature-parity guard test)

**Interfaces:**
- Consumes: all `compat.Is*` functions defined in Tasks MC-1..MC-5; `type UF = brdoc.UF` (MC-1).
- Produces: no new exported symbols — adds a guard test ensuring the exact paemuri signatures stay frozen.

Steps:

- [ ] **Step 1: Write a compile-time signature-parity guard test.** Append to `D:/weaver-sync/development/personal/projects/brdoc/compat/compat_test.go`. Assigning each wrapper to a variable of the EXACT paemuri function type makes any future signature drift a compile error:
  ```go
  // TestSignatureParity is a compile-time guard: each assignment fails to build
  // if a wrapper's signature drifts from paemuri/brdoc v3. It also exercises the
  // values at runtime so the test counts toward coverage.
  func TestSignatureParity(t *testing.T) {
  	t.Parallel()
  	var (
  		_ func(string) bool                = IsCPF
  		_ func(string) bool                = IsCNPJ
  		_ func(string) bool                = IsCNH
  		_ func(string) bool                = IsPIS
  		_ func(string) bool                = IsRENAVAM
  		_ func(string) bool                = IsVoterID
  		_ func(string) bool                = IsCNS
  		_ func(string) bool                = IsPlate
  		_ func(string) bool                = IsNationalPlate
  		_ func(string) bool                = IsMercosulPlate
  		_ func(string) (bool, UF)          = IsCEP
  		_ func(string, ...UF) bool         = IsCEPFrom
  		_ func(string) (bool, UF)          = IsPhone
  		_ func(string, ...UF) bool         = IsPhoneFrom
  		_ func(string, UF) (bool, error)   = IsRG
  	)
  	// UF must be the SAME type as brdoc.UF (alias, not a defined type), so a
  	// brdoc.UF is assignable to compat.UF with no conversion.
  	var u UF = brdoc.UFSP
  	assert.Equal(t, "SP", string(u))
  }
  ```

- [ ] **Step 2: Run the guard test and watch it PASS.** Run:
  ```
  go test ./compat/ -run TestSignatureParity -v
  ```
  Expected PASS: `--- PASS: TestSignatureParity`. (If any wrapper signature is wrong, the build fails here with a `cannot use X (...) as func(...) value` error.)

- [ ] **Step 3: Run go vet on the compat package.** Run:
  ```
  go vet ./compat/
  ```
  Expected: no output (clean).

- [ ] **Step 4: Run the full compat suite with coverage and confirm >=80%.** Run:
  ```
  go test ./compat/ -cover
  ```
  Expected PASS: `ok  github.com/inovacc/brdoc/compat` with `coverage: 100.0% of statements` (every wrapper is a one-liner exercised by the parity tests; the gate is >=80%).

- [ ] **Step 5: Run golangci-lint on the new package.** Run:
  ```
  golangci-lint run ./compat/ --timeout=5m
  ```
  Expected: `0 issues`.

- [ ] **Step 6: Commit.** Run:
  ```
  git add compat/compat_test.go
  git commit -m "test(compat): add compile-time signature-parity guard and coverage gate"
  ```


---

## Milestone M3 — MCP server

> Deliverable: `mcp/server.go` (go-sdk `github.com/modelcontextprotocol/go-sdk/mcp`) exposing the registry as agent tools over stdio, a `brdoc mcp` Cobra subcommand wired into `cmd/brdoc/main.go`, and `mcp/server_test.go` using in-memory transports.
>
> The MCP adapter is **derived from the registry** (`brdoc.Kinds()`, `brdoc.Validate/Generate/Format/Detect`, capability assertions `OriginResolver`/`UFScoped`) — never hand-duplicated per type. The root package imports neither adapter; `mcp/` imports the root package `github.com/inovacc/brdoc`.
>
> go-sdk surface used (locked to the v1.x three-arg typed-handler API):
> - `mcp.NewServer(&mcp.Implementation{Name, Version}, *mcp.ServerOptions) *mcp.Server`
> - `mcp.AddTool[In, Out any](s *mcp.Server, t *mcp.Tool, h mcp.ToolHandlerFor[In, Out])` where the handler is `func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error)`
> - `(*mcp.Server).Run(ctx, &mcp.StdioTransport{}) error` (blocks)
> - `mcp.NewInMemoryTransports() (t1, t2 *mcp.InMemoryTransport)`
> - `(*mcp.Server).Connect(ctx, t1, nil) (*mcp.ServerSession, error)`
> - `mcp.NewClient(&mcp.Implementation{Name, Version}, *mcp.ClientOptions) *mcp.Client`; `(*mcp.Client).Connect(ctx, t2, nil) (*mcp.ClientSession, error)`
> - `(*mcp.ClientSession).CallTool(ctx, &mcp.CallToolParams{Name, Arguments map[string]any}) (*mcp.CallToolResult, error)`
> - Result fields used: `result.IsError bool`, `result.Content []mcp.Content`, `*mcp.TextContent{Text}`, `result.StructuredContent any`.
> - `jsonschema` struct tags supply field descriptions/enums on typed inputs.

### Task M3-1: Add the go-sdk dependency

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/go.mod` (require block)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/go.sum` (generated)

**Interfaces:**
- Consumes: nothing.
- Produces: module dependency `github.com/modelcontextprotocol/go-sdk v1.2.0` available to import as `github.com/modelcontextprotocol/go-sdk/mcp`.

- [ ] **Step 1: Add the require.** Run the exact command:
  ```
  go get github.com/modelcontextprotocol/go-sdk@v1.2.0
  ```
  Expected: `go.mod` gains `require github.com/modelcontextprotocol/go-sdk v1.2.0` (plus its transitive deps, e.g. `github.com/google/jsonschema-go`, marked `// indirect`); `go.sum` updated.

- [ ] **Step 2: Verify it resolves.** Run:
  ```
  go list -m github.com/modelcontextprotocol/go-sdk
  ```
  Expected output: `github.com/modelcontextprotocol/go-sdk v1.2.0`.

- [ ] **Step 3: Tidy.** Run:
  ```
  go mod tidy
  ```
  Expected: exit 0, no errors. (The dep is unused until M3-2 adds the import; `go mod tidy` keeps it because it will be imported by the end of this milestone — if tidy drops it, that is fine; M3-2 re-adds it on first build. Do NOT remove the entry manually.)

- [ ] **Step 4: Commit.** Run:
  ```
  git add go.mod go.sum
  git commit -m "chore: add modelcontextprotocol/go-sdk dependency for MCP server"
  ```

### Task M3-2: MCP server package — typed inputs, builder, and tool handlers

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/mcp/server.go`

**Interfaces:**
- Consumes (from M0 frozen contract, root package `github.com/inovacc/brdoc`):
  - `func Kinds() []Kind`
  - `func Validate(kind Kind, value string) (bool, error)`
  - `func Generate(kind Kind) (string, error)`
  - `func Format(kind Kind, value string) (string, error)`
  - `func Detect(value string) (Kind, bool)`
  - `func Get(kind Kind) (Document, bool)`
  - `type Kind string`; `func (k Kind) String() string`
  - `type UF string`; `func (u UF) Valid() bool`
  - `type OriginResolver interface { Origin(value string) (string, error) }`
  - `type UFScoped interface { ValidateUF(value string, uf UF) (bool, error); ImplementedUFs() []UF }`
  - `const MCPServerName = "brdoc"` (from `meta.go`, M0-5)
- Consumes (from go-sdk, M3-1): the `mcp` API listed in the milestone preamble.
- Produces (later tasks rely on these exact names):
  - `func NewServer(version string) *mcp.Server` — builds and registers all 5 tools.
  - `func Serve(ctx context.Context, version string) error` — runs the server over `mcp.StdioTransport` (blocks).
  - Input/output structs: `ValidateInput`, `ValidateOutput`, `GenerateInput`, `GenerateOutput`, `FormatInput`, `FormatOutput`, `DetectInput`, `DetectOutput`, `ListInput`, `ListOutput`.
  - `func kindEnum() []any` — enum values (one per registered kind) for the `kind` jsonschema field.

- [ ] **Step 1: Create the package directory.** Run:
  ```
  mkdir -p D:/weaver-sync/development/personal/projects/brdoc/mcp
  ```

- [ ] **Step 2: Write the full server implementation.** Create `D:/weaver-sync/development/personal/projects/brdoc/mcp/server.go` with this exact content:
  ```go
  // Package mcp adapts the brdoc registry to a Model Context Protocol server.
  //
  // It exposes five tools (validate_document, generate_document,
  // format_document, detect_document, list_document_types) over stdio.
  // Every tool is derived from the brdoc registry, so adding a new document
  // type to the registry automatically widens the MCP surface with no edits
  // here.
  package mcp

  import (
  	"context"
  	"fmt"
  	"log/slog"
  	"os"

  	brdoc "github.com/inovacc/brdoc"
  	"github.com/modelcontextprotocol/go-sdk/mcp"
  )

  // ValidateInput is the typed input for the validate_document tool.
  type ValidateInput struct {
  	Kind  string `json:"kind" jsonschema:"document kind, e.g. cpf or cnpj"`
  	Value string `json:"value" jsonschema:"the document value to validate"`
  	UF    string `json:"uf,omitempty" jsonschema:"federative unit, only used for kind rg, e.g. SP"`
  }

  // ValidateOutput is the typed output for the validate_document tool.
  type ValidateOutput struct {
  	Valid  bool   `json:"valid" jsonschema:"true when the value is a valid document of the given kind"`
  	Origin string `json:"origin,omitempty" jsonschema:"geographic origin when the kind supports it (cpf region, cep/phone/voter_id UF)"`
  }

  // GenerateInput is the typed input for the generate_document tool.
  type GenerateInput struct {
  	Kind  string `json:"kind" jsonschema:"document kind to generate"`
  	Count int    `json:"count,omitempty" jsonschema:"how many values to generate, defaults to 1"`
  }

  // GenerateOutput is the typed output for the generate_document tool.
  type GenerateOutput struct {
  	Values []string `json:"values" jsonschema:"the generated, valid document values"`
  }

  // FormatInput is the typed input for the format_document tool.
  type FormatInput struct {
  	Kind  string `json:"kind" jsonschema:"document kind"`
  	Value string `json:"value" jsonschema:"the document value to format with its canonical mask"`
  }

  // FormatOutput is the typed output for the format_document tool.
  type FormatOutput struct {
  	Formatted string `json:"formatted" jsonschema:"the value rendered with its canonical mask"`
  }

  // DetectInput is the typed input for the detect_document tool.
  type DetectInput struct {
  	Value string `json:"value" jsonschema:"a document value of unknown kind"`
  }

  // DetectOutput is the typed output for the detect_document tool.
  type DetectOutput struct {
  	Kind  string `json:"kind" jsonschema:"the detected document kind, empty when unknown"`
  	Valid bool   `json:"valid" jsonschema:"true when a kind was detected and the value validates"`
  }

  // ListInput is the (empty) typed input for the list_document_types tool.
  type ListInput struct{}

  // ListOutput is the typed output for the list_document_types tool.
  type ListOutput struct {
  	Kinds []string `json:"kinds" jsonschema:"all document kinds the server supports"`
  }

  // kindEnum returns one enum value per registered kind, for the jsonschema
  // "kind" field. Sourced from the registry so it stays in sync automatically.
  func kindEnum() []any {
  	kinds := brdoc.Kinds()
  	out := make([]any, 0, len(kinds))
  	for _, k := range kinds {
  		out = append(out, k.String())
  	}
  	return out
  }

  // errResult builds a tool result flagged as an error with a human-readable
  // message. The typed Out zero value is returned alongside.
  func errResult[Out any](msg string) (*mcp.CallToolResult, Out, error) {
  	var zero Out
  	return &mcp.CallToolResult{
  		IsError: true,
  		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
  	}, zero, nil
  }

  func validateHandler(_ context.Context, _ *mcp.CallToolRequest, in ValidateInput) (*mcp.CallToolResult, ValidateOutput, error) {
  	kind := brdoc.Kind(in.Kind)
  	doc, ok := brdoc.Get(kind)
  	if !ok {
  		return errResult[ValidateOutput](fmt.Sprintf("unknown document kind %q", in.Kind))
  	}

  	var out ValidateOutput
  	if in.UF != "" {
  		scoped, isScoped := doc.(brdoc.UFScoped)
  		if !isScoped {
  			return errResult[ValidateOutput](fmt.Sprintf("kind %q does not accept a uf", in.Kind))
  		}
  		valid, err := scoped.ValidateUF(in.Value, brdoc.UF(in.UF))
  		if err != nil {
  			return errResult[ValidateOutput](err.Error())
  		}
  		out.Valid = valid
  	} else {
  		valid, err := brdoc.Validate(kind, in.Value)
  		if err != nil {
  			return errResult[ValidateOutput](err.Error())
  		}
  		out.Valid = valid
  	}

  	if res, hasOrigin := doc.(brdoc.OriginResolver); hasOrigin && out.Valid {
  		if origin, err := res.Origin(in.Value); err == nil {
  			out.Origin = origin
  		}
  	}

  	return &mcp.CallToolResult{StructuredContent: out}, out, nil
  }

  func generateHandler(_ context.Context, _ *mcp.CallToolRequest, in GenerateInput) (*mcp.CallToolResult, GenerateOutput, error) {
  	count := in.Count
  	if count <= 0 {
  		count = 1
  	}

  	values := make([]string, 0, count)
  	for i := 0; i < count; i++ {
  		v, err := brdoc.Generate(brdoc.Kind(in.Kind))
  		if err != nil {
  			return errResult[GenerateOutput](err.Error())
  		}
  		values = append(values, v)
  	}

  	out := GenerateOutput{Values: values}
  	return &mcp.CallToolResult{StructuredContent: out}, out, nil
  }

  func formatHandler(_ context.Context, _ *mcp.CallToolRequest, in FormatInput) (*mcp.CallToolResult, FormatOutput, error) {
  	formatted, err := brdoc.Format(brdoc.Kind(in.Kind), in.Value)
  	if err != nil {
  		return errResult[FormatOutput](err.Error())
  	}
  	out := FormatOutput{Formatted: formatted}
  	return &mcp.CallToolResult{StructuredContent: out}, out, nil
  }

  func detectHandler(_ context.Context, _ *mcp.CallToolRequest, in DetectInput) (*mcp.CallToolResult, DetectOutput, error) {
  	kind, ok := brdoc.Detect(in.Value)
  	out := DetectOutput{Kind: kind.String(), Valid: ok}
  	if !ok {
  		out.Kind = ""
  	}
  	return &mcp.CallToolResult{StructuredContent: out}, out, nil
  }

  func listHandler(_ context.Context, _ *mcp.CallToolRequest, _ ListInput) (*mcp.CallToolResult, ListOutput, error) {
  	kinds := brdoc.Kinds()
  	names := make([]string, 0, len(kinds))
  	for _, k := range kinds {
  		names = append(names, k.String())
  	}
  	out := ListOutput{Kinds: names}
  	return &mcp.CallToolResult{StructuredContent: out}, out, nil
  }

  // NewServer builds an MCP server with all five brdoc tools registered.
  // version is stamped into the server Implementation (use build info).
  func NewServer(version string) *mcp.Server {
  	if version == "" {
  		version = "dev"
  	}

  	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
  	srv := mcp.NewServer(
  		&mcp.Implementation{Name: brdoc.MCPServerName, Version: version},
  		&mcp.ServerOptions{Logger: logger},
  	)

  	mcp.AddTool(srv, &mcp.Tool{
  		Name:        "validate_document",
  		Description: "Validate a Brazilian document of a given kind; returns valid and optional origin.",
  	}, validateHandler)

  	mcp.AddTool(srv, &mcp.Tool{
  		Name:        "generate_document",
  		Description: "Generate one or more valid Brazilian documents of a given kind.",
  	}, generateHandler)

  	mcp.AddTool(srv, &mcp.Tool{
  		Name:        "format_document",
  		Description: "Format a Brazilian document with its canonical mask.",
  	}, formatHandler)

  	mcp.AddTool(srv, &mcp.Tool{
  		Name:        "detect_document",
  		Description: "Detect the kind of an unknown Brazilian document value.",
  	}, detectHandler)

  	mcp.AddTool(srv, &mcp.Tool{
  		Name:        "list_document_types",
  		Description: "List every Brazilian document kind this server supports.",
  	}, listHandler)

  	return srv
  }

  // Serve runs the MCP server over stdio until the context is cancelled or
  // stdin closes. The logger writes to stderr because stdout carries the
  // JSON-RPC stream.
  func Serve(ctx context.Context, version string) error {
  	srv := NewServer(version)
  	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
  		return fmt.Errorf("brdoc mcp: %w", err)
  	}
  	return nil
  }
  ```
  Note: `kindEnum` is defined and exported-by-package for use as the jsonschema enum source; M3-3 wires it onto the tool input schemas after confirming the go-sdk schema-customization call. Keeping it here avoids re-deriving the kind list.

- [ ] **Step 3: Build the package (expected to compile, no test yet).** Run:
  ```
  go build ./mcp/
  ```
  Expected: exit 0. If the import line is missing in `go.mod` (because M3-1 tidy dropped it), this build re-resolves it — re-run `go mod tidy` and rebuild. Expected PASS: no output.

- [ ] **Step 4: Vet.** Run:
  ```
  go vet ./mcp/
  ```
  Expected: exit 0, no diagnostics.

- [ ] **Step 5: Commit.** Run:
  ```
  git add mcp/server.go go.mod go.sum
  git commit -m "feat(mcp): registry-derived MCP server with five document tools"
  ```

### Task M3-3: In-memory transport tests — every tool round-trips

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/mcp/server_test.go`

**Interfaces:**
- Consumes:
  - `func NewServer(version string) *mcp.Server` (M3-2)
  - go-sdk: `mcp.NewInMemoryTransports`, `mcp.NewClient`, `(*mcp.Server).Connect`, `(*mcp.Client).Connect`, `(*mcp.ClientSession).CallTool`, `mcp.CallToolParams`, `mcp.CallToolResult{IsError, Content}`, `*mcp.TextContent`.
  - root package: `brdoc.NewCPF()`, `brdoc.Validate`, `brdoc.Kinds()` (only for asserting expectations).
- Produces: test coverage for the five tools; no new exported symbols.

- [ ] **Step 1: Write the failing test file.** Create `D:/weaver-sync/development/personal/projects/brdoc/mcp/server_test.go` with this exact content:
  ```go
  package mcp

  import (
  	"context"
  	"encoding/json"
  	"testing"

  	brdoc "github.com/inovacc/brdoc"
  	"github.com/modelcontextprotocol/go-sdk/mcp"
  	"github.com/stretchr/testify/assert"
  	"github.com/stretchr/testify/require"
  )

  // newTestSession spins up the server and a client over in-memory transports
  // and returns a connected client session. Cleanup is registered on t.
  func newTestSession(t *testing.T) (context.Context, *mcp.ClientSession) {
  	t.Helper()
  	ctx := context.Background()

  	st, ct := mcp.NewInMemoryTransports()

  	srv := NewServer("test")
  	ss, err := srv.Connect(ctx, st, nil)
  	require.NoError(t, err)
  	t.Cleanup(func() { _ = ss.Close() })

  	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
  	cs, err := client.Connect(ctx, ct, nil)
  	require.NoError(t, err)
  	t.Cleanup(func() { _ = cs.Close() })

  	return ctx, cs
  }

  // decodeResult unmarshals the first TextContent of a non-error result into v.
  // The go-sdk serialises StructuredContent into a JSON TextContent, so this
  // works regardless of whether the test reads StructuredContent directly.
  func decodeResult(t *testing.T, res *mcp.CallToolResult, v any) {
  	t.Helper()
  	require.False(t, res.IsError, "tool returned an error result")
  	require.NotEmpty(t, res.Content, "result has no content")
  	tc, ok := res.Content[0].(*mcp.TextContent)
  	require.True(t, ok, "first content is not TextContent")
  	require.NoError(t, json.Unmarshal([]byte(tc.Text), v))
  }

  func TestValidateDocumentTool(t *testing.T) {
  	ctx, cs := newTestSession(t)

  	// A freshly generated CPF is always valid; use the registry to source one.
  	cpf := brdoc.NewCPF().Generate()
  	require.True(t, brdoc.NewCPF().Validate(cpf), "generated CPF must validate")

  	tests := []struct {
  		name      string
  		args      map[string]any
  		wantValid bool
  		wantErr   bool
  	}{
  		{
  			name:      "valid cpf",
  			args:      map[string]any{"kind": "cpf", "value": cpf},
  			wantValid: true,
  		},
  		{
  			name:      "invalid cpf all equal",
  			args:      map[string]any{"kind": "cpf", "value": "11111111111"},
  			wantValid: false,
  		},
  		{
  			name:      "cnpj regression sample 39591842000010",
  			args:      map[string]any{"kind": "cnpj", "value": "39591842000010"},
  			wantValid: true,
  		},
  		{
  			name:    "unknown kind is error result",
  			args:    map[string]any{"kind": "bogus", "value": "x"},
  			wantErr: true,
  		},
  	}

  	for _, tt := range tests {
  		t.Run(tt.name, func(t *testing.T) {
  			res, err := cs.CallTool(ctx, &mcp.CallToolParams{
  				Name:      "validate_document",
  				Arguments: tt.args,
  			})
  			require.NoError(t, err)

  			if tt.wantErr {
  				assert.True(t, res.IsError)
  				return
  			}

  			var out ValidateOutput
  			decodeResult(t, res, &out)
  			assert.Equal(t, tt.wantValid, out.Valid)
  		})
  	}
  }

  func TestGenerateDocumentTool(t *testing.T) {
  	ctx, cs := newTestSession(t)

  	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
  		Name:      "generate_document",
  		Arguments: map[string]any{"kind": "cpf", "count": 3},
  	})
  	require.NoError(t, err)

  	var out GenerateOutput
  	decodeResult(t, res, &out)
  	require.Len(t, out.Values, 3)
  	for _, v := range out.Values {
  		assert.True(t, brdoc.NewCPF().Validate(v), "generated %q must validate", v)
  	}

  	// count omitted -> defaults to 1.
  	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
  		Name:      "generate_document",
  		Arguments: map[string]any{"kind": "cnpj"},
  	})
  	require.NoError(t, err)
  	var one GenerateOutput
  	decodeResult(t, res, &one)
  	assert.Len(t, one.Values, 1)

  	// unknown kind -> error result.
  	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
  		Name:      "generate_document",
  		Arguments: map[string]any{"kind": "bogus"},
  	})
  	require.NoError(t, err)
  	assert.True(t, res.IsError)
  }

  func TestFormatDocumentTool(t *testing.T) {
  	ctx, cs := newTestSession(t)

  	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
  		Name:      "format_document",
  		Arguments: map[string]any{"kind": "cpf", "value": "11144477735"},
  	})
  	require.NoError(t, err)

  	var out FormatOutput
  	decodeResult(t, res, &out)
  	assert.Equal(t, "111.444.777-35", out.Formatted)

  	// bad length -> error result (ErrInvalidLength surfaced as message).
  	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
  		Name:      "format_document",
  		Arguments: map[string]any{"kind": "cpf", "value": "123"},
  	})
  	require.NoError(t, err)
  	assert.True(t, res.IsError)
  }

  func TestDetectDocumentTool(t *testing.T) {
  	ctx, cs := newTestSession(t)

  	tests := []struct {
  		name      string
  		value     string
  		wantKind  string
  		wantValid bool
  	}{
  		{name: "cpf length", value: "11144477735", wantKind: "cpf", wantValid: true},
  		{name: "cnpj length", value: "39591842000010", wantKind: "cnpj", wantValid: true},
  		{name: "unknown length", value: "12345", wantKind: "", wantValid: false},
  	}

  	for _, tt := range tests {
  		t.Run(tt.name, func(t *testing.T) {
  			res, err := cs.CallTool(ctx, &mcp.CallToolParams{
  				Name:      "detect_document",
  				Arguments: map[string]any{"value": tt.value},
  			})
  			require.NoError(t, err)

  			var out DetectOutput
  			decodeResult(t, res, &out)
  			assert.Equal(t, tt.wantKind, out.Kind)
  			assert.Equal(t, tt.wantValid, out.Valid)
  		})
  	}
  }

  func TestListDocumentTypesTool(t *testing.T) {
  	ctx, cs := newTestSession(t)

  	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
  		Name:      "list_document_types",
  		Arguments: map[string]any{},
  	})
  	require.NoError(t, err)

  	var out ListOutput
  	decodeResult(t, res, &out)

  	want := make([]string, 0, len(brdoc.Kinds()))
  	for _, k := range brdoc.Kinds() {
  		want = append(want, k.String())
  	}
  	assert.Equal(t, want, out.Kinds)
  	assert.Contains(t, out.Kinds, "cpf")
  	assert.Contains(t, out.Kinds, "cnpj")
  }

  func TestKindEnumMatchesRegistry(t *testing.T) {
  	enum := kindEnum()
  	require.Len(t, enum, len(brdoc.Kinds()))
  	assert.Equal(t, brdoc.Kinds()[0].String(), enum[0])
  }
  ```

- [ ] **Step 2: Run the test, expecting it to compile and pass against M3-2's implementation.** Run:
  ```
  go test ./mcp/ -run Test -v
  ```
  Expected PASS: `ok  github.com/inovacc/brdoc/mcp` with `--- PASS:` for `TestValidateDocumentTool`, `TestGenerateDocumentTool`, `TestFormatDocumentTool`, `TestDetectDocumentTool`, `TestListDocumentTypesTool`, `TestKindEnumMatchesRegistry`.
  - If `decodeResult` fails with "first content is not TextContent" or empty `tc.Text`, the go-sdk in use serialises structured output differently: change `decodeResult` to read `res.StructuredContent` instead — replace its body with:
    ```go
    require.False(t, res.IsError, "tool returned an error result")
    require.NotNil(t, res.StructuredContent)
    b, err := json.Marshal(res.StructuredContent)
    require.NoError(t, err)
    require.NoError(t, json.Unmarshal(b, v))
    ```
    Then re-run the same command; expected PASS. (Both paths are valid go-sdk behaviour; pick whichever the linked version emits.)

- [ ] **Step 3: Confirm coverage of the package.** Run:
  ```
  go test ./mcp/ -cover
  ```
  Expected: `coverage:` reported >= 80% for the package.

- [ ] **Step 4: Commit.** Run:
  ```
  git add mcp/server_test.go
  git commit -m "test(mcp): in-memory round-trip tests for all five tools"
  ```

### Task M3-4: Wire the `brdoc mcp` Cobra subcommand

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/mcp.go`
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/main.go` (the `newRootCmd()` body from Task M1-4 — add one `root.AddCommand(newMCPCmd())` line after the existing `newDetectCmd()`/`newVersionCmd()` registrations)

**Interfaces:**
- Consumes:
  - `func Serve(ctx context.Context, version string) error` (M3-2, package `github.com/inovacc/brdoc/mcp`)
  - existing CLI (frozen by M1-4): `func newRootCmd() *cobra.Command`, which calls `registerKindCommands(root)`, `root.AddCommand(newDetectCmd())`, and `root.AddCommand(newVersionCmd())`. The root is built via `Execute()` in `main()`.
- Produces: top-level command `brdoc mcp` registered inside `newRootCmd()`; `func newMCPCmd() *cobra.Command` (factory form, matching the M1 `newDetectCmd`/`newVersionCmd` style — there is no package-level `var rootCmd`/`var mcpCmd` after the M1-4 rewrite).

- [ ] **Step 1: Create the subcommand file.** Create `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/mcp.go` with this exact content:
  ```go
  package main

  import (
  	"os/signal"
  	"syscall"

  	mcpserver "github.com/inovacc/brdoc/mcp"
  	"github.com/spf13/cobra"
  )

  // newMCPCmd builds the top-level "mcp" command, which runs the brdoc MCP
  // server over stdio. It matches the M1-4 factory style (newDetectCmd /
  // newVersionCmd); the version string is resolved from the shared version()
  // helper defined in version.go (M1-4).
  func newMCPCmd() *cobra.Command {
  	return &cobra.Command{
  		Use:   "mcp",
  		Short: "Run brdoc as a Model Context Protocol server over stdio",
  		Long: "Start an MCP server exposing brdoc's validate, generate, format, " +
  			"detect, and list tools to agents over stdin/stdout. Logs go to stderr.",
  		Args: cobra.NoArgs,
  		RunE: func(cmd *cobra.Command, _ []string) error {
  			// Cancel on SIGINT/SIGTERM so the stdio loop exits cleanly.
  			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
  			defer stop()
  			return mcpserver.Serve(ctx, version())
  		},
  	}
  }
  ```
  Note: `cmd.Context()` is non-nil — cobra defaults it to `context.Background()` when `main()` calls `Execute()` (the M1-4 entrypoint). If a future task switches to `ExecuteContext`, this code still works. The MCP server version is sourced from the same `version()` helper (version.go, M1-4) the `version` subcommand uses, so there is no separate `mcpVersion` var.

- [ ] **Step 2: Register the command inside `newRootCmd()`.** In `D:/weaver-sync/development/personal/projects/brdoc/cmd/brdoc/main.go`, locate the registration block written in Task M1-4:
  ```go
  	registerKindCommands(root)
  	root.AddCommand(newDetectCmd())
  	root.AddCommand(newVersionCmd())
  ```
  and add immediately after the last line:
  ```go
  	root.AddCommand(newMCPCmd())
  ```
  `mcp` is a top-level command, not a per-kind one, so it is added once after the registry-driven `registerKindCommands(root)` loop — never inside it.

- [ ] **Step 3: Build the CLI.** Run:
  ```
  go build ./cmd/brdoc/
  ```
  Expected: exit 0, no output. `signal.NotifyContext` returns the cancellable context, so `context` need not be imported directly; `os/signal` and `syscall` are the only stdlib imports in mcp.go.

- [ ] **Step 4: Smoke-test the command is registered.** Run:
  ```
  go run ./cmd/brdoc --help
  ```
  Expected: the help output lists `mcp   Run brdoc as a Model Context Protocol server over stdio` among the available commands. (Do NOT run `go run ./cmd/brdoc mcp` interactively — it blocks on stdin waiting for JSON-RPC; the in-memory tests in M3-3 already prove the server works.)

- [ ] **Step 5: Vet the whole module.** Run:
  ```
  go vet ./...
  ```
  Expected: exit 0, no diagnostics.

- [ ] **Step 6: Commit.** Run:
  ```
  git add cmd/brdoc/mcp.go cmd/brdoc/main.go
  git commit -m "feat(cli): add brdoc mcp subcommand to start the MCP stdio server"
  ```

### Task M3-5: Milestone gate — full build, test, and lint

**Files:**
- Modify: none (verification only).

**Interfaces:**
- Consumes: everything produced in M3-1 through M3-4.
- Produces: a green milestone checkpoint.

- [ ] **Step 1: Run the full module test suite.** Run:
  ```
  go test ./...
  ```
  Expected: `ok` for `github.com/inovacc/brdoc`, `github.com/inovacc/brdoc/mcp`, and any other packages; no `FAIL` lines.

- [ ] **Step 2: Lint.** Run:
  ```
  golangci-lint run ./... --timeout=5m
  ```
  Expected: exit 0, `0 issues`. (If the linter flags the generic `errResult[Out any]` helper for an unused type parameter in any call, it is a false positive — every call site instantiates `Out`; do not delete it.)

- [ ] **Step 3: Confirm no stray dependency drift.** Run:
  ```
  go mod tidy
  git diff --exit-code go.mod go.sum
  ```
  Expected: exit 0 (no diff) — `go-sdk` and its transitive deps are already recorded.

- [ ] **Step 4: Final milestone commit (only if Step 3 produced changes).** If `go mod tidy` changed anything, run:
  ```
  git add go.mod go.sum
  git commit -m "chore(mcp): tidy module dependencies after MCP milestone"
  ```
  Otherwise skip. M3 is complete: `brdoc mcp` serves the five registry-derived tools over stdio with passing in-memory transport tests.


---

## Milestone M4 — PIX key validation

This milestone creates `pix.go` (root package `brdoc`) implementing the frozen `Document`
interface for PIX keys, plus an exported `DetectPIXKind` classifier and a full
table-driven `pix_test.go`. A PIX key is valid if it is a well-formed key of ANY of the 5
BCB key kinds:

1. **CPF** — 11-digit value that passes the existing `*CPF` validator.
2. **CNPJ** — 14-char value that passes the existing `*CNPJ` validator (alphanumeric).
3. **Email** — RFC 5322-lite regex (single `@`, dotted domain, sane local part).
4. **Phone** — E.164 Brazilian form: `+55` followed by a 2-digit DDD and an 8- or 9-digit
   subscriber number (total 10 or 11 digits after `+55`). PIX requires the strict E.164
   prefix `+55`, so this check is done in `pix.go` directly (it does NOT relax to the
   looser formats the M2 `phone.go` validator may accept).
5. **EVP** (Endereço Virtual de Pagamento) — a random key in canonical UUIDv4 form
   (`xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx`, variant `8/9/a/b`).

`Generate` returns a freshly minted EVP key (a valid UUIDv4) — itself always a valid PIX
key, so `Generate -> Validate` round-trips true. `Format` is identity over the cleaned
value (PIX keys have no canonical mask; CPF/CNPJ/email/phone keys are kept verbatim because
the key string is what gets stored at the bank). `PIX` self-registers via `init()`.

> Consumes from M0: `Kind`, `KindPIX`, `Document` interface, `Register`, `onlyDigits`,
> sentinel `ErrInvalidLength`. Consumes from M0-6/M0-7: `NewCPF`, `(*CPF).Validate`,
> `NewCNPJ`, `(*CNPJ).Validate`. No dependency on M2 `phone.go` (E.164 check is local).

---

### Task M4-1: PIX key kinds, classifier, and the EVP/email/phone regexes

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/pix.go`
- Create: `D:/weaver-sync/development/personal/projects/brdoc/pix_test.go`

**Interfaces:**
- Consumes:
  - `func onlyDigits(s string) string` (M0-8)
  - `func NewCPF() *CPF` / `func (c *CPF) Validate(value string) bool` (M0-6)
  - `func NewCNPJ() *CNPJ` / `func (c *CNPJ) Validate(value string) bool` (M0-7)
- Produces:
  - `const ( PIXKindCPF = "cpf"; PIXKindCNPJ = "cnpj"; PIXKindEmail = "email"; PIXKindPhone = "phone"; PIXKindEVP = "evp" )`
  - `func DetectPIXKind(value string) (string, bool)` — returns one of the five PIXKind*
    strings + true when `value` is a well-formed key of that kind; `("", false)` otherwise.

- [ ] **Step 1: Write failing classifier test.** Append to `pix_test.go`:
```go
package brdoc

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectPIXKind(t *testing.T) {
	t.Parallel()

	// Real valid samples (generated/known-good for each kind).
	const (
		validCPF   = "52998224725"                            // passes CPF check digits
		validCNPJ  = "39591842000010"                          // paemuri regression sample
		validEmail = "joao.silva@example.com.br"
		validPhone = "+5511998765432"                          // +55, DDD 11, 9-digit mobile
		validEVP   = "123e4567-e89b-42d3-a456-426614174000"   // UUIDv4 shape
	)

	tests := []struct {
		name     string
		value    string
		wantKind string
		wantOK   bool
	}{
		{"cpf key", validCPF, PIXKindCPF, true},
		{"cpf key formatted", "529.982.247-25", PIXKindCPF, true},
		{"cnpj key", validCNPJ, PIXKindCNPJ, true},
		{"cnpj key formatted", "39.591.842/0001-10", PIXKindCNPJ, true},
		{"email key", validEmail, PIXKindEmail, true},
		{"email key simple", "a@b.co", PIXKindEmail, true},
		{"phone key 9 digit", validPhone, PIXKindPhone, true},
		{"phone key 8 digit", "+551133224455", PIXKindPhone, true},
		{"evp key", validEVP, PIXKindEVP, true},
		{"evp key uppercase", "123E4567-E89B-42D3-A456-426614174000", PIXKindEVP, true},
		{"empty", "", "", false},
		{"all-equal cpf rejected", "11111111111", "", false},
		{"off-by-one cpf dv", "52998224724", "", false},
		{"email no domain dot", "joao@example", "", false},
		{"email double at", "a@@b.co", "", false},
		{"phone no plus55", "11998765432", "", false},
		{"phone wrong country", "+1198765432", "", false},
		{"phone too short", "+55119987", "", false},
		{"evp wrong version", "123e4567-e89b-12d3-a456-426614174000", "", false},
		{"evp wrong variant", "123e4567-e89b-42d3-c456-426614174000", "", false},
		{"random junk", "not-a-key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotKind, gotOK := DetectPIXKind(tt.value)
			assert.Equal(t, tt.wantOK, gotOK)
			assert.Equal(t, tt.wantKind, gotKind)
		})
	}
}

// compile-time anchor so the regexp import is used before impl lands.
var _ = regexp.MustCompile
```

- [ ] **Step 2: Run the failing test (expect compile failure).**
  Command: `go test -run TestDetectPIXKind ./...`
  Expected: FAIL — `undefined: DetectPIXKind`, `undefined: PIXKindCPF` (and the other PIXKind* consts).

- [ ] **Step 3: Create `pix.go` with constants, regexes, and `DetectPIXKind`.** Write the
  full file:
```go
package brdoc

import (
	"regexp"
	"strings"
)

// PIX key-kind identifiers returned by DetectPIXKind.
const (
	PIXKindCPF   = "cpf"
	PIXKindCNPJ  = "cnpj"
	PIXKindEmail = "email"
	PIXKindPhone = "phone"
	PIXKindEVP   = "evp"
)

// pixEmailRe is an RFC 5322-lite matcher: a sane local part, a single '@', and a
// dotted domain. It deliberately rejects consecutive '@', missing domain dot, and
// leading/trailing dots in the domain.
var pixEmailRe = regexp.MustCompile(
	`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?)+$`,
)

// pixPhoneRe matches the strict E.164 Brazilian form required by PIX: "+55", a
// 2-digit DDD, then an 8- or 9-digit subscriber number (10 or 11 trailing digits).
var pixPhoneRe = regexp.MustCompile(`^\+55\d{10,11}$`)

// pixEVPRe matches a canonical UUIDv4 (version nibble 4, variant nibble 8/9/a/b).
var pixEVPRe = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`,
)

// DetectPIXKind reports which of the five BCB PIX key kinds value is, and whether it
// is a well-formed key at all. The five kinds are checked in a deterministic order:
// EVP, email, phone, then CPF/CNPJ by digit length. It returns ("", false) when value
// is not a well-formed key of any kind.
func DetectPIXKind(value string) (string, bool) {
	v := strings.TrimSpace(value)

	// EVP (UUIDv4) — most specific shape, checked first.
	if pixEVPRe.MatchString(v) {
		return PIXKindEVP, true
	}

	// Email — contains '@' and matches the RFC5322-lite shape.
	if strings.Contains(v, "@") {
		if pixEmailRe.MatchString(v) {
			return PIXKindEmail, true
		}
		return "", false
	}

	// Phone — strict E.164 "+55..." form.
	if strings.HasPrefix(v, "+") {
		if pixPhoneRe.MatchString(v) {
			return PIXKindPhone, true
		}
		return "", false
	}

	// CPF / CNPJ — discriminate by digit count, then run the real check-digit validator.
	switch len(onlyDigits(v)) {
	case CpfLength:
		if NewCPF().Validate(v) {
			return PIXKindCPF, true
		}
	case CnpjLength:
		if NewCNPJ().Validate(v) {
			return PIXKindCNPJ, true
		}
	}

	return "", false
}
```

- [ ] **Step 4: Run the classifier test (expect PASS).**
  Command: `go test -run TestDetectPIXKind ./...`
  Expected: PASS — `ok  github.com/inovacc/brdoc`.

- [ ] **Step 5: Commit.**
```
git add pix.go pix_test.go
git commit -m "feat: add PIX key kind classifier (DetectPIXKind) with CPF/CNPJ/email/phone/EVP"
```

---

### Task M4-2: `PIX` type implementing `Document` (Kind/Validate/Format)

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/pix.go` (append `PIX` type + methods)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/pix_test.go` (append type tests)

**Interfaces:**
- Consumes:
  - `func DetectPIXKind(value string) (string, bool)` (M4-1)
  - `type Kind string`, `const KindPIX Kind = "pix"`, `type Document interface{...}` (M0-1)
- Produces:
  - `type PIX struct{}`
  - `func NewPIX() *PIX`
  - `func (p *PIX) Kind() Kind` — returns `KindPIX`
  - `func (p *PIX) Validate(value string) bool` — true iff `DetectPIXKind` reports a kind
  - `func (p *PIX) Format(value string) (string, error)` — identity over the trimmed value;
    `ErrInvalidLength` (wrapped `%w`) when the value is not a valid PIX key

- [ ] **Step 1: Write failing `PIX` method test.** Append to `pix_test.go`:
```go
func TestPIXKind(t *testing.T) {
	t.Parallel()
	assert.Equal(t, KindPIX, NewPIX().Kind())
	assert.Equal(t, "pix", NewPIX().Kind().String())
}

func TestPIXValidate(t *testing.T) {
	t.Parallel()

	p := NewPIX()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid cpf", "52998224725", true},
		{"valid cnpj regression", "39591842000010", true},
		{"valid email", "joao.silva@example.com.br", true},
		{"valid phone e164", "+5511998765432", true},
		{"valid evp uuidv4", "123e4567-e89b-42d3-a456-426614174000", true},
		{"invalid cpf dv", "52998224724", false},
		{"invalid email", "joao@example", false},
		{"invalid phone", "11998765432", false},
		{"invalid evp version", "123e4567-e89b-12d3-a456-426614174000", false},
		{"empty", "", false},
		{"junk", "totally-not-a-pix-key", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Validate(tt.value))
		})
	}
}

func TestPIXFormat(t *testing.T) {
	t.Parallel()

	p := NewPIX()

	t.Run("identity on valid key", func(t *testing.T) {
		t.Parallel()
		out, err := p.Format("  joao.silva@example.com.br  ")
		assert.NoError(t, err)
		assert.Equal(t, "joao.silva@example.com.br", out) // trimmed, otherwise verbatim
	})

	t.Run("identity on valid cpf key keeps mask", func(t *testing.T) {
		t.Parallel()
		out, err := p.Format("529.982.247-25")
		assert.NoError(t, err)
		assert.Equal(t, "529.982.247-25", out) // PIX has no canonical mask; verbatim
	})

	t.Run("error on invalid key", func(t *testing.T) {
		t.Parallel()
		_, err := p.Format("not-a-pix-key")
		assert.ErrorIs(t, err, ErrInvalidLength)
	})
}
```

- [ ] **Step 2: Run the failing test (expect compile failure).**
  Command: `go test -run TestPIX ./...`
  Expected: FAIL — `undefined: NewPIX`, `undefined: PIX`.

- [ ] **Step 3: Append the `PIX` type + methods to `pix.go`.** Add after `DetectPIXKind`:
```go
// PIX validates Brazilian PIX keys across all five BCB key kinds: CPF, CNPJ, email,
// phone (E.164 "+55..."), and EVP (UUIDv4). It implements the Document interface and
// self-registers in init().
type PIX struct{}

// NewPIX creates a new PIX key validator instance.
func NewPIX() *PIX { return &PIX{} }

// Kind returns KindPIX.
func (p *PIX) Kind() Kind { return KindPIX }

// Validate reports whether value is a well-formed PIX key of any of the five kinds.
func (p *PIX) Validate(value string) bool {
	_, ok := DetectPIXKind(value)
	return ok
}

// Format returns the cleaned (whitespace-trimmed) PIX key. PIX keys have no canonical
// mask, so the key is returned verbatim; CPF/CNPJ/email/phone formatting is intentionally
// preserved because the stored key string is what the bank matches. It returns
// ErrInvalidLength (wrapped) when value is not a valid PIX key.
func (p *PIX) Format(value string) (string, error) {
	v := strings.TrimSpace(value)
	if _, ok := DetectPIXKind(v); !ok {
		return "", fmt.Errorf("brdoc: %q is not a valid PIX key: %w", value, ErrInvalidLength)
	}
	return v, nil
}
```

- [ ] **Step 4: Add the `fmt` import to `pix.go`.** Update the import block at the top of
  `pix.go` from:
```go
import (
	"regexp"
	"strings"
)
```
  to:
```go
import (
	"fmt"
	"regexp"
	"strings"
)
```

- [ ] **Step 5: Run the type test (expect PASS).**
  Command: `go test -run TestPIX ./...`
  Expected: PASS — `ok  github.com/inovacc/brdoc`.

- [ ] **Step 6: Commit.**
```
git add pix.go pix_test.go
git commit -m "feat: add PIX type implementing Document (Kind/Validate/Format)"
```

---

### Task M4-3: `Generate` (EVP/UUIDv4) + round-trip + registry registration

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/pix.go` (append `Generate`, `init`)
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/pix_test.go` (append generate + registry tests)

**Interfaces:**
- Consumes:
  - `func Register(d Document)` (M0-4)
  - `func Get(kind Kind) (Document, bool)` (M0-4)
  - `func Validate(kind Kind, value string) (bool, error)` (M0-4)
  - `func (p *PIX) Validate(value string) bool` (M4-2), `pixEVPRe` (M4-1)
- Produces:
  - `func (p *PIX) Generate() string` — returns a random EVP (UUIDv4) PIX key, lowercase,
    canonical 8-4-4-4-12 form; always a valid PIX key
  - registry side effect: `init()` calls `Register(&PIX{})`

- [ ] **Step 1: Write failing generate + registry tests.** Append to `pix_test.go`:
```go
func TestPIXGenerate(t *testing.T) {
	t.Parallel()

	p := NewPIX()

	// Round-trip: every generated key is a valid PIX key, and a valid EVP UUIDv4.
	for i := 0; i < 200; i++ {
		key := p.Generate()

		assert.True(t, p.Validate(key), "generated key must validate: %q", key)
		assert.Truef(t, pixEVPRe.MatchString(key), "generated key must be UUIDv4: %q", key)

		kind, ok := DetectPIXKind(key)
		assert.True(t, ok)
		assert.Equal(t, PIXKindEVP, kind)

		// Canonical UUIDv4: 36 chars, version '4', lowercase hex.
		assert.Len(t, key, 36)
		assert.Equal(t, byte('4'), key[14], "version nibble must be 4: %q", key)
		assert.Equal(t, strings.ToLower(key), key, "generated key must be lowercase: %q", key)
	}
}

func TestPIXGenerateUnique(t *testing.T) {
	t.Parallel()

	p := NewPIX()
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		key := p.Generate()
		_, dup := seen[key]
		assert.Falsef(t, dup, "generated duplicate EVP key: %q", key)
		seen[key] = struct{}{}
	}
}

func TestPIXRegistered(t *testing.T) {
	t.Parallel()

	doc, ok := Get(KindPIX)
	assert.True(t, ok, "PIX must self-register")
	assert.Equal(t, KindPIX, doc.Kind())

	// Round-trip through the registry dispatcher.
	valid, err := Validate(KindPIX, "52998224725")
	assert.NoError(t, err)
	assert.True(t, valid)

	valid, err = Validate(KindPIX, "not-a-key")
	assert.NoError(t, err)
	assert.False(t, valid)
}
```

- [ ] **Step 2: Run the failing test (expect FAIL).**
  Command: `go test -run TestPIXGenerate ./...`
  Expected: FAIL — `p.Generate undefined (type *PIX has no field or method Generate)`.

- [ ] **Step 3: Append `Generate` + `init` to `pix.go`, and add `crypto/rand`.** Update the
  import block to:
```go
import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
)
```
  Then append:
```go
// Generate returns a random EVP (Endereço Virtual de Pagamento) PIX key: a canonical
// lowercase UUIDv4, which is itself always a valid PIX key. Uses crypto/rand so generated
// keys are unpredictable (PIX EVP keys are bank-assigned random identifiers).
func (p *PIX) Generate() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand.Read never fails on supported platforms; fall back deterministically.
		return "00000000-0000-4000-8000-000000000000"
	}

	// Set version (4) and variant (10xx) bits per RFC 4122.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	const hexdigits = "0123456789abcdef"
	var out [36]byte
	pos := 0
	for i, by := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[pos] = '-'
			pos++
		}
		out[pos] = hexdigits[by>>4]
		out[pos+1] = hexdigits[by&0x0f]
		pos += 2
	}

	return string(out[:])
}

func init() { Register(&PIX{}) }
```

- [ ] **Step 4: Run the generate + registry tests (expect PASS).**
  Command: `go test -run TestPIX ./...`
  Expected: PASS — `ok  github.com/inovacc/brdoc` (covers Kind/Validate/Format/Generate/Registered).

- [ ] **Step 5: Run the full package test suite to confirm no regressions.**
  Command: `go test ./...`
  Expected: PASS — all packages `ok` (CPF/CNPJ/registry/PIX all green).

- [ ] **Step 6: Verify the compile-time `Document` assertion holds.** Append to `pix.go`:
```go
// compile-time guarantee that *PIX satisfies the Document interface.
var _ Document = (*PIX)(nil)
```

- [ ] **Step 7: Run vet + the suite once more.**
  Command: `go vet ./... && go test ./...`
  Expected: PASS — no vet diagnostics; all tests `ok`.

- [ ] **Step 8: Commit.**
```
git add pix.go pix_test.go
git commit -m "feat: add PIX EVP generator and registry self-registration"
```

---

### Task M4-4: Manual smoke check + fuzz round-trip guard

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/pix_test.go` (append fuzz test)

**Interfaces:**
- Consumes:
  - `func NewPIX() *PIX`, `func (p *PIX) Generate() string`, `func (p *PIX) Validate(value string) bool` (M4-2/M4-3)
  - `func DetectPIXKind(value string) (string, bool)` (M4-1)
- Produces:
  - `func FuzzPIXValidate(f *testing.F)` — arbitrary input never panics; generated keys
    always validate.

- [ ] **Step 1: Write the fuzz test.** Append to `pix_test.go`:
```go
func FuzzPIXValidate(f *testing.F) {
	seeds := []string{
		"52998224725",
		"39591842000010",
		"joao.silva@example.com.br",
		"+5511998765432",
		"123e4567-e89b-42d3-a456-426614174000",
		"",
		"not-a-key",
		"@@@",
		"+55",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	p := NewPIX()

	f.Fuzz(func(t *testing.T, value string) {
		// Must never panic on arbitrary input.
		_ = p.Validate(value)
		_, _ = DetectPIXKind(value)

		// A Generate-produced key must always validate (round-trip invariant).
		key := p.Generate()
		if !p.Validate(key) {
			t.Fatalf("generated key failed to validate: %q", key)
		}
	})
}
```

- [ ] **Step 2: Run the fuzz seed corpus (short, non-fuzzing pass).**
  Command: `go test -run FuzzPIXValidate ./...`
  Expected: PASS — seed corpus executes as a normal test; `ok  github.com/inovacc/brdoc`.

- [ ] **Step 3: Run a brief active fuzz pass.**
  Command: `go test -run xxx -fuzz FuzzPIXValidate -fuzztime 10s .`
  Expected: PASS — `elapsed: ...`, `PASS`; no new failing corpus entries written to
  `testdata/fuzz/`.

- [ ] **Step 4: Manual smoke check via `go run` (registry dispatch).** Confirm the registry
  now lists `pix` and round-trips a generated key. Command:
```
go run ./cmd/brdoc detect 52998224725
```
  Expected: detection output (CPF length value resolves; pix subcommand will be present once
  the M1 registry-driven CLI is wired — this step only confirms the binary builds with the
  new type registered, no panic).

- [ ] **Step 5: Run the full suite with coverage to confirm the type is exercised.**
  Command: `go test -cover ./...`
  Expected: PASS — `ok  github.com/inovacc/brdoc  coverage: >= 80.0% of statements`.

- [ ] **Step 6: Commit.**
```
git add pix_test.go
git commit -m "test: add PIX fuzz round-trip and no-panic guard"
```


---

## Milestone M5 — Hardening & Docs

Goal: lock the toolkit down with Go-native fuzz tests proving the `Generate→Validate` invariant and no-panic safety per check-digit type, runnable godoc `Example*` functions that render on pkg.go.dev, a refreshed README + docs listing all 12 kinds and the paemuri migration note, a `Taskfile.yml` with `test` (`-short`) / `test:full` / lint-gate / temp coverage targets, and a `docs/BACKLOG.md` recording the `ValidateDocument` deprecation (removal after 2026-07-18) plus the deferred v2 items (Inscrição Estadual, multi-state RG).

This milestone modifies no document algorithms — it only adds tests, examples, docs, and tooling. All concrete types, the registry, sentinels, and helpers it uses are already frozen in M0 and implemented in M0–M4.

---

### Task M5-1: Fuzz tests for every check-digit type (Generate→Validate invariant + no-panic)

Native Go 1.24 fuzz targets, one `_fuzz_test.go` file. Two assertions per type: (a) every `Generate()` output validates true (round-trip), and (b) arbitrary corpus input never panics `Validate`/`Format`. The expensive randomized round-trip loops are guarded with `testing.Short()`; the no-panic targets are cheap and always run.

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/fuzz_test.go`

**Interfaces:**
- Consumes (frozen M0 + implemented M0–M4):
  - `func NewCPF() *CPF`; `func (c *CPF) Validate(value string) bool`; `func (c *CPF) Generate() string`; `func (c *CPF) Format(value string) (string, error)`
  - `func NewCNPJ() *CNPJ`; `func (c *CNPJ) Validate(value string) bool`; `func (c *CNPJ) Generate() string`; `func (c *CNPJ) Format(value string) (string, error)`
  - `func NewCNH() *CNH`; `func NewPIS() *PIS`; `func NewRenavam() *Renavam`; `func NewVoterID() *VoterID`; `func NewCNS() *CNS` (each `Validate(string) bool`, `Generate() string`, `Format(string) (string,error)`)
  - `func Kinds() []Kind`; `func Get(kind Kind) (Document, bool)`; `type Document interface { Kind() Kind; Validate(value string) bool; Generate() string; Format(value string) (string, error) }`
- Produces: fuzz targets `FuzzCPFValidate`, `FuzzCNPJValidate`, `FuzzCNHValidate`, `FuzzPISValidate`, `FuzzRenavamValidate`, `FuzzVoterIDRoundTrip`, `FuzzCNSRoundTrip`, `FuzzRegistryNoPanic` (consumed by no later task; they are the test surface). The VoterID/CNS round-trip targets use the `*RoundTrip` suffix because their `*Validate` no-panic names are already taken by Task M2B-9 in the same `brdoc` package.

Steps:

- [ ] **Step 1: Write the no-panic registry fuzz target (failing — file does not compile yet).** Create `D:/weaver-sync/development/personal/projects/brdoc/fuzz_test.go` with the package header, imports, and the registry-wide no-panic target only:

```go
package brdoc

import (
	"testing"
)

// FuzzRegistryNoPanic feeds arbitrary bytes to every registered type's
// Validate and Format. Neither may panic on any input.
func FuzzRegistryNoPanic(f *testing.F) {
	seeds := []string{
		"",
		"0",
		"00000000000",
		"11111111111111",
		"abc-1234",
		"+5511999998888",
		"529.982.247-25",
		"39591842000010",
		"\x00\x01\x02",
		"  \t\n  ",
		"ＡＢＣ１２３４",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		for _, k := range Kinds() {
			doc, ok := Get(k)
			if !ok {
				t.Fatalf("registry returned no Document for kind %q", k)
			}
			// Must not panic; result value is irrelevant here.
			_ = doc.Validate(in)
			_, _ = doc.Format(in)
		}
	})
}
```

- [ ] **Step 2: Run it and observe the expected FAIL (compile error — round-trip targets referenced next don't exist yet, but this target alone should pass).** Command: `go test -run=FuzzRegistryNoPanic ./...`. Expected: PASS for this single target. If instead you see a build error, it means M0–M4 types are not yet present in the package — stop and resolve that first. Expected success line: `ok  	github.com/inovacc/brdoc`.

- [ ] **Step 3: Add the per-type round-trip fuzz targets (failing until run-guarded body compiles).** Append to `fuzz_test.go`. Each seeds with concrete valid samples and arbitrary junk, asserts no panic always, and (only when not `-short`) asserts the freshly generated value validates true. CPF/CNPJ/CNH/PIS/RENAVAM use `Fuzz<Type>Validate`; VoterID and CNS use `Fuzz<Type>RoundTrip` to avoid colliding with their existing `Fuzz<Type>Validate` no-panic targets from Task M2B-9 (same package):

```go
// roundTrip is the shared body for a single check-digit type: arbitrary input
// must never panic, and the type's own Generate output must always Validate.
func roundTrip(f *testing.F, gen func() string, val func(string) bool, seeds ...string) {
	base := []string{"", "0", "00000000000", "abc", "\x00\xff", "123456789012345"}
	for _, s := range append(base, seeds...) {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		_ = val(in) // must not panic on arbitrary input

		if testing.Short() {
			return // skip the randomized round-trip under -short
		}
		got := gen()
		if !val(got) {
			t.Fatalf("Generate produced an invalid value: %q", got)
		}
	})
}

func FuzzCPFValidate(f *testing.F) {
	c := NewCPF()
	roundTrip(f, c.Generate, c.Validate, "529.982.247-25", "52998224725", "11111111111")
}

func FuzzCNPJValidate(f *testing.F) {
	c := NewCNPJ()
	roundTrip(f, c.Generate, c.Validate, "11.444.777/0001-61", "11444777000161", "39591842000010")
}

func FuzzCNHValidate(f *testing.F) {
	c := NewCNH()
	roundTrip(f, c.Generate, c.Validate, "02650306461", "00000000000")
}

func FuzzPISValidate(f *testing.F) {
	p := NewPIS()
	roundTrip(f, p.Generate, p.Validate, "120.1234.567-8", "12012345678")
}

func FuzzRenavamValidate(f *testing.F) {
	r := NewRenavam()
	roundTrip(f, r.Generate, r.Validate, "16778534256", "00000000000")
}

// NOTE: VoterID and CNS already have no-panic fuzz targets named
// FuzzVoterIDValidate / FuzzCNSValidate in voterid_test.go / cns_test.go
// (Task M2B-9, same package brdoc). Re-declaring those names here would be a
// duplicate-symbol compile error. The round-trip targets below therefore use
// distinct *RoundTrip names so M5-1 adds the -short-guarded Generate→Validate
// invariant for those two types without colliding with M2B-9's targets.
func FuzzVoterIDRoundTrip(f *testing.F) {
	v := NewVoterID()
	roundTrip(f, v.Generate, v.Validate, "102385010671", "000000000000")
}

func FuzzCNSRoundTrip(f *testing.F) {
	c := NewCNS()
	roundTrip(f, c.Generate, c.Validate, "700509377540001", "100000000000000")
}
```

- [ ] **Step 4: Run the full fuzz-test set in unit mode (deterministic seed corpus only) — expect PASS.** Command: `go test -short -run=Fuzz ./...`. Expected output ends with `ok  	github.com/inovacc/brdoc`. The `-run` (not `-fuzz`) form executes only the seed corpus, so it finishes in well under a second; `-short` skips the randomized round-trip body, leaving only the no-panic assertions.

- [ ] **Step 5: Run one target in true fuzzing mode briefly to confirm the round-trip invariant holds — expect PASS, no crashers.** Command: `go test -run=^$ -fuzz=FuzzCNPJValidate -fuzztime=15s ./...`. Expected: a line like `fuzz: elapsed: ..., execs: ... (n/sec), new interesting: ...` followed by `PASS` and `ok`. No file should appear under `testdata/fuzz/` (a crasher would create one). If a crasher appears, the corresponding type has a Generate/Validate bug introduced in M0–M4 — fix it there, not here.

- [ ] **Step 6: Commit.** Commands:
```
git add fuzz_test.go
git commit -m "test: add Go-native fuzz targets for check-digit types (round-trip + no-panic)"
```

---

### Task M5-2: Godoc Example* functions per type (render on pkg.go.dev)

Runnable `Example*` functions live in a dedicated `example_test.go` so they compile and run under `go test` and render in package documentation. Each uses `// Output:` so the example is verified, not merely compiled. Values are deterministic samples (no `Generate()` output, which is random and would break `// Output:`).

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/example_test.go`

**Interfaces:**
- Consumes:
  - `func NewCPF() *CPF`; `func (c *CPF) Validate(string) bool`; `func (c *CPF) Format(string) (string, error)`; `func (c *CPF) Origin(value string) (string, error)`
  - `func NewCNPJ() *CNPJ`; `func (c *CNPJ) Validate(string) bool`; `func (c *CNPJ) Format(string) (string, error)`
  - `func Validate(kind Kind, value string) (bool, error)`; `func Format(kind Kind, value string) (string, error)`; `func Detect(value string) (Kind, bool)`; `func Kinds() []Kind`
  - Kind constants `KindCPF`, `KindCNPJ`; `func (k Kind) String() string`
  - `func NewPIS() *PIS`; `func (p *PIS) Format(string) (string, error)`; `func NewVoterID() *VoterID`; `func (v *VoterID) Origin(value string) (string, error)`
- Produces: example functions `ExampleNewCPF_Validate`, `ExampleNewCPF_Format`, `ExampleCPF_Origin`, `ExampleNewCNPJ_Validate`, `ExampleNewCNPJ_Format`, `ExampleNewPIS_Format`, `ExampleNewVoterID_Origin`, `ExampleValidate`, `ExampleFormat`, `ExampleDetect`, `ExampleKinds` (test-only, in `package brdoc_test`; no later task consumes them). `ExampleNewVoterID_Origin` uses the constructor form deliberately so it does not collide with the package-internal `ExampleVoterID_Origin` from Task M2B-10.

Steps:

- [ ] **Step 1: Write the CPF/CNPJ examples (failing until file compiles + outputs match).** Create `D:/weaver-sync/development/personal/projects/brdoc/example_test.go`:

```go
package brdoc_test

import (
	"fmt"

	"github.com/inovacc/brdoc"
)

func ExampleNewCPF_Validate() {
	fmt.Println(brdoc.NewCPF().Validate("529.982.247-25"))
	fmt.Println(brdoc.NewCPF().Validate("111.111.111-11"))
	// Output:
	// true
	// false
}

func ExampleNewCPF_Format() {
	out, _ := brdoc.NewCPF().Format("52998224725")
	fmt.Println(out)
	// Output: 529.982.247-25
}

func ExampleCPF_Origin() {
	origin, _ := brdoc.NewCPF().Origin("529.982.247-25")
	fmt.Println(origin)
	// Output: SP, MS, MT, GO, DF
}

func ExampleNewCNPJ_Validate() {
	fmt.Println(brdoc.NewCNPJ().Validate("11.444.777/0001-61"))
	fmt.Println(brdoc.NewCNPJ().Validate("39591842000010"))
	// Output:
	// true
	// true
}

func ExampleNewCNPJ_Format() {
	out, _ := brdoc.NewCNPJ().Format("11444777000161")
	fmt.Println(out)
	// Output: 11.444.777/0001-61
}
```

- [ ] **Step 2: Run the examples and observe expected behavior.** Command: `go test -run=Example ./...`. Expected: PASS, ending `ok  	github.com/inovacc/brdoc`. NOTE: the literal `// Output:` strings above (the CPF origin string `SP, MS, MT, GO, DF` and the `39591842000010` validity) must match what the M0/M2 implementations actually return. If an example FAILs with a diff like `got: ... want: ...`, update the `// Output:` line in this file to the implementation's real value — the implementation is the source of truth, the example documents it. Re-run until PASS.

- [ ] **Step 3: Add the registry-dispatch and PIS/VoterID examples.** Append to `example_test.go`:

```go
func ExampleNewPIS_Format() {
	out, _ := brdoc.NewPIS().Format("12012345678")
	fmt.Println(out)
	// Output: 120.1234.567-8
}

// Named ExampleNewVoterID_Origin (constructor form, matching ExampleNewPIS_Format)
// to avoid colliding with the package-internal ExampleVoterID_Origin added by
// Task M2B-10 in voterid_test.go (package brdoc). This file is package brdoc_test.
func ExampleNewVoterID_Origin() {
	origin, _ := brdoc.NewVoterID().Origin("102385010671")
	fmt.Println(origin)
	// Output: SP
}

func ExampleValidate() {
	ok, err := brdoc.Validate(brdoc.KindCPF, "529.982.247-25")
	fmt.Println(ok, err)
	// Output: true <nil>
}

func ExampleFormat() {
	out, _ := brdoc.Format(brdoc.KindCNPJ, "11444777000161")
	fmt.Println(out)
	// Output: 11.444.777/0001-61
}

func ExampleDetect() {
	kind, ok := brdoc.Detect("52998224725")
	fmt.Println(kind, ok)
	// Output: cpf true
}

func ExampleKinds() {
	for _, k := range brdoc.Kinds() {
		fmt.Println(k)
	}
	// Output:
	// cep
	// cnh
	// cnpj
	// cns
	// cpf
	// phone
	// pis
	// pix
	// plate
	// renavam
	// rg
	// voter_id
}
```

- [ ] **Step 4: Run all examples again — reconcile any `// Output:` mismatches.** Command: `go test -run=Example ./...`. The `ExampleKinds` block lists all 12 kinds in the sorted order `Kinds()` returns (alphabetical by Kind string). The `ExampleNewVoterID_Origin` output (`SP`) must match the M2 voter-ID UF-code→state mapping. The `ExampleNewPIS_Format` mask `120.1234.567-8` must match the M2 PIS `###.#####.##-#` mask. Adjust any literal that differs from the real implementation output, then re-run until `ok  	github.com/inovacc/brdoc`.

- [ ] **Step 5: Verify the examples render in package docs.** Command: `go doc -all . | grep -i example`. Expected: lines listing the exported `Example*` functions (go doc surfaces verified examples). This is a smoke check; no assertion failure is expected.

- [ ] **Step 6: Commit.** Commands:
```
git add example_test.go
git commit -m "docs: add runnable godoc Example functions for all document types"
```

---

### Task M5-3: README + docs refresh (all types + paemuri migration note)

Rewrite the supported-types table in `README.md` to list all 12 kinds (CPF, CNPJ, CNH, PIS, RENAVAM, Voter ID, CEP, Phone, Plate, CNS, RG, PIX) with their capabilities, document the registry dispatch + `detect` + `mcp` surfaces, and add a "Migrating from paemuri/brdoc" section showing the one-line `compat/` import swap. Add a short `docs/ARCHITECTURE.md` capturing the interface+registry design so the README stays focused.

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/README.md` (replace the features/supported-types section and add the migration section)
- Create: `D:/weaver-sync/development/personal/projects/brdoc/docs/ARCHITECTURE.md`

**Interfaces:**
- Consumes (documentation references only): `func Kinds() []Kind`; `func Detect(value string) (Kind, bool)`; `func Validate(kind Kind, value string) (bool, error)`; the `compat` package signatures `IsCPF`, `IsCNPJ`, `IsCEP`, `IsRG`, etc.; CLI subcommands `brdoc <kind>`, `brdoc detect`, `brdoc mcp`, `brdoc version`.
- Produces: no Go symbols (docs only).

Steps:

- [ ] **Step 1: Open the README and locate the current feature/supported-types section.** Command: `go run ./cmd/brdoc --help` (capture the live subcommand list so the README matches reality). Then read `README.md` and identify the CPF/CNPJ-only feature list and any "Supported documents" table to be replaced. No commit yet.

- [ ] **Step 2: Replace the supported-types table.** In `README.md`, substitute the old (CPF/CNPJ-only) capability table with this full matrix (the checkmarks mirror §4 of the design spec):

```markdown
## Supported documents

| Kind (`brdoc <kind>`) | Validate | Generate | Format (mask)        | Origin (UF/region) |
|-----------------------|:--------:|:--------:|----------------------|:------------------:|
| `cpf`                 | yes      | yes      | `###.###.###-##`     | yes (9th digit)    |
| `cnpj`                | yes      | yes      | `##.###.###/####-##` | —                  |
| `cnh`                 | yes      | yes      | identity (11 digits) | —                  |
| `pis`                 | yes      | yes      | `###.#####.##-#`     | —                  |
| `renavam`             | yes      | yes      | identity (11 digits) | —                  |
| `voter_id`            | yes      | yes      | spaced groups        | yes (UF code)      |
| `cep`                 | yes      | yes      | `#####-###`          | yes (prefix range) |
| `phone`               | yes      | yes      | `(##) #####-####`    | yes (DDD)          |
| `plate`               | yes      | yes      | dash insert/strip    | —                  |
| `cns`                 | yes      | yes      | identity (15 digits) | —                  |
| `rg`                  | yes (SP/RJ, `--uf`) | yes | `##.###.###-#`   | —                  |
| `pix`                 | yes      | yes (EVP)| identity             | —                  |

All subcommands accept `-g/--generate`, `-v/--validate VALUE`, `--format VALUE`,
`-f/--from FILE|-` (bulk file/stdin), and `-n/--count N`. `--origin VALUE` is
available on origin-aware kinds; `--uf SP` applies only to `rg`.
```

- [ ] **Step 3: Document the registry + top-level commands.** Add this section to `README.md` directly below the table:

```markdown
## Library API

```go
import "github.com/inovacc/brdoc"

ok, err := brdoc.Validate(brdoc.KindCPF, "529.982.247-25") // registry dispatch
val, err := brdoc.Generate(brdoc.KindCNPJ)                 // random valid value
out, err := brdoc.Format(brdoc.KindCEP, "01001000")        // "01001-000"
kind, ok := brdoc.Detect("52998224725")                    // -> brdoc.KindCPF
kinds   := brdoc.Kinds()                                    // all registered kinds, sorted
```

Ergonomic concrete types remain available: `brdoc.NewCPF().Validate(s)`,
`brdoc.NewCNPJ().Generate()`, etc. Every type self-registers via `init()`, so the
CLI and MCP server derive their surfaces from the registry with no per-type drift.

## Command line

```
brdoc <kind> [-g] [-v VALUE] [--format VALUE] [--origin VALUE] [-f FILE|-] [-n N] [--uf SP]
brdoc detect <value>     # auto-detect the kind
brdoc mcp                # start the MCP server over stdio
brdoc version            # print version
```
```

- [ ] **Step 4: Add the paemuri migration note.** Append this section to `README.md`:

```markdown
## Migrating from paemuri/brdoc

`brdoc` is a strict superset of `paemuri/brdoc` (validation-only). The `compat`
subpackage mirrors paemuri's `Is*` signatures exactly, so migration is a one-line
import swap:

```go
// before
import "github.com/paemuri/brdoc/v3"

// after
import "github.com/inovacc/brdoc/compat"
```

Call sites are unchanged — `compat.IsCPF`, `compat.IsCNPJ`, `compat.IsCEP`,
`compat.IsPhone`, `compat.IsRG`, `compat.IsPlate`, `compat.IsCNS`,
`compat.IsVoterID`, `compat.IsRENAVAM`, `compat.IsCNH`, `compat.IsPIS` keep
paemuri's exact parameter and return shapes (including `(bool, UF)` for CEP/phone
and `(bool, error)` for RG). Once migrated, you also gain `Generate`, `Format`,
`Origin`, the registry dispatch, the CLI, and the MCP server from the root package.
```

- [ ] **Step 5: Create the architecture doc.** Create `D:/weaver-sync/development/personal/projects/brdoc/docs/ARCHITECTURE.md`:

```markdown
# Architecture

`brdoc` is one core library (root package `brdoc`) plus three thin adapters:
a Cobra CLI (`cmd/brdoc`), an MCP server (`mcp/`, `brdoc mcp`), and a
paemuri-compatible drop-in (`compat/`).

## Interface + registry (hybrid)

Every document type implements the `Document` interface and self-registers in
`init()`:

```go
type Document interface {
	Kind() Kind
	Validate(value string) bool
	Generate() string
	Format(value string) (string, error)
}
```

Optional capabilities are discovered by type assertion, never bloating the base
interface:

- `OriginResolver` — `Origin(value string) (string, error)` (CPF region, CEP/phone/voter UF).
- `UFScoped` — `ValidateUF(value string, uf UF) (bool, error)` + `ImplementedUFs() []UF` (RG).

The registry (`Register`, `Get`, `Kinds`, `Validate`, `Generate`, `Format`,
`Detect`) is the single source of truth. Adding a type to the registry
automatically lights up its CLI subcommand and MCP enum value — no boilerplate
duplication per type.

## Concurrency

Generation uses `math/rand/v2` top-level functions (goroutine-safe), so the
registered singletons serve concurrent `Generate()` calls without mutexes.

## Errors

Sentinel errors (`ErrInvalidLength`, `ErrInvalidFormat`, `ErrUnknownKind`,
`ErrUnsupported`, `ErrUFNotImplemented`) are wrapped with `%w` and compared with
`errors.Is`/`errors.As`.

## Dependency direction

Adapters import the root package; the root package imports no adapter. Brand
strings live only in `meta.go`, so a future rename is mechanical.
```

- [ ] **Step 6: Verify docs are internally consistent with the live CLI.** Command: `go run ./cmd/brdoc --help`. Confirm every `brdoc <kind>` row in the README table corresponds to a real subcommand in the output and that `detect`, `mcp`, and `version` are present. Fix any drift in the README. (Documentation-only; no Go test runs.)

- [ ] **Step 7: Commit.** Commands:
```
git add README.md docs/ARCHITECTURE.md
git commit -m "docs: refresh README with all 12 types, registry API, and paemuri migration note"
```

---

### Task M5-4: Taskfile.yml targets (`test` -short, `test:full`, lint gate, temp coverage)

Rework `Taskfile.yml` so the default `test` target is fast (`-short`, lint-gated) and a separate `test:full` runs the entire suite including fuzz seed corpora and benchmarks. Add a `lint` target and a `coverage` target that writes the profile to the system temp dir with a datetime suffix (per global standards), printing the total coverage.

**Files:**
- Modify: `D:/weaver-sync/development/personal/projects/brdoc/Taskfile.yml` (replace the `test` task; add `lint`, `test:full`, `coverage`)

**Interfaces:**
- Consumes: the `-short`-guarded fuzz round-trips from Task M5-1 (so `test` stays fast); `golangci-lint` config `.golangci.yml` (already present).
- Produces: task names `lint`, `test`, `test:full`, `coverage` (invoked by humans/CI; no Go symbols).

Steps:

- [ ] **Step 1: Replace the Taskfile contents.** Overwrite `D:/weaver-sync/development/personal/projects/brdoc/Taskfile.yml` with:

```yaml
# https://taskfile.dev

version: '3'

tasks:
  lint:
    desc: Format and lint (gate for test / CI)
    cmds:
      - golangci-lint fmt
      - golangci-lint run --fix ./... --timeout=5m

  test:
    desc: Fast unit tests (-short) behind the lint gate
    deps: [lint]
    cmds:
      - go test -short -race -p=1 ./...

  test:full:
    desc: Full suite — all tests, fuzz seed corpora, and benchmarks
    deps: [lint]
    cmds:
      - go test -race -p=1 ./...
      - go test -run=Fuzz -race ./...
      - go test -race -bench=. -benchmem ./...

  coverage:
    desc: Coverage profile written to the system temp dir with a datetime suffix
    cmds:
      - 'go test -short -covermode=atomic -coverprofile="{{.OUT}}" ./...'
      - 'go tool cover -func="{{.OUT}}" | tail -n 1'
      - 'echo "coverage profile: {{.OUT}}"'
    vars:
      OUT: '{{.TMPDIR | default (env "TMPDIR") | default (env "TEMP") | default "/tmp"}}/brdoc_coverage_{{now | date "20060102_150405"}}.out'

  upgrade:
    cmds:
      - go get -u ./...
      - go mod tidy -v

  build-dev:
    cmds:
      - goreleaser build --snapshot --clean

  build-prod:
    cmds:
      - goreleaser --snapshot --skip-publish --rm-dist
```

- [ ] **Step 2: Run the fast test target and observe expected PASS.** Command: `task test`. Expected: `golangci-lint fmt` and `golangci-lint run --fix` complete with no errors, then `go test -short ...` ends with `ok  	github.com/inovacc/brdoc`. The `-short` flag skips the randomized fuzz round-trips from M5-1, keeping this under a few seconds. If `task` is unavailable in the environment, run the equivalent directly: `go test -short -race -p=1 ./...` and expect the same `ok` line.

- [ ] **Step 3: Run the coverage target and confirm a temp file + total are emitted.** Command: `task coverage`. Expected: a final `total:	(statements)	NN.N%` line (target >=80%, project sits near ~95%) and a `coverage profile: .../brdoc_coverage_YYYYMMDD_HHMMSS.out` line pointing at the system temp dir. If `task` is unavailable, verify the underlying command works: `go test -short -covermode=atomic -coverprofile="$TEMP/brdoc_cov.out" ./... && go tool cover -func="$TEMP/brdoc_cov.out" | tail -n 1`.

- [ ] **Step 4: Run the full suite once to confirm it is green.** Command: `task test:full`. Expected: the unit run, the `-run=Fuzz` seed-corpus run, and the benchmark run all finish with `ok  	github.com/inovacc/brdoc` (benchmarks also print `Benchmark...` lines). This is the gate CI will use.

- [ ] **Step 5: Commit.** Commands:
```
git add Taskfile.yml
git commit -m "chore: split Taskfile into lint/test(-short)/test:full/coverage targets"
```

---

### Task M5-5: docs/BACKLOG.md — ValidateDocument DEPRECATION + v2 items

Create the project backlog recording the `ValidateDocument` deprecation (introduced in M0, removal after 2026-07-18, per the global deprecation policy) and the two deferred v2 differentiators from the spec/gap analysis: Inscrição Estadual (27 per-UF algorithms) and multi-state RG (beyond SP/RJ). Include the deprecation tag, removal date, and the migration path so the eventual cleanup commit is unambiguous.

**Files:**
- Create: `D:/weaver-sync/development/personal/projects/brdoc/docs/BACKLOG.md`

**Interfaces:**
- Consumes (documentation references only): `func ValidateDocument(doc string) (docType string, isValid bool)` (the deprecated M0-8 wrapper); `func Detect(value string) (Kind, bool)`; `func Validate(kind Kind, value string) (bool, error)`; the `UFScoped` interface; `ErrUFNotImplemented`.
- Produces: no Go symbols (docs only).

Steps:

- [ ] **Step 1: Create the backlog file.** Create `D:/weaver-sync/development/personal/projects/brdoc/docs/BACKLOG.md`:

```markdown
# Backlog

Future work and tech debt. Items are grouped by tag.

## DEPRECATION

### `ValidateDocument(doc string) (docType string, isValid bool)` — removal after 2026-07-18

- **Introduced:** M0 (2026-06-18) as a thin, deprecated wrapper preserving the
  legacy `"CPF"` / `"CNPJ"` / `"UNKNOWN"` labels.
- **Reason:** superseded by the registry-based API. Use `Detect(value) (Kind, bool)`
  to identify a document and `Validate(kind, value) (bool, error)` to validate it.
- **Doc comment in code:**
  `// Deprecated: Use Detect + Validate instead. Will be removed after 2026-07-18.`
- **Migration path:**

  ```go
  // before
  docType, ok := brdoc.ValidateDocument("52998224725") // "CPF", true

  // after
  kind, _ := brdoc.Detect("52998224725") // brdoc.KindCPF
  ok, _   := brdoc.Validate(kind, "52998224725")
  ```

- **Logging:** each call logs a `slog` warning naming the removal date so usage is
  visible in consumers.
- **Cleanup:** after 2026-07-18, remove the wrapper (and its warning) in a
  dedicated `chore: remove deprecated ValidateDocument` commit — not mixed with
  features. Drop this backlog entry in the same commit.

## v2 (post-rebrand, scoped separately)

### Inscrição Estadual (27 per-UF algorithms)

- **What:** state tax-registration validator; 27 distinct per-UF algorithms
  (length and check-digit rules vary by state). Upstream `paemuri/brdoc` issue #7
  has been open since inception and never shipped by anyone — this is the single
  biggest open gap in the Go ecosystem and a clear differentiator.
- **Shape:** new file `ie.go`, divergent signature
  `Validate(value string, uf UF) (bool, error)` implementing the frozen `UFScoped`
  interface; CLI `brdoc ie --uf XX`; return `ErrUFNotImplemented` for UFs not yet
  covered.
- **Plan:** land incrementally — SP, RJ, MG, RS, PR first; expand UF-by-UF.

### RG multi-state (beyond SP/RJ)

- **What:** extend `rg.go` past the v1 SP/RJ algorithms. Upstream issue #22 ships
  only SP/RJ; add more UFs where check-digit rules are documented, returning
  `ErrUFNotImplemented` elsewhere.
- **Shape:** extends the existing `RG` `UFScoped` implementation; `ImplementedUFs()`
  grows as states are added. No signature change.
```

- [ ] **Step 2: Verify the deprecation date is consistent with the code comment.** Command: `grep -rn "2026-07-18" .`. Expected: matches in both `docs/BACKLOG.md` (this task) and the M0-8 `ValidateDocument` doc comment in `brdoc.go`. If the dates differ, align `docs/BACKLOG.md` to the date actually written in code (the code comment is authoritative for the removal date). Documentation-only; no Go test runs.

- [ ] **Step 3: Verify the documented migration snippet compiles in spirit (Detect + Validate exist).** Command: `go doc . Detect && go doc . Validate && go doc . ValidateDocument`. Expected: `go doc` prints the signatures of all three (confirming the backlog's "before/after" snippet refers to real exported functions, and that the deprecated wrapper still exists pre-2026-07-18). No assertion failure expected.

- [ ] **Step 4: Commit.** Commands:
```
git add docs/BACKLOG.md
git commit -m "docs: add BACKLOG with ValidateDocument deprecation and v2 items (IE, multi-state RG)"
```
