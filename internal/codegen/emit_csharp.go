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

// emit_csharp.go is the C# language emitter (plan M6; mirrors the M2 TypeScript
// reference in emit_ts.go). It renders, for one kind, an idiomatic C# class
// (static Validate/Format/Origin and, for RG/IE, UF-param variants), an xUnit
// test driven by the golden vector, the vector JSON itself, and the shared
// scaffolding (mod-11 reducer, data tables, .csproj/.sln). The check-digit kinds
// reuse a single shared Mod11.cs reducer parameterized by each kind's CheckDigit
// spec, exactly as the TS port reuses mod11.ts.
//
// As in TS, the C# SOURCE is rendered purely from the declarative KindPlan and
// the static templates, so it is deterministic and snapshot-stable. Only the
// vector JSON is non-deterministic (it mixes selo.Generate output); the golden
// snapshot test compares the deterministic source files and validates the
// vectors against selo rather than byte-comparing them.

//go:embed templates/csharp/Mod11.cs.tmpl templates/csharp/VectorModel.cs.tmpl templates/csharp/Selo.csproj.tmpl templates/csharp/Selo.Tests.csproj.tmpl templates/csharp/Selo.sln.tmpl
var csharpTemplates embed.FS

func init() { Register(csharpEmitter{}) }

// csharpEmitter implements Emitter for C#.
type csharpEmitter struct{}

// Lang reports the target language.
func (csharpEmitter) Lang() Lang { return LangCSharp }

// Emit renders the full C# file set for kind: the per-kind class, its test, the
// vector JSON, and the shared scaffolding (idempotent across kinds).
func (e csharpEmitter) Emit(kind selo.Kind, plan KindPlan, vec Vector) ([]File, error) {
	module, err := e.renderModule(kind, plan)
	if err != nil {
		return nil, err
	}

	test := e.renderTest(kind, plan, vec)

	vectorJSON, err := json.MarshalIndent(vec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("codegen/csharp: marshal %q vector: %w", kind, err)
	}

	vectorJSON = append(vectorJSON, '\n')

	shared, err := e.sharedFiles()
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, 3+len(shared))
	files = append(files,
		File{Path: "src/Selo/" + csName(kind) + ".cs", Content: []byte(module)},
		File{Path: "src/Selo.Tests/" + csName(kind) + "Tests.cs", Content: []byte(test)},
		File{Path: "vectors/" + kind.String() + ".json", Content: vectorJSON},
	)
	files = append(files, shared...)

	return files, nil
}

// sharedFiles returns the language-wide files that do not depend on a single
// kind: the mod-11 reducer, the embedded data tables, and the build/test
// scaffolding. They are emitted on every per-kind call and are byte-identical,
// so re-writing them is idempotent.
func (e csharpEmitter) sharedFiles() ([]File, error) {
	mod11, err := csharpTemplates.ReadFile("templates/csharp/Mod11.cs.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/csharp: read Mod11 template: %w", err)
	}

	lib, err := csharpTemplates.ReadFile("templates/csharp/Selo.csproj.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/csharp: read Selo.csproj template: %w", err)
	}

	tests, err := csharpTemplates.ReadFile("templates/csharp/Selo.Tests.csproj.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/csharp: read Selo.Tests.csproj template: %w", err)
	}

	sln, err := csharpTemplates.ReadFile("templates/csharp/Selo.sln.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/csharp: read Selo.sln template: %w", err)
	}

	vectorModel, err := csharpTemplates.ReadFile("templates/csharp/VectorModel.cs.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/csharp: read VectorModel template: %w", err)
	}

	return []File{
		{Path: "src/Selo/Mod11.cs", Content: mod11},
		{Path: "src/Selo/Data.cs", Content: []byte(e.renderData())},
		{Path: "src/Selo/Selo.csproj", Content: lib},
		{Path: "src/Selo.Tests/Selo.Tests.csproj", Content: tests},
		{Path: "src/Selo.Tests/VectorModel.cs", Content: vectorModel},
		{Path: "Selo.sln", Content: sln},
	}, nil
}

