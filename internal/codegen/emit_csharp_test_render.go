package codegen

import (
	"fmt"
	"strings"

	"github.com/inovacc/selo"
)

// emit_csharp_test_render.go renders the per-kind xUnit test that loads the
// golden vector and asserts validate/format/origin behaviour against the emitted
// class. Mirrors emit_ts_test_render.go.

// csHasOrigin reports whether kind has an Origin method (mirrors originFnName).
func csHasOrigin(kind selo.Kind) bool {
	switch kind { //nolint:exhaustive // only origin-capable kinds return true; all others fall through to false
	case selo.KindCPF, selo.KindCEP, selo.KindPhone, selo.KindVoterID:
		return true
	default:
		return false
	}
}

// renderTest emits src/Selo.Tests/<Class>Tests.cs driven by vectors/<kind>.json.
func (e csharpEmitter) renderTest(kind selo.Kind, _ KindPlan, _ Vector) string {
	class := csName(kind)
	hasOrigin := csHasOrigin(kind)

	var b strings.Builder
	b.WriteString(csHeaderComment())
	b.WriteString("\n")
	b.WriteString("using System;\n")
	b.WriteString("using System.Collections.Generic;\n")
	b.WriteString("using Inovacc.Selo;\n")
	b.WriteString("using Xunit;\n\n")
	b.WriteString("namespace Inovacc.Selo.Tests\n{\n")
	fmt.Fprintf(&b, "    public class %sTests\n    {\n", class)
	fmt.Fprintf(&b, "        private static readonly Vector V = VectorLoader.Load(%q);\n\n", kind.String())

	// validate
	b.WriteString("        public static IEnumerable<object[]> ValidateCases()\n        {\n")
	b.WriteString("            foreach (var c in V.Validate)\n            {\n")
	b.WriteString("                yield return new object[] { c.Input, c.Valid };\n")
	b.WriteString("            }\n        }\n\n")
	b.WriteString("        [Theory]\n")
	b.WriteString("        [MemberData(nameof(ValidateCases))]\n")
	b.WriteString("        public void ValidateMatchesVector(string input, bool valid)\n        {\n")
	fmt.Fprintf(&b, "            Assert.Equal(valid, %s.Validate(input));\n", class)
	b.WriteString("        }\n\n")

	// format
	b.WriteString("        public static IEnumerable<object[]> FormatCases()\n        {\n")
	b.WriteString("            foreach (var c in V.Format)\n            {\n")
	b.WriteString("                yield return new object[] { c.Input, c.Output, c.Error };\n")
	b.WriteString("            }\n        }\n\n")
	b.WriteString("        [Theory]\n")
	b.WriteString("        [MemberData(nameof(FormatCases))]\n")
	b.WriteString("        public void FormatMatchesVector(string input, string? output, string? error)\n        {\n")
	b.WriteString("            if (error != null)\n            {\n")
	fmt.Fprintf(&b, "                Assert.Throws<FormatException>(() => %s.Format(input));\n", class)
	b.WriteString("            }\n            else\n            {\n")
	fmt.Fprintf(&b, "                Assert.Equal(output, %s.Format(input));\n", class)
	b.WriteString("            }\n        }\n")

	// origin
	if hasOrigin {
		b.WriteString("\n        public static IEnumerable<object[]> OriginCases()\n        {\n")
		b.WriteString("            foreach (var c in V.Origin ?? new List<OriginCase>())\n            {\n")
		b.WriteString("                yield return new object[] { c.Input, c.Output };\n")
		b.WriteString("            }\n        }\n\n")
		b.WriteString("        [Theory]\n")
		b.WriteString("        [MemberData(nameof(OriginCases))]\n")
		b.WriteString("        public void OriginMatchesVector(string input, string output)\n        {\n")
		fmt.Fprintf(&b, "            Assert.Equal(output, %s.Origin(input));\n", class)
		b.WriteString("        }\n")
	}

	b.WriteString("    }\n}\n")

	return b.String()
}
