package codegen

import (
	"fmt"
	"strings"

	"github.com/inovacc/selo"
)

// emit_rust_test_render.go renders the per-kind inline #[cfg(test)] block that
// loads the golden vector and asserts validate/format/origin behaviour against
// the emitted module (mirrors emit_php_test_render.go). The block is appended to
// the module source by renderModule rather than written as its own File — Rust
// idiomatically colocates unit tests with the module under test.

// renderTest emits the `#[cfg(test)] mod tests { … }` block for kind, driven by
// vectors/<kind>.json read at test time via serde_json.
func (e rustEmitter) renderTest(kind selo.Kind) string {
	name := rustName(kind)
	validateFn := "validate_" + name
	formatFn := "format_" + name
	originFn := "origin_" + name
	generateFn := "generate_" + name
	hasOrigin := rustHasOrigin(kind)

	var b strings.Builder
	b.WriteString("#[cfg(test)]\n")
	b.WriteString("mod tests {\n")
	b.WriteString("    use super::*;\n\n")

	b.WriteString("    fn vector() -> serde_json::Value {\n")
	fmt.Fprintf(&b, "        let p = concat!(env!(\"CARGO_MANIFEST_DIR\"), \"/vectors/%s.json\");\n", kind.String())
	b.WriteString("        let s = std::fs::read_to_string(p).expect(\"read vector\");\n")
	b.WriteString("        serde_json::from_str(&s).expect(\"parse vector\")\n")
	b.WriteString("    }\n\n")

	// validate cases
	b.WriteString("    #[test]\n")
	b.WriteString("    fn validate_cases() {\n")
	b.WriteString("        for case in vector()[\"validate\"].as_array().unwrap() {\n")
	b.WriteString("            let input = case[\"input\"].as_str().unwrap();\n")
	b.WriteString("            let valid = case[\"valid\"].as_bool().unwrap();\n")
	fmt.Fprintf(&b, "            assert_eq!(valid, %s(input), \"validate {}\", input);\n", validateFn)
	b.WriteString("        }\n")
	b.WriteString("    }\n\n")

	// format cases
	b.WriteString("    #[test]\n")
	b.WriteString("    fn format_cases() {\n")
	b.WriteString("        for case in vector()[\"format\"].as_array().unwrap() {\n")
	b.WriteString("            let input = case[\"input\"].as_str().unwrap();\n")
	b.WriteString("            let is_error = case\n")
	b.WriteString("                .get(\"error\")\n")
	b.WriteString("                .and_then(|e| e.as_str())\n")
	b.WriteString("                .map(|e| !e.is_empty())\n")
	b.WriteString("                .unwrap_or(false);\n")
	b.WriteString("            if is_error {\n")
	fmt.Fprintf(&b, "                assert!(%s(input).is_err(), \"format expected err {}\", input);\n", formatFn)
	b.WriteString("            } else {\n")
	b.WriteString("                let want = case[\"output\"].as_str().unwrap();\n")
	fmt.Fprintf(&b, "                assert_eq!(want, %s(input).unwrap(), \"format {}\", input);\n", formatFn)
	b.WriteString("            }\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n\n")

	// origin cases (only for origin-capable kinds)
	if hasOrigin {
		b.WriteString("    #[test]\n")
		b.WriteString("    fn origin_cases() {\n")
		b.WriteString("        if let Some(arr) = vector().get(\"origin\").and_then(|o| o.as_array()) {\n")
		b.WriteString("            for case in arr {\n")
		b.WriteString("                let input = case[\"input\"].as_str().unwrap();\n")
		b.WriteString("                let want = case[\"output\"].as_str().unwrap();\n")
		fmt.Fprintf(&b, "                assert_eq!(want, %s(input).unwrap(), \"origin {}\", input);\n", originFn)
		b.WriteString("            }\n")
		b.WriteString("        }\n")
		b.WriteString("    }\n\n")
	}

	// generate round-trip
	b.WriteString("    #[test]\n")
	b.WriteString("    fn generate_round_trip() {\n")
	b.WriteString("        for _ in 0..100 {\n")
	fmt.Fprintf(&b, "            let v = %s();\n", generateFn)
	fmt.Fprintf(&b, "            assert!(%s(&v), \"generate produced invalid: {}\", v);\n", validateFn)
	b.WriteString("        }\n")
	b.WriteString("    }\n")

	b.WriteString("}\n")

	return b.String()
}
