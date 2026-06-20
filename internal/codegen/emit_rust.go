package codegen

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/inovacc/selo"
)

// emit_rust.go is the Rust language emitter (the 8th target; mirrors the PHP and
// Python references). For one kind it renders an idiomatic Rust module
// (src/<kind>.rs exposing free validate/format/origin/generate functions and,
// for RG/IE, UF-param variants) with an inline #[cfg(test)] block driven by the
// golden vector, the vector JSON itself, and the shared scaffolding (mod-11
// reducer, embedded data tables, lib.rs re-exports, Cargo.toml). The check-digit
// kinds reuse a single shared mod11 reducer parameterized by each kind's
// CheckDigit spec.
//
// Every Rust algorithm is translated VERBATIM (logic) from the proven PHP/Python
// emitter — only the syntax differs — so the port is correct by construction and
// the Rust vector tests (run on CI via `cargo test`) pass. The Rust SOURCE is
// deterministic and snapshot-stable; only the vector JSON is non-deterministic
// (it mixes selo.Generate output), so the golden snapshot compares the
// deterministic source files and re-validates the vectors against selo.

//go:embed templates/rust/mod11.rs.tmpl templates/rust/Cargo.toml.tmpl
var rustTemplates embed.FS

func init() { Register(rustEmitter{}) }

// rustEmitter implements Emitter for Rust.
type rustEmitter struct{}

// Lang reports the target language.
func (rustEmitter) Lang() Lang { return LangRust }

// Emit renders the full Rust file set for kind: the per-kind module (with its
// inline test), the vector JSON, and the shared scaffolding (idempotent across
// kinds).
func (e rustEmitter) Emit(kind selo.Kind, plan KindPlan, vec Vector) ([]File, error) {
	module, err := e.renderModule(kind, plan)
	if err != nil {
		return nil, err
	}

	vectorJSON, err := json.MarshalIndent(vec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("codegen/rust: marshal %q vector: %w", kind, err)
	}

	vectorJSON = append(vectorJSON, '\n')

	shared, err := e.sharedFiles()
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, 2+len(shared))
	files = append(files,
		File{Path: "src/" + rustName(kind) + ".rs", Content: []byte(module)},
		File{Path: "vectors/" + kind.String() + ".json", Content: vectorJSON},
	)
	files = append(files, shared...)

	return files, nil
}

// sharedFiles returns the language-wide files that do not depend on a single
// kind: the mod-11 reducer, the embedded data tables, the lib.rs re-exports, and
// the Cargo manifest. They are emitted on every per-kind call and are
// byte-identical, so re-writing them is idempotent.
func (e rustEmitter) sharedFiles() ([]File, error) {
	mod11, err := rustTemplates.ReadFile("templates/rust/mod11.rs.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/rust: read mod11 template: %w", err)
	}

	cargo, err := rustTemplates.ReadFile("templates/rust/Cargo.toml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/rust: read Cargo template: %w", err)
	}

	return []File{
		{Path: "src/mod11.rs", Content: mod11},
		{Path: "src/data.rs", Content: []byte(e.renderData())},
		{Path: "src/lib.rs", Content: []byte(e.renderIndex())},
		{Path: "Cargo.toml", Content: cargo},
	}, nil
}

