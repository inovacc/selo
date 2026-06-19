package codegen

import (
	"fmt"
	"strings"

	"github.com/inovacc/selo"
)

// emit_js_test_render.go renders the per-kind vitest test for JavaScript.

// renderTest emits test/<kind>.test.js driven by vectors/<kind>.json.
func (e jsEmitter) renderTest(kind selo.Kind, _ KindPlan, _ Vector) string {
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
	b.WriteString(jsHeaderComment())
	b.WriteString("\n")
	b.WriteString("import { describe, it, expect } from \"vitest\";\n")
	fmt.Fprintf(&b, "import { %s } from \"../src/%s.js\";\n", strings.Join(imports, ", "), kind.String())
	fmt.Fprintf(&b, "import vector from \"../vectors/%s.json\" assert { type: \"json\" };\n\n", kind.String())

	b.WriteString("const v = vector;\n\n")

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
