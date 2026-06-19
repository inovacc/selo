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

// emit_ruby.go is the Ruby language emitter (plan M4; mirrors the M2 TypeScript
// reference). For one kind it renders an idiomatic Ruby module
// (Selo::<Kind>.valid?/.format/.origin and, for RG/IE, UF-param variants), a
// Minitest test driven by the golden vector, the vector JSON itself, and the
// shared scaffolding (mod-11 reducer, embedded data tables, Gemfile, Rakefile).
// The check-digit kinds reuse a single shared lib/selo/mod11.rb reducer
// parameterized by each kind's CheckDigit spec.
//
// Every Ruby algorithm is translated VERBATIM (logic) from the proven TS
// emitter — only the syntax differs — so the port is correct by construction and
// the Ruby vector tests (run on CI) pass. The Ruby SOURCE is deterministic and
// snapshot-stable; only the vector JSON is non-deterministic (it mixes
// selo.Generate output), so the golden snapshot compares the deterministic
// source files and re-validates the vectors against selo.

//go:embed templates/ruby/mod11.rb.tmpl templates/ruby/gemfile.tmpl templates/ruby/rakefile.tmpl
var rubyTemplates embed.FS

func init() { Register(rubyEmitter{}) }

// rubyEmitter implements Emitter for Ruby.
type rubyEmitter struct{}

// Lang reports the target language.
func (rubyEmitter) Lang() Lang { return LangRuby }

// Emit renders the full Ruby file set for kind: the per-kind module, its test,
// the vector JSON, and the shared scaffolding (idempotent across kinds).
func (e rubyEmitter) Emit(kind selo.Kind, plan KindPlan, vec Vector) ([]File, error) {
	module, err := e.renderModule(kind, plan)
	if err != nil {
		return nil, err
	}

	test := e.renderTest(kind)

	vectorJSON, err := json.MarshalIndent(vec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("codegen/ruby: marshal %q vector: %w", kind, err)
	}

	vectorJSON = append(vectorJSON, '\n')

	shared, err := e.sharedFiles()
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, 3+len(shared))
	files = append(files,
		File{Path: "lib/selo/" + kind.String() + ".rb", Content: []byte(module)},
		File{Path: "test/" + kind.String() + "_test.rb", Content: []byte(test)},
		File{Path: "vectors/" + kind.String() + ".json", Content: vectorJSON},
	)
	files = append(files, shared...)

	return files, nil
}

// sharedFiles returns the language-wide files that do not depend on a single
// kind: the mod-11 reducer, the embedded data tables, and the build/test
// scaffolding. They are emitted on every per-kind call and are byte-identical,
// so re-writing them is idempotent.
func (e rubyEmitter) sharedFiles() ([]File, error) {
	mod11, err := rubyTemplates.ReadFile("templates/ruby/mod11.rb.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/ruby: read mod11 template: %w", err)
	}

	gemfile, err := rubyTemplates.ReadFile("templates/ruby/gemfile.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/ruby: read gemfile template: %w", err)
	}

	rakefile, err := rubyTemplates.ReadFile("templates/ruby/rakefile.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/ruby: read rakefile template: %w", err)
	}

	return []File{
		{Path: "lib/selo/mod11.rb", Content: mod11},
		{Path: "lib/selo/data.rb", Content: []byte(e.renderData())},
		{Path: "Gemfile", Content: gemfile},
		{Path: "Rakefile", Content: rakefile},
	}, nil
}

// renderData emits the embedded data tables (CEP ranges, DDD->UF, CPF region
// map, voter UF names) as Ruby constants under Selo::Data, from the codegen
// accessors. Sort order matches the TS data renderer for snapshot parity.
func (e rubyEmitter) renderData() string {
	var b strings.Builder
	b.WriteString(rubyHeaderComment())
	b.WriteString("\n")
	b.WriteString("module Selo\n")
	b.WriteString("  module Data\n")

	// CEP prefix ranges (scan order; first match wins).
	b.WriteString("    CEP_RANGES = [\n")

	for _, r := range CEPRanges() {
		fmt.Fprintf(&b, "      { uf: %q, from: %d, to: %d },\n", r.UF, r.From, r.To)
	}

	b.WriteString("    ].freeze\n\n")

	// DDD -> UF.
	b.WriteString("    DDD_TO_UF = {\n")

	dddMap := DDDtoUF()

	ddds := make([]string, 0, len(dddMap))
	for d := range dddMap {
		ddds = append(ddds, d)
	}

	sort.Strings(ddds)

	for _, d := range ddds {
		fmt.Fprintf(&b, "      %q => %q,\n", d, dddMap[d].String())
	}

	b.WriteString("    }.freeze\n\n")

	// CPF ninth-digit region map.
	b.WriteString("    CPF_REGIONS = {\n")

	cpfRegions := CPFRegions()
	for i := 0; i <= 9; i++ {
		if name, ok := cpfRegions[i]; ok {
			fmt.Fprintf(&b, "      %d => %q,\n", i, name)
		}
	}

	b.WriteString("    }.freeze\n\n")

	// Voter-ID UF code -> region name.
	b.WriteString("    VOTER_UF_NAMES = {\n")

	voterNames := VoterUFNames()

	codes := make([]int, 0, len(voterNames))
	for c := range voterNames {
		codes = append(codes, c)
	}

	sort.Ints(codes)

	for _, c := range codes {
		fmt.Fprintf(&b, "      %d => %q,\n", c, voterNames[c])
	}

	b.WriteString("    }.freeze\n")
	b.WriteString("  end\n")
	b.WriteString("end\n")

	return b.String()
}

