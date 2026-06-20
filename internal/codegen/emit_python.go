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

// emit_python.go is the Python language emitter (plan M6; mirrors the M2
// TypeScript reference). For one kind it renders an idiomatic Python module
// (selo/<kind>.py exposing validate_/format_/origin_/generate_ functions and,
// for RG/IE, UF-param variants), a pytest test driven by the golden vector, the
// vector JSON itself, and the shared scaffolding (mod-11 reducer, embedded data
// tables, package __init__, pyproject.toml). The check-digit kinds reuse a
// single shared selo/mod11.py reducer parameterized by each kind's CheckDigit
// spec.
//
// Every Python algorithm is translated VERBATIM (logic) from the proven TS
// emitter — only the syntax differs — so the port is correct by construction and
// the Python vector tests (run on CI) pass. The Python SOURCE is deterministic
// and snapshot-stable; only the vector JSON is non-deterministic (it mixes
// selo.Generate output), so the golden snapshot compares the deterministic
// source files and re-validates the vectors against selo.

//go:embed templates/python/mod11.py.tmpl templates/python/pyproject.toml.tmpl
var pythonTemplates embed.FS

func init() { Register(pythonEmitter{}) }

// pythonEmitter implements Emitter for Python.
type pythonEmitter struct{}

// Lang reports the target language.
func (pythonEmitter) Lang() Lang { return LangPython }

// Emit renders the full Python file set for kind: the per-kind module, its test,
// the vector JSON, and the shared scaffolding (idempotent across kinds).
func (e pythonEmitter) Emit(kind selo.Kind, plan KindPlan, vec Vector) ([]File, error) {
	module, err := e.renderModule(kind, plan)
	if err != nil {
		return nil, err
	}

	test := e.renderTest(kind)

	vectorJSON, err := json.MarshalIndent(vec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("codegen/python: marshal %q vector: %w", kind, err)
	}

	vectorJSON = append(vectorJSON, '\n')

	shared, err := e.sharedFiles()
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, 3+len(shared))
	files = append(files,
		File{Path: "selo/" + kind.String() + ".py", Content: []byte(module)},
		File{Path: "tests/test_" + kind.String() + ".py", Content: []byte(test)},
		File{Path: "vectors/" + kind.String() + ".json", Content: vectorJSON},
	)
	files = append(files, shared...)

	return files, nil
}

// sharedFiles returns the language-wide files that do not depend on a single
// kind: the mod-11 reducer, the embedded data tables, the package __init__, and
// the build/test scaffolding. They are emitted on every per-kind call and are
// byte-identical, so re-writing them is idempotent.
func (e pythonEmitter) sharedFiles() ([]File, error) {
	mod11, err := pythonTemplates.ReadFile("templates/python/mod11.py.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/python: read mod11 template: %w", err)
	}

	pyproject, err := pythonTemplates.ReadFile("templates/python/pyproject.toml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/python: read pyproject template: %w", err)
	}

	return []File{
		{Path: "selo/mod11.py", Content: mod11},
		{Path: "selo/data.py", Content: []byte(e.renderData())},
		{Path: "selo/__init__.py", Content: []byte(e.renderIndex())},
		{Path: "pyproject.toml", Content: pyproject},
	}, nil
}

// renderIndex re-exports every kind's module from selo/__init__.py, in stable
// order, so callers can `from selo import validate_cpf` (mirrors the TS index).
func (e pythonEmitter) renderIndex() string {
	var b strings.Builder
	b.WriteString(pythonHeaderComment())
	b.WriteString("\n")

	for _, k := range KindStrings() {
		fmt.Fprintf(&b, "from .%s import *  # noqa: F401,F403\n", k)
	}

	return b.String()
}

// renderData emits the embedded data tables (CEP ranges, DDD->UF, CPF region
// map, voter UF names) as Python module-level constants, from the codegen
// accessors. Sort order matches the TS data renderer for snapshot parity.
func (e pythonEmitter) renderData() string {
	var b strings.Builder
	b.WriteString(pythonHeaderComment())
	b.WriteString("\n")
	b.WriteString("from typing import Dict, List, TypedDict\n\n\n")

	// CEP prefix ranges (scan order; first match wins).
	b.WriteString("class UFRange(TypedDict):\n")
	b.WriteString("    uf: str\n")
	b.WriteString("    from_: int\n")
	b.WriteString("    to: int\n\n\n")
	b.WriteString("CEP_RANGES: List[UFRange] = [\n")

	for _, r := range CEPRanges() {
		fmt.Fprintf(&b, "    {\"uf\": %q, \"from_\": %d, \"to\": %d},\n", r.UF, r.From, r.To)
	}

	b.WriteString("]\n\n")

	// DDD -> UF.
	b.WriteString("DDD_TO_UF: Dict[str, str] = {\n")

	dddMap := DDDtoUF()

	ddds := make([]string, 0, len(dddMap))
	for d := range dddMap {
		ddds = append(ddds, d)
	}

	sort.Strings(ddds)

	for _, d := range ddds {
		fmt.Fprintf(&b, "    %q: %q,\n", d, dddMap[d].String())
	}

	b.WriteString("}\n\n")

	// CPF ninth-digit region map.
	b.WriteString("CPF_REGIONS: Dict[int, str] = {\n")

	cpfRegions := CPFRegions()
	for i := 0; i <= 9; i++ {
		if name, ok := cpfRegions[i]; ok {
			fmt.Fprintf(&b, "    %d: %q,\n", i, name)
		}
	}

	b.WriteString("}\n\n")

	// Voter-ID UF code -> region name.
	b.WriteString("VOTER_UF_NAMES: Dict[int, str] = {\n")

	voterNames := VoterUFNames()

	codes := make([]int, 0, len(voterNames))
	for c := range voterNames {
		codes = append(codes, c)
	}

	sort.Ints(codes)

	for _, c := range codes {
		fmt.Fprintf(&b, "    %d: %q,\n", c, voterNames[c])
	}

	b.WriteString("}\n")

	return b.String()
}

