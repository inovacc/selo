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

// emit_ts.go is the TypeScript language emitter (design spec §5/§6/§7, plan M2).
// It renders, for one kind, an idiomatic TS module (validate/format/origin and,
// for RG/IE, UF-param variants), a vitest test driven by the golden vector, the
// vector JSON itself, and the shared scaffolding (mod-11 reducer, package.json,
// tsconfig, vitest config, src/index.ts re-exports). The check-digit kinds reuse
// a single shared mod11.ts reducer parameterized by each kind's CheckDigit spec.
//
// The TS SOURCE is rendered purely from the declarative KindPlan and the static
// templates, so it is deterministic and snapshot-stable. Only the vector JSON is
// non-deterministic (it mixes selo.Generate output); the golden snapshot test
// compares the deterministic source files and validates the vectors against selo
// rather than byte-comparing them.

//go:embed templates/ts/mod11.ts.tmpl templates/ts/package.json.tmpl templates/ts/tsconfig.json.tmpl templates/ts/vitest.config.ts.tmpl
var tsTemplates embed.FS

func init() { Register(tsEmitter{}) }

// tsEmitter implements Emitter for TypeScript.
type tsEmitter struct{}

// Lang reports the target language.
func (tsEmitter) Lang() Lang { return LangTS }

// Emit renders the full TS file set for kind: the per-kind module, its test, the
// vector JSON, and the shared scaffolding (idempotent across kinds).
func (e tsEmitter) Emit(kind selo.Kind, plan KindPlan, vec Vector) ([]File, error) {
	module, err := e.renderModule(kind, plan)
	if err != nil {
		return nil, err
	}
	test := e.renderTest(kind, plan, vec)
	vectorJSON, err := json.MarshalIndent(vec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("codegen/ts: marshal %q vector: %w", kind, err)
	}
	vectorJSON = append(vectorJSON, '\n')

	files := []File{
		{Path: "src/" + kind.String() + ".ts", Content: []byte(module)},
		{Path: "test/" + kind.String() + ".test.ts", Content: []byte(test)},
		{Path: "vectors/" + kind.String() + ".json", Content: vectorJSON},
	}
	shared, err := e.sharedFiles()
	if err != nil {
		return nil, err
	}
	return append(files, shared...), nil
}

// sharedFiles returns the language-wide files that do not depend on a single
// kind: the mod-11 reducer, the embedded data tables, the index re-exports, and
// the build/test scaffolding. They are emitted on every per-kind call and are
// byte-identical, so re-writing them is idempotent.
func (e tsEmitter) sharedFiles() ([]File, error) {
	mod11, err := tsTemplates.ReadFile("templates/ts/mod11.ts.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/ts: read mod11 template: %w", err)
	}
	pkg, err := tsTemplates.ReadFile("templates/ts/package.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/ts: read package.json template: %w", err)
	}
	tsconfig, err := tsTemplates.ReadFile("templates/ts/tsconfig.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/ts: read tsconfig template: %w", err)
	}
	vitest, err := tsTemplates.ReadFile("templates/ts/vitest.config.ts.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/ts: read vitest template: %w", err)
	}
	return []File{
		{Path: "src/mod11.ts", Content: mod11},
		{Path: "src/data.ts", Content: []byte(e.renderData())},
		{Path: "src/index.ts", Content: []byte(e.renderIndex())},
		{Path: "package.json", Content: pkg},
		{Path: "tsconfig.json", Content: tsconfig},
		{Path: "vitest.config.ts", Content: vitest},
	}, nil
}

// renderIndex re-exports every kind's module from src/index.ts, in stable order.
func (e tsEmitter) renderIndex() string {
	var b strings.Builder
	b.WriteString(headerComment())
	b.WriteString("\n")
	for _, k := range KindStrings() {
		fmt.Fprintf(&b, "export * from \"./%s.js\";\n", k)
	}
	return b.String()
}

