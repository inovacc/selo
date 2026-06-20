package codegen

import (
	"fmt"
	"strings"

	"github.com/inovacc/selo"
)

// emit_php_test_render.go renders the per-kind PHPUnit test that loads the golden
// vector and asserts validate/format/origin behaviour against the emitted class
// (mirrors emit_python_test_render.go).

// renderTest emits tests/<Class>Test.php driven by vectors/<kind>.json.
func (e phpEmitter) renderTest(kind selo.Kind) string {
	class := phpClassName(kind)
	suffix := phpFnSuffix(kind)
	validateFn := "validate" + suffix
	formatFn := "format" + suffix
	originFn := "origin" + suffix
	generateFn := "generate" + suffix
	hasOrigin := phpHasOrigin(kind)

	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo\\Tests;\n\n")
	b.WriteString("use PHPUnit\\Framework\\TestCase;\n")
	b.WriteString("use PHPUnit\\Framework\\Attributes\\DataProvider;\n")
	fmt.Fprintf(&b, "use Selo\\%s;\n\n", class)

	fmt.Fprintf(&b, "final class %sTest extends TestCase\n{\n", class)

	// vector loader
	b.WriteString("    /** @return array<string, mixed> */\n")
	b.WriteString("    private static function vector(): array\n")
	b.WriteString("    {\n")
	fmt.Fprintf(&b, "        $path = __DIR__ . '/../vectors/%s.json';\n", kind.String())
	b.WriteString("        $data = json_decode((string) file_get_contents($path), true);\n")
	b.WriteString("        assert(is_array($data));\n")
	b.WriteString("        return $data;\n")
	b.WriteString("    }\n\n")

	// validate provider + test
	b.WriteString("    /** @return iterable<int, array{0: string, 1: bool}> */\n")
	b.WriteString("    public static function validateCases(): iterable\n")
	b.WriteString("    {\n")
	b.WriteString("        foreach (self::vector()['validate'] as $case) {\n")
	b.WriteString("            yield [(string) $case['input'], (bool) $case['valid']];\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n\n")
	b.WriteString("    #[DataProvider('validateCases')]\n")
	b.WriteString("    public function testValidate(string $input, bool $valid): void\n")
	b.WriteString("    {\n")
	fmt.Fprintf(&b, "        $this->assertSame($valid, %s::%s($input), \"validate \" . $input);\n", class, validateFn)
	b.WriteString("    }\n\n")

	// format provider + test
	b.WriteString("    /** @return iterable<int, array{0: string, 1: ?string, 2: bool}> */\n")
	b.WriteString("    public static function formatCases(): iterable\n")
	b.WriteString("    {\n")
	b.WriteString("        foreach (self::vector()['format'] as $case) {\n")
	b.WriteString("            $isError = isset($case['error']) && $case['error'] !== '';\n")
	b.WriteString("            yield [(string) $case['input'], $isError ? null : (string) $case['output'], $isError];\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n\n")
	b.WriteString("    #[DataProvider('formatCases')]\n")
	b.WriteString("    public function testFormat(string $input, ?string $output, bool $isError): void\n")
	b.WriteString("    {\n")
	b.WriteString("        if ($isError) {\n")
	b.WriteString("            $this->expectException(\\InvalidArgumentException::class);\n")
	fmt.Fprintf(&b, "            %s::%s($input);\n", class, formatFn)
	b.WriteString("            return;\n")
	b.WriteString("        }\n")
	fmt.Fprintf(&b, "        $this->assertSame($output, %s::%s($input), \"format \" . $input);\n", class, formatFn)
	b.WriteString("    }\n\n")

	// origin provider + test
	if hasOrigin {
		b.WriteString("    /** @return iterable<int, array{0: string, 1: string}> */\n")
		b.WriteString("    public static function originCases(): iterable\n")
		b.WriteString("    {\n")
		b.WriteString("        foreach (self::vector()['origin'] ?? [] as $case) {\n")
		b.WriteString("            yield [(string) $case['input'], (string) $case['output']];\n")
		b.WriteString("        }\n")
		b.WriteString("    }\n\n")
		b.WriteString("    #[DataProvider('originCases')]\n")
		b.WriteString("    public function testOrigin(string $input, string $output): void\n")
		b.WriteString("    {\n")
		fmt.Fprintf(&b, "        $this->assertSame($output, %s::%s($input), \"origin \" . $input);\n", class, originFn)
		b.WriteString("    }\n\n")
	}

	// generate round-trip
	b.WriteString("    public function testGenerateRoundTrip(): void\n")
	b.WriteString("    {\n")
	b.WriteString("        for ($i = 0; $i < 100; $i++) {\n")
	fmt.Fprintf(&b, "            $val = %s::%s();\n", class, generateFn)
	fmt.Fprintf(&b, "            $this->assertTrue(%s::%s($val), \"generate produced invalid: \" . $val);\n", class, validateFn)
	b.WriteString("        }\n")
	b.WriteString("    }\n")

	b.WriteString("}\n")

	return b.String()
}
