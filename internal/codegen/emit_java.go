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

// emit_java.go is the Java language emitter (plan M5; mirrors the M2 TypeScript
// reference emit_ts.go). It renders, for one kind, an idiomatic Java class
// (static validate/format/origin and, for RG/IE, UF-param variants) under the
// package com.inovacc.selo, a JUnit 5 + Jackson test driven by the golden
// vector, the vector JSON itself, and the shared scaffolding (mod-11 reducer,
// data tables, pom.xml). The check-digit kinds reuse a single shared Mod11
// reducer parameterized by each kind's CheckDigit spec.
//
// The Java SOURCE is rendered purely from the declarative KindPlan and the
// static templates, so it is deterministic and snapshot-stable. Only the vector
// JSON is non-deterministic (it mixes selo.Generate output); the golden snapshot
// test compares the deterministic source files and validates the vectors against
// selo rather than byte-comparing them.

//go:embed templates/java/Mod11.java.tmpl templates/java/pom.xml.tmpl
var javaTemplates embed.FS

func init() { Register(javaEmitter{}) }

// javaPackagePath is the source directory for the com.inovacc.selo package.
const javaPackagePath = "src/main/java/com/inovacc/selo/"

// javaEmitter implements Emitter for Java.
type javaEmitter struct{}

// Lang reports the target language.
func (javaEmitter) Lang() Lang { return LangJava }

// Emit renders the full Java file set for kind: the per-kind class, its test, the
// vector JSON, and the shared scaffolding (idempotent across kinds).
func (e javaEmitter) Emit(kind selo.Kind, plan KindPlan, vec Vector) ([]File, error) {
	module, err := e.renderModule(kind, plan)
	if err != nil {
		return nil, err
	}

	test := e.renderTest(kind, plan, vec)

	vectorJSON, err := json.MarshalIndent(vec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("codegen/java: marshal %q vector: %w", kind, err)
	}

	vectorJSON = append(vectorJSON, '\n')

	shared, err := e.sharedFiles()
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, 3+len(shared))
	files = append(files,
		File{Path: javaPackagePath + javaName(kind) + ".java", Content: []byte(module)},
		File{Path: "src/test/java/com/inovacc/selo/" + javaName(kind) + "Test.java", Content: []byte(test)},
		File{Path: "vectors/" + kind.String() + ".json", Content: vectorJSON},
	)
	files = append(files, shared...)

	return files, nil
}

// sharedFiles returns the language-wide files that do not depend on a single
// kind: the mod-11 reducer, the embedded data tables, and the build scaffolding
// (pom.xml). They are emitted on every per-kind call and are byte-identical, so
// re-writing them is idempotent.
func (e javaEmitter) sharedFiles() ([]File, error) {
	mod11, err := javaTemplates.ReadFile("templates/java/Mod11.java.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/java: read Mod11 template: %w", err)
	}

	pom, err := javaTemplates.ReadFile("templates/java/pom.xml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("codegen/java: read pom.xml template: %w", err)
	}

	return []File{
		{Path: javaPackagePath + "Mod11.java", Content: mod11},
		{Path: javaPackagePath + "Data.java", Content: []byte(e.renderData())},
		{Path: "pom.xml", Content: pom},
	}, nil
}

// renderData emits the embedded data tables (CEP ranges, DDD->UF, CPF region
// map, voter UF names) as Java static collections, from the codegen accessors.
func (e javaEmitter) renderData() string {
	var b strings.Builder
	b.WriteString(javaHeaderComment())
	b.WriteString("package com.inovacc.selo;\n\n")
	b.WriteString("import java.util.List;\n")
	b.WriteString("import java.util.Map;\n\n")
	b.WriteString("/** Data holds the embedded geographic lookup tables (CEP, DDD, CPF, voter). */\n")
	b.WriteString("public final class Data {\n")
	b.WriteString("    private Data() {\n")
	b.WriteString("    }\n\n")

	// UFRange nested record.
	b.WriteString("    /** UFRange is one CEP prefix range (inclusive) mapped to a UF. */\n")
	b.WriteString("    public record UFRange(String uf, int from, int to) {\n")
	b.WriteString("    }\n\n")

	// CEP prefix ranges (scan order; first match wins).
	b.WriteString("    /** CEP_RANGES is the CEP prefix->UF allocation table in scan order. */\n")
	b.WriteString("    public static final List<UFRange> CEP_RANGES = List.of(\n")

	cepRanges := CEPRanges()
	for i, r := range cepRanges {
		sep := ","
		if i == len(cepRanges)-1 {
			sep = ""
		}

		fmt.Fprintf(&b, "        new UFRange(%s, %d, %d)%s\n", strconv.Quote(r.UF), r.From, r.To, sep)
	}

	b.WriteString("    );\n\n")

	// DDD -> UF.
	b.WriteString("    /** DDD_TO_UF maps a two-digit area code to its UF. */\n")
	b.WriteString("    public static final Map<String, String> DDD_TO_UF = Map.ofEntries(\n")

	dddMap := DDDtoUF()

	ddds := make([]string, 0, len(dddMap))
	for d := range dddMap {
		ddds = append(ddds, d)
	}

	sort.Strings(ddds)

	for i, d := range ddds {
		sep := ","
		if i == len(ddds)-1 {
			sep = ""
		}

		fmt.Fprintf(&b, "        Map.entry(%s, %s)%s\n", strconv.Quote(d), strconv.Quote(dddMap[d].String()), sep)
	}

	b.WriteString("    );\n\n")

	// CPF ninth-digit region map.
	b.WriteString("    /** CPF_REGIONS maps the CPF ninth digit to its issuing region. */\n")
	b.WriteString("    public static final Map<Integer, String> CPF_REGIONS = Map.ofEntries(\n")

	cpfRegions := CPFRegions()

	cpfKeys := make([]int, 0, len(cpfRegions))
	for i := 0; i <= 9; i++ {
		if _, ok := cpfRegions[i]; ok {
			cpfKeys = append(cpfKeys, i)
		}
	}

	for i, k := range cpfKeys {
		sep := ","
		if i == len(cpfKeys)-1 {
			sep = ""
		}

		fmt.Fprintf(&b, "        Map.entry(%d, %s)%s\n", k, strconv.Quote(cpfRegions[k]), sep)
	}

	b.WriteString("    );\n\n")

	// Voter-ID UF code -> region name.
	b.WriteString("    /** VOTER_UF_NAMES maps a voter-ID UF code (1..28) to its region. */\n")
	b.WriteString("    public static final Map<Integer, String> VOTER_UF_NAMES = Map.ofEntries(\n")

	voterNames := VoterUFNames()

	codes := make([]int, 0, len(voterNames))
	for c := range voterNames {
		codes = append(codes, c)
	}

	sort.Ints(codes)

	for i, c := range codes {
		sep := ","
		if i == len(codes)-1 {
			sep = ""
		}

		fmt.Fprintf(&b, "        Map.entry(%d, %s)%s\n", c, strconv.Quote(voterNames[c]), sep)
	}

	b.WriteString("    );\n")
	b.WriteString("}\n")

	return b.String()
}