// renderData emits the embedded data tables (CEP ranges, DDD->UF, CPF region
// map, voter UF names) as C# static readonly members, from the codegen
// accessors. Mirrors emit_ts.go renderData.
func (e csharpEmitter) renderData() string {
	var b strings.Builder
	b.WriteString(csHeaderComment())
	b.WriteString("\n")
	b.WriteString("using System.Collections.Generic;\n\n")
	b.WriteString("namespace Inovacc.Selo\n{\n")

	// UFRange record + CEP prefix ranges (scan order; first match wins).
	b.WriteString("    /// <summary>One CEP prefix range mapped to a UF (inclusive 3-digit prefixes).</summary>\n")
	b.WriteString("    public readonly struct UFRange\n    {\n")
	b.WriteString("        public UFRange(string uf, int from, int to)\n        {\n")
	b.WriteString("            Uf = uf;\n            From = from;\n            To = to;\n        }\n\n")
	b.WriteString("        public string Uf { get; }\n\n")
	b.WriteString("        public int From { get; }\n\n")
	b.WriteString("        public int To { get; }\n    }\n\n")

	b.WriteString("    /// <summary>Embedded selo data tables rendered as C# constants.</summary>\n")
	b.WriteString("    public static class Data\n    {\n")

	// CEP ranges.
	b.WriteString("        /// <summary>CEP prefix-&gt;UF allocation table in scan order (first match wins).</summary>\n")
	b.WriteString("        public static readonly UFRange[] CepRanges =\n        {\n")

	for _, r := range CEPRanges() {
		fmt.Fprintf(&b, "            new UFRange(%s, %d, %d),\n", strconv.Quote(r.UF), r.From, r.To)
	}

	b.WriteString("        };\n\n")

	// DDD -> UF.
	b.WriteString("        /// <summary>DDD area-code-&gt;UF map.</summary>\n")
	b.WriteString("        public static readonly Dictionary<string, string> DddToUf = new Dictionary<string, string>\n        {\n")

	dddMap := DDDtoUF()

	ddds := make([]string, 0, len(dddMap))
	for d := range dddMap {
		ddds = append(ddds, d)
	}

	sort.Strings(ddds)

	for _, d := range ddds {
		fmt.Fprintf(&b, "            [%s] = %s,\n", strconv.Quote(d), strconv.Quote(dddMap[d].String()))
	}

	b.WriteString("        };\n\n")

	// CPF ninth-digit region map.
	b.WriteString("        /// <summary>CPF ninth-digit-&gt;region map.</summary>\n")
	b.WriteString("        public static readonly Dictionary<int, string> CpfRegions = new Dictionary<int, string>\n        {\n")

	cpfRegions := CPFRegions()
	for i := 0; i <= 9; i++ {
		if name, ok := cpfRegions[i]; ok {
			fmt.Fprintf(&b, "            [%d] = %s,\n", i, strconv.Quote(name))
		}
	}

	b.WriteString("        };\n\n")

	// Voter-ID UF code -> region name.
	b.WriteString("        /// <summary>Voter-ID UF code (1..28)-&gt;region name map.</summary>\n")
	b.WriteString("        public static readonly Dictionary<int, string> VoterUfNames = new Dictionary<int, string>\n        {\n")

	voterNames := VoterUFNames()

	codes := make([]int, 0, len(voterNames))
	for c := range voterNames {
		codes = append(codes, c)
	}

	sort.Ints(codes)

	for _, c := range codes {
		fmt.Fprintf(&b, "            [%d] = %s,\n", c, strconv.Quote(voterNames[c]))
	}

	b.WriteString("        };\n")
	b.WriteString("    }\n}\n")

	return b.String()
}

