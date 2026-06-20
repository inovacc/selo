package codegen

import (
	"fmt"
	"strings"

	"github.com/inovacc/selo"
)

// emit_python_test_render.go renders the per-kind pytest test that loads the
// golden vector and asserts validate/format/origin behaviour against the emitted
// module (mirrors emit_ts_test_render.go).

// renderTest emits tests/test_<kind>.py driven by vectors/<kind>.json.
func (e pythonEmitter) renderTest(kind selo.Kind) string {
	name := pythonName(kind)
	validateFn := "validate_" + name
	formatFn := "format_" + name
	generateFn := "generate_" + name
	hasOrigin := pythonHasOrigin(kind)

	imports := []string{validateFn, formatFn, generateFn}
	if hasOrigin {
		imports = append(imports, "origin_"+name)
	}

	var b strings.Builder
	b.WriteString(pythonHeaderComment())
	b.WriteString("\n")
	b.WriteString("import json\n")
	b.WriteString("import os\n\n")
	b.WriteString("import pytest\n\n")
	fmt.Fprintf(&b, "from selo.%s import (\n", kind.String())

	for _, imp := range imports {
		fmt.Fprintf(&b, "    %s,\n", imp)
	}

	b.WriteString(")\n\n")

	fmt.Fprintf(&b, "_VECTOR_PATH = os.path.join(os.path.dirname(__file__), \"..\", \"vectors\", %q)\n", kind.String()+".json")
	b.WriteString("with open(_VECTOR_PATH, encoding=\"utf-8\") as _f:\n")
	b.WriteString("    VECTOR = json.load(_f)\n\n\n")

	// validate
	fmt.Fprintf(&b, "@pytest.mark.parametrize(\"case\", VECTOR[\"validate\"])\n")
	b.WriteString("def test_validate(case):\n")
	fmt.Fprintf(&b, "    assert %s(case[\"input\"]) == case[\"valid\"], f\"validate {case['input']!r}\"\n\n\n", validateFn)

	// format
	fmt.Fprintf(&b, "@pytest.mark.parametrize(\"case\", VECTOR[\"format\"])\n")
	b.WriteString("def test_format(case):\n")
	b.WriteString("    if \"error\" in case:\n")
	b.WriteString("        with pytest.raises(ValueError):\n")
	fmt.Fprintf(&b, "            %s(case[\"input\"])\n", formatFn)
	b.WriteString("    else:\n")
	fmt.Fprintf(&b, "        assert %s(case[\"input\"]) == case[\"output\"], f\"format {case['input']!r}\"\n\n\n", formatFn)

	// origin
	if hasOrigin {
		fmt.Fprintf(&b, "@pytest.mark.parametrize(\"case\", VECTOR.get(\"origin\", []))\n")
		b.WriteString("def test_origin(case):\n")
		fmt.Fprintf(&b, "    assert origin_%s(case[\"input\"]) == case[\"output\"], f\"origin {case['input']!r}\"\n\n\n", name)
	}

	// generate round-trip
	b.WriteString("def test_generate_round_trip():\n")
	b.WriteString("    for _ in range(100):\n")
	fmt.Fprintf(&b, "        val = %s()\n", generateFn)
	fmt.Fprintf(&b, "        assert %s(val), f\"generate produced invalid: {val!r}\"\n", validateFn)

	return b.String()
}