// renderModule dispatches to the per-group renderer for kind.
func (e pythonEmitter) renderModule(kind selo.Kind, plan KindPlan) (string, error) {
	switch kind {
	case selo.KindCPF:
		return e.renderCPF(plan), nil
	case selo.KindPIS:
		return e.renderSimpleNumeric(plan, "pis", 11), nil
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
		return "", fmt.Errorf("codegen/python: no module renderer for kind %q", kind)
	}
}

// --- shared rendering helpers -----------------------------------------------

// pythonHeaderComment is the generated-file banner.
func pythonHeaderComment() string {
	return "# Code generated by selo gen --lang python. DO NOT EDIT.\n"
}

// pythonName returns the snake_case suffix used in each kind's public function
// names, e.g. "cpf" -> "cpf", "voter_id" -> "voter_id". It is the kind string
// itself (selo kinds are already snake_case), kept as a helper for parity with
// the TS/Ruby name mappers.
func pythonName(kind selo.Kind) string {
	return kind.String()
}

// pythonIntList renders a Go int slice as a Python list literal: "[1, 2, 3]".
func pythonIntList(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}

	return "[" + strings.Join(parts, ", ") + "]"
}

// pythonCheckDigitLiteral renders a CheckDigit as a Python dict literal matching
// the spec keys that mod11.compute_digit expects.
func pythonCheckDigitLiteral(cd CheckDigit) string {
	fields := make([]string, 0, 6)

	fields = append(fields, "\"weights\": "+pythonIntList(cd.Weights))
	fields = append(fields, "\"rule\": \""+cd.Rule.String()+"\"")

	if len(cd.RemainderTo0) > 0 {
		fields = append(fields, "\"remainder_to0\": "+pythonIntList(cd.RemainderTo0))
	}

	if cd.MultiplyBy10 {
		fields = append(fields, "\"multiply_by10\": True")
	}

	if cd.EncodeXAt != 0 {
		fields = append(fields, "\"encode_x_at\": "+strconv.Itoa(cd.EncodeXAt))
	}

	if cd.EncodeZeroAt != 0 {
		fields = append(fields, "\"encode_zero_at\": "+strconv.Itoa(cd.EncodeZeroAt))
	}

	return "{" + strings.Join(fields, ", ") + "}"
}

// pythonRaise emits the standard Python raise for a format error, mapping a
// sentinel name to a ValueError whose message is the sentinel.
func pythonRaise(sentinel string) string {
	return "raise ValueError(" + strconv.Quote(sentinel) + ")"
}

// writePythonHeader writes the generated-file banner and the standard imports
// from the shared mod11 reducer. importNames are the symbols imported from
// .mod11; when importData is true the named data-table symbols are imported from
// .data.
func writePythonHeader(b *strings.Builder, importNames []string, dataImports string) {
	b.WriteString(pythonHeaderComment())
	b.WriteString("\n")
	b.WriteString("from __future__ import annotations\n\n")
	b.WriteString("import random\n\n")
	fmt.Fprintf(b, "from .mod11 import %s\n", strings.Join(importNames, ", "))

	if dataImports != "" {
		fmt.Fprintf(b, "from .data import %s\n", dataImports)
	}

	b.WriteString("\n")
}

// pythonMaskExpr converts a '#'/'X'-placeholder mask (e.g. "###.#####.##-#")
// into a Python f-string interpolation expression slicing the cleaned digit
// variable v, e.g. f"{v[0:3]}.{v[3:8]}.{v[8:10]}-{v[10:11]}". The leading f"
// and trailing " are included.
func pythonMaskExpr(mask, v string) string {
	var b strings.Builder
	b.WriteString("f\"")

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

			fmt.Fprintf(&b, "{%s[%d:%d]}", v, start, pos)

			continue
		}
		// literal separator
		b.WriteByte(c)

		i++
	}

	b.WriteString("\"")

	return b.String()
}

// pythonStringList renders the UFs list (fallback) as a Python list literal of
// quoted strings: ["SP", "RJ"].
func pythonStringList(fallback []string) string {
	quoted := make([]string, len(fallback))
	for i, s := range fallback {
		quoted[i] = strconv.Quote(s)
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}