// renderModule dispatches to the per-group renderer for kind.
func (e javaEmitter) renderModule(kind selo.Kind, plan KindPlan) (string, error) {
	switch kind {
	case selo.KindCPF:
		return e.renderCPF(plan), nil
	case selo.KindPIS:
		return e.renderSimpleNumeric(plan, "Pis", 11), nil
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
		return "", fmt.Errorf("codegen/java: no module renderer for kind %q", kind)
	}
}

// --- shared rendering helpers -----------------------------------------------

// javaHeaderComment is the generated-file banner.
func javaHeaderComment() string {
	return "// Code generated by selo gen --lang java. DO NOT EDIT.\n"
}

// javaName returns the PascalCase class-name suffix used for each kind's file
// and exported method names, e.g. "cpf" -> "CPF", "voter_id" -> "VoterId".
func javaName(kind selo.Kind) string {
	switch kind {
	case selo.KindCPF:
		return "CPF"
	case selo.KindCNPJ:
		return "CNPJ"
	case selo.KindCNH:
		return "CNH"
	case selo.KindPIS:
		return "Pis"
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

// intArrayLiteral renders a Go int slice as a Java new int[]{...} literal.
func intArrayLiteral(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}

	return "new int[]{" + strings.Join(parts, ", ") + "}"
}

// checkDigitNew renders a CheckDigit as a Java Mod11.CheckDigit constructor call
// matching the (weights, rule, remainderTo0, multiplyBy10, encodeXAt,
// encodeZeroAt) signature.
func checkDigitNew(cd CheckDigit) string {
	weights := intArrayLiteral(cd.Weights)
	rule := "Mod11.DVRule." + javaRuleEnum(cd.Rule)

	remainder := "null"
	if len(cd.RemainderTo0) > 0 {
		remainder = intArrayLiteral(cd.RemainderTo0)
	}

	multiply := "false"
	if cd.MultiplyBy10 {
		multiply = "true"
	}

	return fmt.Sprintf("new Mod11.CheckDigit(%s, %s, %s, %s, %d, %d)",
		weights, rule, remainder, multiply, cd.EncodeXAt, cd.EncodeZeroAt)
}

// javaRuleEnum maps a DVRule's stable string to its Java enum constant name.
func javaRuleEnum(r DVRule) string {
	return strings.ToUpper(r.String())
}

// javaThrow emits the standard Java throw for a format/origin error, mapping a
// sentinel name to a thrown IllegalArgumentException whose message is the
// sentinel.
func javaThrow(sentinel string) string {
	return "throw new IllegalArgumentException(" + strconv.Quote(sentinel) + ");"
}

// javaStringArray renders a list of strings as a Java new String[]{...} literal.
func javaStringArray(ss []string) string {
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = strconv.Quote(s)
	}

	return "new String[]{" + strings.Join(quoted, ", ") + "}"
}

// javaMaskExpr converts a '#'/'X'-placeholder mask (e.g. "###.#####.##-#") into
// a Java string-concatenation expression slicing the cleaned digit variable v
// via v.substring(start, end), e.g.
// v.substring(0, 3) + "." + v.substring(3, 8) + ...
func javaMaskExpr(mask, v string) string {
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

			parts = append(parts, fmt.Sprintf("%s.substring(%d, %d)", v, start, pos))

			continue
		}
		// literal separator: accumulate consecutive separators into one literal.
		var lit strings.Builder

		for i < len(mask) && mask[i] != '#' && mask[i] != 'X' {
			lit.WriteByte(mask[i])
			i++
		}

		parts = append(parts, strconv.Quote(lit.String()))
	}

	return strings.Join(parts, " + ")
}