// renderModule dispatches to the per-group renderer for kind.
func (e rubyEmitter) renderModule(kind selo.Kind, plan KindPlan) (string, error) {
	switch kind {
	case selo.KindCPF:
		return e.renderCPF(plan), nil
	case selo.KindPIS:
		return e.renderSimpleNumeric(plan, "PIS", 11), nil
	case selo.KindRenavam:
		return e.renderRenavam(plan), nil
	case selo.KindCNH:
		return e.renderCNH(), nil
	case selo.KindRG:
		return e.renderRG(plan), nil
	case selo.KindIE:
		return e.renderIE(plan), nil
	case selo.KindCNS:
		return e.renderCNS(plan), nil
	case selo.KindCNPJ:
		return e.renderCNPJ(plan), nil
	case selo.KindPlate:
		return e.renderPlate(), nil
	case selo.KindPIX:
		return e.renderPIX(), nil
	case selo.KindCEP:
		return e.renderCEP(), nil
	case selo.KindPhone:
		return e.renderPhone(), nil
	case selo.KindVoterID:
		return e.renderVoterID(plan), nil
	default:
		return "", fmt.Errorf("codegen/ruby: no module renderer for kind %q", kind)
	}
}

// --- shared rendering helpers -----------------------------------------------

// rubyHeaderComment is the generated-file banner.
func rubyHeaderComment() string {
	return "# Code generated by selo gen --lang ruby. DO NOT EDIT.\n"
}

// rubyName returns the Ruby module-name suffix used in each kind's nested
// module, e.g. "cpf" -> "CPF", "voter_id" -> "VoterId" (mirrors tsName).
func rubyName(kind selo.Kind) string {
	switch kind {
	case selo.KindCPF:
		return "CPF"
	case selo.KindCNPJ:
		return "CNPJ"
	case selo.KindCNH:
		return "CNH"
	case selo.KindPIS:
		return "PIS"
	case selo.KindRenavam:
		return "Renavam"
	case selo.KindVoterID:
		return "VoterId"
	case selo.KindCEP:
		return "CEP"
	case selo.KindPhone:
		return "Phone"
	case selo.KindPlate:
		return "Plate"
	case selo.KindCNS:
		return "CNS"
	case selo.KindRG:
		return "RG"
	case selo.KindPIX:
		return "PIX"
	case selo.KindIE:
		return "IE"
	default:
		return strings.ToUpper(kind.String())
	}
}

// rubyIntSlice renders a Go int slice as a Ruby array literal: "[1, 2, 3]".
func rubyIntSlice(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}

	return "[" + strings.Join(parts, ", ") + "]"
}

// rubyCheckDigitLiteral renders a CheckDigit as a Ruby Hash literal matching the
// spec keys that Selo::Mod11.compute_digit expects.
func rubyCheckDigitLiteral(cd CheckDigit) string {
	fields := make([]string, 0, 6)

	fields = append(fields, "weights: "+rubyIntSlice(cd.Weights))
	fields = append(fields, "rule: '"+cd.Rule.String()+"'")

	if len(cd.RemainderTo0) > 0 {
		fields = append(fields, "remainder_to0: "+rubyIntSlice(cd.RemainderTo0))
	}

	if cd.MultiplyBy10 {
		fields = append(fields, "multiply_by10: true")
	}

	if cd.EncodeXAt != 0 {
		fields = append(fields, "encode_x_at: "+strconv.Itoa(cd.EncodeXAt))
	}

	if cd.EncodeZeroAt != 0 {
		fields = append(fields, "encode_zero_at: "+strconv.Itoa(cd.EncodeZeroAt))
	}

	return "{ " + strings.Join(fields, ", ") + " }"
}

// rubyRaise emits the standard Ruby raise for a format error, mapping a sentinel
// name to an ArgumentError whose message is the sentinel.
func rubyRaise(sentinel string) string {
	return "raise ArgumentError, " + strconv.Quote(sentinel)
}

// writeRubyHeader writes the generated-file banner and the standard
// require_relative lines for the shared mod11 reducer (always) and, when
// requireData is true, the embedded data tables.
func writeRubyHeader(b *strings.Builder, requireData bool) {
	b.WriteString(rubyHeaderComment())
	b.WriteString("\n")
	b.WriteString("require_relative 'mod11'\n")

	if requireData {
		b.WriteString("require_relative 'data'\n")
	}

	b.WriteString("\n")
}

// rubyMaskExpr converts a '#'/'X'-placeholder mask (e.g. "###.#####.##-#") into
// a Ruby string-interpolation expression slicing the cleaned digit variable v,
// e.g. "#{v[0, 3]}.#{v[3, 5]}.#{v[8, 2]}-#{v[10, 1]}".
func rubyMaskExpr(mask, v string) string {
	var b strings.Builder
	b.WriteString("\"")

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

			fmt.Fprintf(&b, "#{%s[%d, %d]}", v, start, pos-start)

			continue
		}
		// literal separator
		b.WriteByte(c)

		i++
	}

	b.WriteString("\"")

	return b.String()
}

// rubyStringArray renders the UFs list (fallback) as a Ruby array literal of
// quoted strings: ['SP', 'RJ'].
func rubyStringArray(fallback []string) string {
	quoted := make([]string, len(fallback))
	for i, s := range fallback {
		quoted[i] = "'" + s + "'"
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}
