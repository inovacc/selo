package codegen

import (
	"fmt"
	"strings"

	"github.com/inovacc/selo"
)

// emit_java_test_render.go renders the per-kind JUnit 5 + Jackson test that loads
// the golden vector from the classpath and asserts validate/format/origin
// behaviour against the emitted class. It mirrors the M2 TypeScript reference
// (emit_ts_test_render.go).

// javaHasOrigin reports whether kind exposes an origin() method.
func javaHasOrigin(kind selo.Kind) bool {
	switch kind { //nolint:exhaustive // only origin-capable kinds return true; all others fall through to false
	case selo.KindCPF, selo.KindCEP, selo.KindPhone, selo.KindVoterID:
		return true
	default:
		return false
	}
}

// renderTest emits src/test/java/com/inovacc/selo/<Kind>Test.java driven by
// vectors/<kind>.json (loaded from the classpath via the test resource mapping).
func (e javaEmitter) renderTest(kind selo.Kind, plan KindPlan, _ Vector) string {
	name := javaName(kind)
	className := name + "Test"

	var b strings.Builder
	b.WriteString(javaHeaderComment())
	b.WriteString("package com.inovacc.selo;\n\n")
	b.WriteString("import static org.junit.jupiter.api.Assertions.assertEquals;\n")
	b.WriteString("import static org.junit.jupiter.api.Assertions.assertThrows;\n\n")
	b.WriteString("import com.fasterxml.jackson.databind.JsonNode;\n")
	b.WriteString("import com.fasterxml.jackson.databind.ObjectMapper;\n")
	b.WriteString("import java.io.InputStream;\n")
	b.WriteString("import org.junit.jupiter.api.Test;\n\n")

	fmt.Fprintf(&b, "/** %s exercises %s against its golden vector. */\n", className, name)
	fmt.Fprintf(&b, "class %s {\n", className)

	fmt.Fprintf(&b, `    /** vector loads the committed golden vector for %[1]s from the classpath. */
    private static JsonNode vector() throws Exception {
        ObjectMapper mapper = new ObjectMapper();
        try (InputStream in = %[2]s.class.getResourceAsStream("/vectors/%[3]s.json")) {
            if (in == null) {
                throw new IllegalStateException("missing vector resource: /vectors/%[3]s.json");
            }
            return mapper.readTree(in);
        }
    }

`, name, className, kind.String())

	// validate
	b.WriteString("    @Test\n")
	b.WriteString("    void validate() throws Exception {\n")
	b.WriteString("        for (JsonNode c : vector().get(\"validate\")) {\n")
	b.WriteString("            String input = c.get(\"input\").asText();\n")
	b.WriteString("            boolean want = c.get(\"valid\").asBoolean();\n")

	if plan.UFScoped {
		fmt.Fprintf(&b, `            boolean got;
            if (c.hasNonNull("uf")) {
                got = %s.validateForUF(input, c.get("uf").asText());
            } else {
                got = %s.validate(input);
            }
            assertEquals(want, got, "validate " + input);
`, name, name)
	} else {
		fmt.Fprintf(&b, "            assertEquals(want, %s.validate(input), \"validate \" + input);\n", name)
	}

	b.WriteString("        }\n")
	b.WriteString("    }\n\n")

	// format
	b.WriteString("    @Test\n")
	b.WriteString("    void format() throws Exception {\n")
	b.WriteString("        for (JsonNode c : vector().get(\"format\")) {\n")
	b.WriteString("            String input = c.get(\"input\").asText();\n")
	b.WriteString("            if (c.hasNonNull(\"error\")) {\n")
	fmt.Fprintf(&b, "                assertThrows(RuntimeException.class, () -> %s.format(input), \"format \" + input);\n", name)
	b.WriteString("            } else {\n")
	fmt.Fprintf(&b, "                assertEquals(c.get(\"output\").asText(), %s.format(input), \"format \" + input);\n", name)
	b.WriteString("            }\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n")

	// origin
	if javaHasOrigin(kind) {
		b.WriteString("\n    @Test\n")
		b.WriteString("    void origin() throws Exception {\n")
		b.WriteString("        JsonNode origin = vector().get(\"origin\");\n")
		b.WriteString("        if (origin == null) {\n")
		b.WriteString("            return;\n")
		b.WriteString("        }\n")
		b.WriteString("        for (JsonNode c : origin) {\n")
		b.WriteString("            String input = c.get(\"input\").asText();\n")
		fmt.Fprintf(&b, "            assertEquals(c.get(\"output\").asText(), %s.origin(input), \"origin \" + input);\n", name)
		b.WriteString("        }\n")
		b.WriteString("    }\n")
	}

	b.WriteString("}\n")

	return b.String()
}