// renderData emits src/data.rs: the embedded data tables (CEP ranges, DDD->UF,
// CPF region map, voter UF names) as pub static const slices, from the codegen
// accessors. Sort order matches the PHP/Python data renderers for parity.
func (e rustEmitter) renderData() string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("//! Embedded data tables (CEP ranges, DDD->UF, CPF region, voter UF names).\n\n")
	b.WriteString("pub struct UfRange {\n")
	b.WriteString("    pub uf: &'static str,\n")
	b.WriteString("    pub from: i32,\n")
	b.WriteString("    pub to: i32,\n")
	b.WriteString("}\n\n")

	// CEP prefix ranges (scan order; first match wins).
	b.WriteString("/// CEP prefix ranges in scan order; first match wins.\n")
	b.WriteString("pub static CEP_RANGES: &[UfRange] = &[\n")

	for _, r := range CEPRanges() {
		fmt.Fprintf(&b, "    UfRange { uf: %s, from: %d, to: %d },\n", rustQuote(r.UF), r.From, r.To)
	}

	b.WriteString("];\n\n")

	// DDD -> UF, sorted by DDD string.
	b.WriteString("/// DDD area-code -> UF, sorted by DDD string.\n")
	b.WriteString("pub static DDD_TO_UF: &[(&str, &str)] = &[\n")

	dddMap := DDDtoUF()

	ddds := make([]string, 0, len(dddMap))
	for d := range dddMap {
		ddds = append(ddds, d)
	}

	sort.Strings(ddds)

	for _, d := range ddds {
		fmt.Fprintf(&b, "    (%s, %s),\n", rustQuote(d), rustQuote(dddMap[d].String()))
	}

	b.WriteString("];\n\n")

	// CPF ninth-digit region map (digit 0..9 -> region).
	b.WriteString("/// CPF ninth-digit (0..9) -> region.\n")
	b.WriteString("pub static CPF_REGIONS: &[(i32, &str)] = &[\n")

	cpfRegions := CPFRegions()
	for i := 0; i <= 9; i++ {
		if name, ok := cpfRegions[i]; ok {
			fmt.Fprintf(&b, "    (%d, %s),\n", i, rustQuote(name))
		}
	}

	b.WriteString("];\n\n")

	// Voter-ID UF code -> region name, ascending.
	b.WriteString("/// Voter-ID UF code (1..28) -> region name.\n")
	b.WriteString("pub static VOTER_UF_NAMES: &[(i32, &str)] = &[\n")

	voterNames := VoterUFNames()

	codes := make([]int, 0, len(voterNames))
	for c := range voterNames {
		codes = append(codes, c)
	}

	sort.Ints(codes)

	for _, c := range codes {
		fmt.Fprintf(&b, "    (%d, %s),\n", c, rustQuote(voterNames[c]))
	}

	b.WriteString("];\n")

	return b.String()
}

// renderIndex emits src/lib.rs: the per-kind `pub mod` declarations plus
// `pub use` re-exports, in KindStrings() sorted order so the output is
// deterministic.
func (e rustEmitter) renderIndex() string {
	var b strings.Builder
	b.WriteString(rustFileHeader())

	// RG and IE both export a `UFS` constant; the per-kind glob re-exports below
	// therefore re-export the name from two modules. That is harmless (callers
	// use the fully-qualified rg::UFS / ie::UFS), so silence the lint.
	b.WriteString("#![allow(ambiguous_glob_reexports)]\n\n")

	b.WriteString("pub mod data;\n")
	b.WriteString("pub mod mod11;\n")

	kinds := KindStrings()
	for _, k := range kinds {
		fmt.Fprintf(&b, "pub mod %s;\n", k)
	}

	b.WriteString("\n")
	b.WriteString("pub use mod11::SeloError;\n")

	for _, k := range kinds {
		fmt.Fprintf(&b, "pub use %s::*;\n", k)
	}

	return b.String()
}

// renderModule dispatches to the per-group renderer for kind, then appends the
// inline #[cfg(test)] block.
func (e rustEmitter) renderModule(kind selo.Kind, plan KindPlan) (string, error) {
	var body string

	switch kind {
	case selo.KindCPF:
		body = e.renderCPF(plan)
	case selo.KindPIS:
		body = e.renderSimpleNumeric(plan, "pis", 11)
	case selo.KindRenavam:
		body = e.renderRenavam(plan)
	case selo.KindCNH:
		body = e.renderCNH()
	case selo.KindRG:
		body = e.renderRG(plan)
	case selo.KindIE:
		body = e.renderIE(plan)
	case selo.KindCNS:
		body = e.renderCNS(plan)
	case selo.KindCNPJ:
		body = e.renderCNPJ(plan)
	case selo.KindPlate:
		body = e.renderPlate()
	case selo.KindPIX:
		body = e.renderPIX()
	case selo.KindCEP:
		body = e.renderCEP()
	case selo.KindPhone:
		body = e.renderPhone()
	case selo.KindVoterID:
		body = e.renderVoterID(plan)
	default:
		return "", fmt.Errorf("codegen/rust: no module renderer for kind %q", kind)
	}

	return body + "\n" + e.renderTest(kind), nil
}

// --- shared rendering helpers -----------------------------------------------

// rustFileHeader is the generated-file banner that every emitted Rust source
// begins with.
func rustFileHeader() string {
	return "// Code generated by selo gen --lang rust. DO NOT EDIT.\n\n"
}