// renderData emits the embedded data tables (CEP ranges, DDD->UF, CPF region
// map, voter UF names) as TS constants, from the codegen accessors.
func (e tsEmitter) renderData() string {
	var b strings.Builder
	b.WriteString(headerComment())
	b.WriteString("\n")

	// CEP prefix ranges (scan order; first match wins).
	b.WriteString("export interface UFRange { uf: string; from: number; to: number }\n\n")
	b.WriteString("export const CEP_RANGES: UFRange[] = [\n")
	for _, r := range CEPRanges() {
		fmt.Fprintf(&b, "  { uf: %q, from: %d, to: %d },\n", r.UF, r.From, r.To)
	}
	b.WriteString("];\n\n")

	// DDD -> UF.
	b.WriteString("export const DDD_TO_UF: Record<string, string> = {\n")
	dddMap := DDDtoUF()
	ddds := make([]string, 0, len(dddMap))
	for d := range dddMap {
		ddds = append(ddds, d)
	}
	sort.Strings(ddds)
	for _, d := range ddds {
		fmt.Fprintf(&b, "  %q: %q,\n", d, dddMap[d].String())
	}
	b.WriteString("};\n\n")

	// CPF ninth-digit region map.
	b.WriteString("export const CPF_REGIONS: Record<number, string> = {\n")
	cpfRegions := CPFRegions()
	for i := 0; i <= 9; i++ {
		if name, ok := cpfRegions[i]; ok {
			fmt.Fprintf(&b, "  %d: %q,\n", i, name)
		}
	}
	b.WriteString("};\n\n")

	// Voter-ID UF code -> region name.
	b.WriteString("export const VOTER_UF_NAMES: Record<number, string> = {\n")
	voterNames := VoterUFNames()
	codes := make([]int, 0, len(voterNames))
	for c := range voterNames {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	for _, c := range codes {
		fmt.Fprintf(&b, "  %d: %q,\n", c, voterNames[c])
	}
	b.WriteString("};\n")
	return b.String()
}

// renderModule dispatches to the per-group renderer for kind.
func (e tsEmitter) renderModule(kind selo.Kind, plan KindPlan) (string, error) {
	switch kind {
	case selo.KindCPF:
		return e.renderCPF(plan), nil
	case selo.KindPIS:
		return e.renderSimpleNumeric(plan, "PIS", 11), nil
	case selo.KindRenavam:
		return e.renderRenavam(plan), nil
	case selo.KindCNH:
		return e.renderCNH(plan), nil
	case selo.KindRG:
		return e.renderRG(plan), nil
	case selo.KindIE:
		return e.renderIE(plan), nil
	case selo.KindCNS:
		return e.renderCNS(plan), nil
	case selo.KindCNPJ:
		return e.renderCNPJ(plan), nil
	case selo.KindPlate:
		return e.renderPlate(plan), nil
	case selo.KindPIX:
		return e.renderPIX(plan), nil
	case selo.KindCEP:
		return e.renderCEP(plan), nil
	case selo.KindPhone:
		return e.renderPhone(plan), nil
	case selo.KindVoterID:
		return e.renderVoterID(plan), nil
	default:
		return "", fmt.Errorf("codegen/ts: no module renderer for kind %q", kind)
	}
}

// --- shared rendering helpers -----------------------------------------------

// headerComment is the generated-file banner.
func headerComment() string {
	return "// Code generated by selo gen --lang ts. DO NOT EDIT.\n"
}

// tsName returns the PascalCase identifier suffix used in exported function
// names, e.g. "cpf" -> "CPF", "voter_id" -> "VoterId".
func tsName(kind selo.Kind) string {
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

// intSlice renders a Go int slice as a TS number[] literal: "[1, 2, 3]".
func intSlice(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// checkDigitLiteral renders a CheckDigit as a TS object literal matching the
// mod11.ts CheckDigit interface.
func checkDigitLiteral(cd CheckDigit) string {
	var fields []string
	fields = append(fields, "weights: "+intSlice(cd.Weights))
	fields = append(fields, "rule: \""+cd.Rule.String()+"\"")
	if len(cd.RemainderTo0) > 0 {
		fields = append(fields, "remainderTo0: "+intSlice(cd.RemainderTo0))
	}
	if cd.MultiplyBy10 {
		fields = append(fields, "multiplyBy10: true")
	}
	if cd.EncodeXAt != 0 {
		fields = append(fields, "encodeXAt: "+strconv.Itoa(cd.EncodeXAt))
	}
	if cd.EncodeZeroAt != 0 {
		fields = append(fields, "encodeZeroAt: "+strconv.Itoa(cd.EncodeZeroAt))
	}
	return "{ " + strings.Join(fields, ", ") + " }"
}

// formatErrorThrow emits the standard TS throw for a format error, mapping a
// sentinel name to a thrown Error whose message is the sentinel.
func formatErrorThrow(sentinel string) string {
	return "throw new Error(" + strconv.Quote(sentinel) + ");"
}
