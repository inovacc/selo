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

// emit_php.go is the PHP language emitter (the 7th target; mirrors the Python
// reference). For one kind it renders an idiomatic PHP class (src/<Kind>.php
// exposing static validate/format/origin/generate methods and, for RG/IE,
// UF-param variants), a PHPUnit test driven by the golden vector, the vector
// JSON itself, and the shared scaffolding (mod-11 reducer, embedded data tables,
// composer.json, phpunit.xml). The check-digit kinds reuse a single shared
// Selo\Mod11 reducer parameterized by each kind's CheckDigit spec.
//
// Every PHP algorithm is translated VERBATIM (logic) from the proven Python
// emitter — only the syntax differs — so the port is correct by construction and
// the PHP vector tests (run on CI) pass. The PHP SOURCE is deterministic and
// snapshot-stable; only the vector JSON is non-deterministic (it mixes
// selo.Generate output), so the golden snapshot compares the deterministic
// source files and re-validates the vectors against selo.

//go:embed templates/php/Mod11.php.tmpl templates/php/composer.json.tmpl templates/php/phpunit.xml.tmpl
var phpTemplates embed.FS

func init() { Register(phpEmitter{}) }

// phpEmitter implements Emitter for PHP.
type phpEmitter struct{}

// Lang reports the target language.
func (phpEmitter) Lang() Lang { return LangPHP }

// Emit renders the full PHP file set for kind: the per-kind class, its test, the
// vector JSON, and the shared scaffolding (idempotent across kinds).
func (e phpEmitter) Emit(kind selo.Kind, plan KindPlan, vec Vector) ([]File, error) {
	module, err := e.renderModule(kind, plan)
	if err != nil {
		return nil, err
	}

	test := e.renderTest(kind)

	vectorJSON, err := json.MarshalIndent(vec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("codegen/php: marshal %q vector: %w", kind, err)
	}

	vectorJSON = append(vectorJSON, '\n')

	shared, err := e.sharedFiles()
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, 3+len(shared))
	files = append(files,
		File{Path: "src/" + phpClassName(kind) + ".php", Content: []byte(module)},
		File{Path: "tests/" + phpClassName(kind) + "Test.php", Content: []byte(test)},
		File{Path: "vectors/" + kind.String() + ".json", Content: vectorJSON},
	)
	files = append(files, shared...)

	return files, nil
}

// sharedFiles returns the language-wide files that do not depend on a single
// kind: the mod-11 reducer, the embedded data tables, and the build/test
// scaffolding. They are emitted on every per-kind call and are byte-identical,
// so re-writing them is idempotent.
func (e phpEmitter) sharedFiles() ([]File, error) {
	mod11, err := phpTemplates.ReadFile("templates/php/Mod11.php.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/php: read Mod11 template: %w", err)
	}

	composer, err := phpTemplates.ReadFile("templates/php/composer.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/php: read composer template: %w", err)
	}

	phpunit, err := phpTemplates.ReadFile("templates/php/phpunit.xml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/php: read phpunit template: %w", err)
	}

	return []File{
		{Path: "src/Mod11.php", Content: mod11},
		{Path: "src/Data.php", Content: []byte(e.renderData())},
		{Path: "composer.json", Content: composer},
		{Path: "phpunit.xml", Content: phpunit},
	}, nil
}

// renderData emits the embedded data tables (CEP ranges, DDD->UF, CPF region
// map, voter UF names) as a PHP class with static array constants, from the
// codegen accessors. Sort order matches the Python data renderer for parity.
func (e phpEmitter) renderData() string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("/**\n")
	b.WriteString(" * Embedded data tables (CEP ranges, DDD->UF, CPF region, voter UF names).\n")
	b.WriteString(" */\n")
	b.WriteString("final class Data\n{\n")

	// CEP prefix ranges (scan order; first match wins).
	b.WriteString("    /** @var array<int, array{uf: string, from: int, to: int}> */\n")
	b.WriteString("    public const CEP_RANGES = [\n")

	for _, r := range CEPRanges() {
		fmt.Fprintf(&b, "        ['uf' => %s, 'from' => %d, 'to' => %d],\n", phpQuote(r.UF), r.From, r.To)
	}

	b.WriteString("    ];\n\n")

	// DDD -> UF.
	b.WriteString("    /** @var array<string, string> */\n")
	b.WriteString("    public const DDD_TO_UF = [\n")

	dddMap := DDDtoUF()

	ddds := make([]string, 0, len(dddMap))
	for d := range dddMap {
		ddds = append(ddds, d)
	}

	sort.Strings(ddds)

	for _, d := range ddds {
		fmt.Fprintf(&b, "        %s => %s,\n", phpQuote(d), phpQuote(dddMap[d].String()))
	}

	b.WriteString("    ];\n\n")

	// CPF ninth-digit region map.
	b.WriteString("    /** @var array<int, string> */\n")
	b.WriteString("    public const CPF_REGIONS = [\n")

	cpfRegions := CPFRegions()
	for i := 0; i <= 9; i++ {
		if name, ok := cpfRegions[i]; ok {
			fmt.Fprintf(&b, "        %d => %s,\n", i, phpQuote(name))
		}
	}

	b.WriteString("    ];\n\n")

	// Voter-ID UF code -> region name.
	b.WriteString("    /** @var array<int, string> */\n")
	b.WriteString("    public const VOTER_UF_NAMES = [\n")

	voterNames := VoterUFNames()

	codes := make([]int, 0, len(voterNames))
	for c := range voterNames {
		codes = append(codes, c)
	}

	sort.Ints(codes)

	for _, c := range codes {
		fmt.Fprintf(&b, "        %d => %s,\n", c, phpQuote(voterNames[c]))
	}

	b.WriteString("    ];\n")
	b.WriteString("}\n")

	return b.String()
}

