package codegen

import (
	"fmt"
	"strings"

	"github.com/inovacc/selo"
)

// emit_ts_test_render.go renders the per-kind vitest test that loads the golden
// vector and asserts validate/format/origin behaviour against the emitted module.

// originFnName returns the TS origin function name for kind, or "" when the kind
// has no origin resolution.
func originFnName(kind selo.Kind) string {
	switch kind { //nolint:exhaustive // only origin-capable kinds return a non-empty name; all others fall through to ""
	case selo.KindCPF, selo.KindCEP, selo.KindPhone, selo.KindVoterID:
		return "origin" + tsName(kind)
	default:
		return ""
	}
}

// renderTest emits test/<kind>.test.ts driven by vectors/<kind>.json.
func (e tsEmitter) renderTest(kind selo.Kind, _ KindPlan, _ Vector) string {
	name := tsName(kind)
	validateFn := "validate" + name
	formatFn := "format" + name
	originFn := originFnName(kind)

	var imports []string

	imports = append(imports, validateFn, formatFn)
	if originFn != "" {
		imports = append(imports, originFn)
	}

	var b strings.Builder
	b.WriteString(headerComment())
	b.WriteString("\n")
	b.WriteString("import { describe, it, expect } from \"vitest\";\n")
	fmt.Fprintf(&b, "import { %s } from \"../src/%s.js\";\n", strings.Join(imports, ", "), kind.String())
	fmt.Fprintf(&b, "import vector from \"../vectors/%s.json\" assert { type: \"json\" };\n\n", kind.String())

	b.WriteString("interface ValidateCase { input: string; valid: boolean; uf?: string }\n")
	b.WriteString("interface FormatCase { input: string; output?: string; error?: string }\n")
	b.WriteString("interface OriginCase { input: string; output: string }\n")
	b.WriteString("interface Vector {\n")
	b.WriteString("  kind: string;\n")
	b.WriteString("  validate: ValidateCase[];\n")
	b.WriteString("  format: FormatCase[];\n")
	b.WriteString("  origin?: OriginCase[];\n")
	b.WriteString("}\n\n")
	b.WriteString("const v = vector as unknown as Vector;\n\n")

	fmt.Fprintf(&b, "describe(%q, () => {\n", kind.String())

	// validate
	b.WriteString("  describe(\"validate\", () => {\n")
	b.WriteString("    for (const c of v.validate) {\n")
	fmt.Fprintf(&b, "      it(`validate ${JSON.stringify(c.input)} -> ${c.valid}`, () => {\n")
	fmt.Fprintf(&b, "        expect(%s(c.input)).toBe(c.valid);\n", validateFn)
	b.WriteString("      });\n")
	b.WriteString("    }\n")
	b.WriteString("  });\n\n")

	// format
	b.WriteString("  describe(\"format\", () => {\n")
	b.WriteString("    for (const c of v.format) {\n")
	fmt.Fprintf(&b, "      it(`format ${JSON.stringify(c.input)}`, () => {\n")
	b.WriteString("        if (c.error !== undefined) {\n")
	fmt.Fprintf(&b, "          expect(() => %s(c.input)).toThrow();\n", formatFn)
	b.WriteString("        } else {\n")
	fmt.Fprintf(&b, "          expect(%s(c.input)).toBe(c.output);\n", formatFn)
	b.WriteString("        }\n")
	b.WriteString("      });\n")
	b.WriteString("    }\n")
	b.WriteString("  });\n")

	// origin
	if originFn != "" {
		b.WriteString("\n  describe(\"origin\", () => {\n")
		b.WriteString("    for (const c of v.origin ?? []) {\n")
		fmt.Fprintf(&b, "      it(`origin ${JSON.stringify(c.input)} -> ${c.output}`, () => {\n")
		fmt.Fprintf(&b, "        expect(%s(c.input)).toBe(c.output);\n", originFn)
		b.WriteString("      });\n")
		b.WriteString("    }\n")
		b.WriteString("  });\n")
	}

	b.WriteString("});\n")

	return b.String()
}
