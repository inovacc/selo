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

// emit_js.go is the JavaScript language emitter (design spec §5/§6/§7, plan M3).
// It mirrors the TypeScript emitter (M2) but omits type annotations, interfaces,
// and tsconfig scaffolding. The output is idiomatic ESM JavaScript.

//go:embed templates/js/mod11.js.tmpl templates/js/package.json.tmpl templates/js/vitest.config.js.tmpl
var jsTemplates embed.FS

func init() { Register(jsEmitter{}) }

// jsEmitter implements Emitter for JavaScript.
type jsEmitter struct{}

// Lang reports the target language.
func (jsEmitter) Lang() Lang { return LangJS }

// Emit renders the full JS file set for kind.
func (e jsEmitter) Emit(kind selo.Kind, plan KindPlan, vec Vector) ([]File, error) {
	module, err := e.renderModule(kind, plan)
	if err != nil {
		return nil, err
	}

	test := e.renderTest(kind, plan, vec)

	vectorJSON, err := json.MarshalIndent(vec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("codegen/js: marshal %q vector: %w", kind, err)
	}

	vectorJSON = append(vectorJSON, '\n')

	shared, err := e.sharedFiles()
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, 3+len(shared))
	files = append(files,
		File{Path: "src/" + kind.String() + ".js", Content: []byte(module)},
		File{Path: "test/" + kind.String() + ".test.js", Content: []byte(test)},
		File{Path: "vectors/" + kind.String() + ".json", Content: vectorJSON},
	)
	files = append(files, shared...)

	return files, nil
}

// sharedFiles returns the language-wide files that do not depend on a single kind.
func (e jsEmitter) sharedFiles() ([]File, error) {
	mod11, err := jsTemplates.ReadFile("templates/js/mod11.js.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/js: read mod11 template: %w", err)
	}

	pkg, err := jsTemplates.ReadFile("templates/js/package.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/js: read package.json template: %w", err)
	}

	vitest, err := jsTemplates.ReadFile("templates/js/vitest.config.js.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/js: read vitest template: %w", err)
	}

	return []File{
		{Path: "src/mod11.js", Content: mod11},
		{Path: "src/data.js", Content: []byte(e.renderData())},
		{Path: "src/index.js", Content: []byte(e.renderIndex())},
		{Path: "package.json", Content: pkg},
		{Path: "vitest.config.js", Content: vitest},
	}, nil
}

// renderIndex re-exports every kind's module from src/index.js, in stable order.
func (e jsEmitter) renderIndex() string {
	var b strings.Builder
	b.WriteString(jsHeaderComment())
	b.WriteString("\n")

	for _, k := range KindStrings() {
		fmt.Fprintf(&b, "export * from \"./%s.js\";\n", k)
	}

	return b.String()
}

// renderData emits the embedded data tables as JS constants (no type annotations).
func (e jsEmitter) renderData() string {
	var b strings.Builder
	b.WriteString(jsHeaderComment())
	b.WriteString("\n")

	// CEP prefix ranges (scan order; first match wins).
	b.WriteString("export const CEP_RANGES = [\n")

	for _, r := range CEPRanges() {
		fmt.Fprintf(&b, "  { uf: %q, from: %d, to: %d },\n", r.UF, r.From, r.To)
	}

	b.WriteString("];\n\n")

	// DDD -> UF.
	b.WriteString("export const DDD_TO_UF = {\n")

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
	b.WriteString("export const CPF_REGIONS = {\n")

	cpfRegions := CPFRegions()
	for i := 0; i <= 9; i++ {
		if name, ok := cpfRegions[i]; ok {
			fmt.Fprintf(&b, "  %d: %q,\n", i, name)
		}
	}

	b.WriteString("};\n\n")

	// Voter-ID UF code -> region name.
	b.WriteString("export const VOTER_UF_NAMES = {\n")

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
func (e jsEmitter) renderModule(kind selo.Kind, plan KindPlan) (string, error) {
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
		return "", fmt.Errorf("codegen/js: no module renderer for kind %q", kind)
	}
}

// jsHeaderComment is the generated-file banner for JS files.
func jsHeaderComment() string {
	return "// Code generated by selo gen --lang js. DO NOT EDIT.\n"
}

// jsCheckDigitLiteral renders a CheckDigit as a JS object literal.
func jsCheckDigitLiteral(cd CheckDigit) string {
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

// jsFormatErrorThrow emits the standard JS throw for a format error.
func jsFormatErrorThrow(sentinel string) string {
	return "throw new Error(" + strconv.Quote(sentinel) + ");"
}

// jsWriteHeader writes the generated-file banner and imports for JS modules.
func jsWriteHeader(b *strings.Builder, dataImports string) {
	b.WriteString(jsHeaderComment())
	b.WriteString("\n")
	b.WriteString("import {\n")
	b.WriteString("  charValue,\n")
	b.WriteString("  weightedSum,\n")
	b.WriteString("  computeDigit,\n")
	b.WriteString("  encodeDigit,\n")
	b.WriteString("  onlyDigits,\n")
	b.WriteString("  allEqual,\n")
	b.WriteString("} from \"./mod11.js\";\n")

	if dataImports != "" {
		fmt.Fprintf(b, "import { %s } from \"./data.js\";\n", dataImports)
	}

	b.WriteString("\n")
}