// renderModule dispatches to the per-group renderer for kind.
func (e csharpEmitter) renderModule(kind selo.Kind, plan KindPlan) (string, error) {
	switch kind {
	case selo.KindCPF:
		return e.renderCPF(plan), nil
	case selo.KindPIS:
		return e.renderSimpleNumeric(plan, kind, 11), nil
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
		return "", fmt.Errorf("codegen/csharp: no module renderer for kind %q", kind)
	}
}

// --- shared rendering helpers -----------------------------------------------

// csHeaderComment is the generated-file banner.
func csHeaderComment() string {
	return "// Code generated by selo gen --lang csharp. DO NOT EDIT.\n"
}

// csName returns the PascalCase C# class-name identifier for kind, e.g.
// "cpf" -> "Cpf", "voter_id" -> "VoterId".
func csName(kind selo.Kind) string {
	switch kind {
	case selo.KindCPF:
		return "Cpf"
	case selo.KindCNPJ:
		return "Cnpj"
	case selo.KindCNH:
		return "Cnh"
	case selo.KindPIS:
		return "Pis"
	case selo.KindRenavam:
		return "Renavam"
	case selo.KindVoterID:
		return "VoterId"
	case selo.KindCEP:
		return "Cep"
	case selo.KindPhone:
		return "Phone"
	case selo.KindPlate:
		return "Plate"
	case selo.KindCNS:
		return "Cns"
	case selo.KindRG:
		return "Rg"
	case selo.KindPIX:
		return "Pix"
	case selo.KindIE:
		return "Ie"
	default:
		return strings.Title(kind.String()) //nolint:staticcheck // simple ASCII title-casing fallback
	}
}

// csIntArray renders a Go int slice as a C# array initializer body: "1, 2, 3".
func csIntArray(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}

	return strings.Join(parts, ", ")
}

// csCheckDigitLiteral renders a CheckDigit as a C# object-initializer expression
// matching the Mod11.cs CheckDigit type.
func csCheckDigitLiteral(cd CheckDigit) string {
	var fields []string

	fields = append(fields, "Weights = new[] { "+csIntArray(cd.Weights)+" }")
	fields = append(fields, "Rule = DVRule."+csRuleName(cd.Rule))

	if len(cd.RemainderTo0) > 0 {
		fields = append(fields, "RemainderTo0 = new[] { "+csIntArray(cd.RemainderTo0)+" }")
	}

	if cd.MultiplyBy10 {
		fields = append(fields, "MultiplyBy10 = true")
	}

	if cd.EncodeXAt != 0 {
		fields = append(fields, "EncodeXAt = "+strconv.Itoa(cd.EncodeXAt))
	}

	if cd.EncodeZeroAt != 0 {
		fields = append(fields, "EncodeZeroAt = "+strconv.Itoa(cd.EncodeZeroAt))
	}

	return "new CheckDigit { " + strings.Join(fields, ", ") + " }"
}

// csRuleName maps a DVRule to its C# enum member name (mirrors mod11.ts DVRule).
func csRuleName(r DVRule) string {
	switch r {
	case DVElevenMinus:
		return "ElevenMinus"
	case DVModRemainder:
		return "ModRemainder"
	case DVRightmostDigit:
		return "RightmostDigit"
	case DVSumZero:
		return "SumZero"
	default:
		return "ModRemainder"
	}
}

// csFormatThrow emits the standard C# throw for a format error, mapping a
// sentinel name to a thrown FormatException whose message is the sentinel.
func csFormatThrow(sentinel string) string {
	return "throw new System.FormatException(" + strconv.Quote(sentinel) + ");"
}

// csStringArray renders a slice of UF strings as a C# array initializer body.
func csStringArray(_ KindPlan, fallback []string) string {
	quoted := make([]string, len(fallback))
	for i, s := range fallback {
		quoted[i] = strconv.Quote(s)
	}

	return strings.Join(quoted, ", ")
}