// rustName returns the snake_case identity name for a kind (selo kinds are
// already snake_case, which is Rust's module + function convention). Kept for
// parity with phpClassName/pythonName.
func rustName(kind selo.Kind) string {
	return kind.String()
}

// rustQuote renders s as a Rust double-quoted string literal (escaping backslash
// and double quote, which is all a normal Rust string literal interprets).
func rustQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return "\"" + r.Replace(s) + "\""
}

// rustIntList renders a Go int slice as a Rust slice literal: "&[1, 2, 3]".
func rustIntList(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}

	return "&[" + strings.Join(parts, ", ") + "]"
}

// rustCheckDigitLiteral renders a CheckDigit as a Rust CheckDigit const struct
// literal. Because the struct has fixed fields, ALL fields are always rendered
// (unlike the PHP/Python literals that omit absent keys); defaults are &[],
// false, 0.
func rustCheckDigitLiteral(cd CheckDigit) string {
	rule := rustRule(cd.Rule)

	remainder := "&[]"
	if len(cd.RemainderTo0) > 0 {
		remainder = rustIntList(cd.RemainderTo0)
	}

	multiply := "false"
	if cd.MultiplyBy10 {
		multiply = "true"
	}

	return fmt.Sprintf(
		"CheckDigit { weights: %s, rule: %s, remainder_to0: %s, multiply_by10: %s, encode_x_at: %d, encode_zero_at: %d }",
		rustIntList(cd.Weights), rule, remainder, multiply, cd.EncodeXAt, cd.EncodeZeroAt,
	)
}

// rustRule maps a DVRule to its Rust Rule enum variant expression.
func rustRule(r DVRule) string {
	switch r {
	case DVElevenMinus:
		return "Rule::ElevenMinus"
	case DVModRemainder:
		return "Rule::ModRemainder"
	case DVRightmostDigit:
		return "Rule::RightmostDigit"
	case DVSumZero:
		return "Rule::SumZero"
	default:
		return "Rule::ModRemainder"
	}
}

// rustThrow maps a sentinel name to a Rust `return Err(...)` statement using the
// SeloError enum (the direct analogue of phpThrow).
func rustThrow(sentinel string) string {
	switch sentinel {
	case "ErrInvalidLength":
		return "return Err(SeloError::InvalidLength);"
	case "ErrInvalidFormat":
		return "return Err(SeloError::InvalidFormat);"
	default:
		return "return Err(SeloError::InvalidFormat);"
	}
}

// rustMaskExpr converts a '#'/'X'-placeholder mask (e.g. "###.#####.##-#") into a
// Rust format! expression slicing the cleaned digit variable named by v (e.g. d).
// Cleaned values are ASCII digits, so byte-slicing &d[a..b] is safe. Returns the
// full format! call.
func rustMaskExpr(mask, v string) string {
	var (
		fmtParts []string
		args     []string
	)

	pos := 0

	i := 0
	for i < len(mask) {
		c := mask[i]
		if c == '#' || c == 'X' {
			start := pos

			for i < len(mask) && (mask[i] == '#' || mask[i] == 'X') {
				i++
				pos++
			}

			fmtParts = append(fmtParts, "{}")
			args = append(args, fmt.Sprintf("&%s[%d..%d]", v, start, pos))

			continue
		}
		// literal separator
		fmtParts = append(fmtParts, rustEscapeFormatLiteral(string(c)))

		i++
	}

	return "format!(\"" + strings.Join(fmtParts, "") + "\", " + strings.Join(args, ", ") + ")"
}

// rustEscapeFormatLiteral escapes a literal separator character for use inside a
// format! string (escaping backslash, double quote, and the format braces).
func rustEscapeFormatLiteral(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "{", "{{", "}", "}}")
	return r.Replace(s)
}

// rustStringList renders a string slice as a Rust slice literal of double-quoted
// strings: &["SP", "RJ"].
func rustStringList(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = rustQuote(s)
	}

	return "&[" + strings.Join(quoted, ", ") + "]"
}

// rustHasOrigin reports whether kind has an origin resolver in the generated Rust
// module (mirrors phpHasOrigin).
func rustHasOrigin(kind selo.Kind) bool {
	switch kind { //nolint:exhaustive // only origin-capable kinds return true; all others fall through
	case selo.KindCPF, selo.KindCEP, selo.KindPhone, selo.KindVoterID:
		return true
	default:
		return false
	}
}