// renderModule dispatches to the per-group renderer for kind.
func (e phpEmitter) renderModule(kind selo.Kind, plan KindPlan) (string, error) {
	switch kind {
	case selo.KindCPF:
		return e.renderCPF(plan), nil
	case selo.KindPIS:
		return e.renderSimpleNumeric(plan, "Pis", 11), nil
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
		return "", fmt.Errorf("codegen/php: no module renderer for kind %q", kind)
	}
}

// --- shared rendering helpers -----------------------------------------------

// phpFileHeader is the generated-file banner plus the PHP open tag and strict
// types declaration that every emitted PHP source begins with.
func phpFileHeader() string {
	return "<?php\n\n// Code generated by selo gen --lang php. DO NOT EDIT.\n\ndeclare(strict_types=1);\n\n"
}

// phpClassName returns the PascalCase class name for a kind, e.g. "cpf" -> "Cpf",
// "voter_id" -> "VoterId". selo kinds are snake_case; each underscore-separated
// word is title-cased.
func phpClassName(kind selo.Kind) string {
	parts := strings.Split(kind.String(), "_")

	var b strings.Builder

	for _, p := range parts {
		if p == "" {
			continue
		}

		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(p[1:])
	}

	return b.String()
}

// phpFnSuffix returns the camelCase suffix used in each kind's public method
// names, e.g. "cpf" -> "Cpf", "voter_id" -> "VoterId". Methods are
// validateCpf/formatCpf/etc., so the suffix is PascalCase appended to the verb.
func phpFnSuffix(kind selo.Kind) string {
	return phpClassName(kind)
}

// phpQuote renders s as a single-quoted PHP string literal (escaping backslash
// and single quote, which is all single-quoted PHP strings interpret).
func phpQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + r.Replace(s) + "'"
}

// phpIntList renders a Go int slice as a PHP array literal: "[1, 2, 3]".
func phpIntList(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}

	return "[" + strings.Join(parts, ", ") + "]"
}

// phpCheckDigitLiteral renders a CheckDigit as a PHP associative-array literal
// matching the spec keys that Mod11::computeDigit expects.
func phpCheckDigitLiteral(cd CheckDigit) string {
	fields := make([]string, 0, 6)

	fields = append(fields, "'weights' => "+phpIntList(cd.Weights))
	fields = append(fields, "'rule' => '"+cd.Rule.String()+"'")

	if len(cd.RemainderTo0) > 0 {
		fields = append(fields, "'remainder_to0' => "+phpIntList(cd.RemainderTo0))
	}

	if cd.MultiplyBy10 {
		fields = append(fields, "'multiply_by10' => true")
	}

	if cd.EncodeXAt != 0 {
		fields = append(fields, "'encode_x_at' => "+strconv.Itoa(cd.EncodeXAt))
	}

	if cd.EncodeZeroAt != 0 {
		fields = append(fields, "'encode_zero_at' => "+strconv.Itoa(cd.EncodeZeroAt))
	}

	return "[" + strings.Join(fields, ", ") + "]"
}

// phpThrow emits the standard PHP throw for a format error, mapping a sentinel
// name to an InvalidArgumentException whose message is the sentinel.
func phpThrow(sentinel string) string {
	return "throw new \\InvalidArgumentException(" + phpQuote(sentinel) + ");"
}

// phpMaskExpr converts a '#'/'X'-placeholder mask (e.g. "###.#####.##-#") into a
// PHP string-concatenation expression slicing the cleaned digit variable named
// by v (e.g. $d), using substr($d, start, len). Returns the full RHS expression.
func phpMaskExpr(mask, v string) string {
	var parts []string

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

			parts = append(parts, fmt.Sprintf("substr(%s, %d, %d)", v, start, pos-start))

			continue
		}
		// literal separator
		parts = append(parts, phpQuote(string(c)))

		i++
	}

	return strings.Join(parts, " . ")
}

// phpStringList renders a string slice as a PHP array literal of single-quoted
// strings: ['SP', 'RJ'].
func phpStringList(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = phpQuote(s)
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}

// phpHasOrigin reports whether kind has an origin resolver in the generated PHP
// class (mirrors pythonHasOrigin).
func phpHasOrigin(kind selo.Kind) bool {
	switch kind { //nolint:exhaustive // only origin-capable kinds return true; all others fall through
	case selo.KindCPF, selo.KindCEP, selo.KindPhone, selo.KindVoterID:
		return true
	default:
		return false
	}
}
