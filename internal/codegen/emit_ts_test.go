package codegen_test

import (
	"strings"
	"testing"

	"github.com/inovacc/selo"
	"github.com/inovacc/selo/internal/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tsFiles emits the TS file set for kind and returns a path->content map.
func tsFiles(t *testing.T, kind selo.Kind) map[string]string {
	t.Helper()
	files, err := codegen.Generate(codegen.LangTS, kind)
	require.NoErrorf(t, err, "Generate(ts, %q)", kind)
	out := make(map[string]string, len(files))
	for _, f := range files {
		out[f.Path] = string(f.Content)
	}
	return out
}

// TestEmitTS_CPFModule asserts the CPF module exports the validator and is
// non-trivially populated.
func TestEmitTS_CPFModule(t *testing.T) {
	files := tsFiles(t, selo.KindCPF)
	mod := files["src/cpf.ts"]
	require.NotEmpty(t, mod, "missing src/cpf.ts")
	assert.Contains(t, mod, "export function validateCPF")
	assert.Contains(t, mod, "export function formatCPF")
	assert.Contains(t, mod, "export function originCPF")
}

// TestEmitTS_AllKindsNonEmpty asserts every kind emits a non-empty module, test,
// and vector, and that the validator is exported.
func TestEmitTS_AllKindsNonEmpty(t *testing.T) {
	require.Len(t, selo.Kinds(), 13)
	for _, k := range selo.Kinds() {
		files := tsFiles(t, k)
		mod := files["src/"+k.String()+".ts"]
		test := files["test/"+k.String()+".test.ts"]
		vec := files["vectors/"+k.String()+".json"]
		assert.NotEmptyf(t, mod, "kind %q: empty module", k)
		assert.NotEmptyf(t, test, "kind %q: empty test", k)
		assert.NotEmptyf(t, vec, "kind %q: empty vector", k)
		assert.Containsf(t, mod, "export function validate", "kind %q: no exported validator", k)
	}
}

// TestEmitTS_SharedFiles asserts the shared scaffolding is emitted with every
// kind (idempotent), so a single-kind generation still yields a runnable tree.
func TestEmitTS_SharedFiles(t *testing.T) {
	files := tsFiles(t, selo.KindCPF)
	for _, p := range []string{
		"src/mod11.ts", "src/data.ts", "src/index.ts",
		"package.json", "tsconfig.json", "vitest.config.ts",
	} {
		assert.NotEmptyf(t, files[p], "missing shared file %q", p)
	}
	assert.Contains(t, files["src/mod11.ts"], "export function computeDigit")
	assert.Contains(t, files["src/data.ts"], "CEP_RANGES")
	assert.Contains(t, files["src/index.ts"], "export * from \"./cpf.js\"")
}

// TestEmitTS_IrregularKindsBespoke asserts the irregular kinds carry their
// bespoke fragments (coupled CNH DVs, voter dual DV, plate/pix regex).
func TestEmitTS_IrregularKindsBespoke(t *testing.T) {
	assert.Contains(t, tsFiles(t, selo.KindCNH)["src/cnh.ts"], "cnhCheckDigits")
	assert.Contains(t, tsFiles(t, selo.KindVoterID)["src/voter_id.ts"], "voterDV2")
	assert.Contains(t, tsFiles(t, selo.KindPlate)["src/plate.ts"], "MERCOSUL")
	assert.Contains(t, tsFiles(t, selo.KindPIX)["src/pix.ts"], "detectPIXKind")
	assert.Contains(t, tsFiles(t, selo.KindCNPJ)["src/cnpj.ts"], "cnpjClean")
}

// TestEmitTS_FileSetShape asserts a full --kind all emission yields the expected
// per-kind and scaffold file counts.
func TestEmitTS_FileSetShape(t *testing.T) {
	modules := map[string]bool{}
	tests := map[string]bool{}
	vectors := map[string]bool{}
	for _, k := range selo.Kinds() {
		files := tsFiles(t, k)
		for p := range files {
			switch {
			case strings.HasPrefix(p, "src/") && strings.HasSuffix(p, ".ts") &&
				p != "src/mod11.ts" && p != "src/data.ts" && p != "src/index.ts":
				modules[p] = true
			case strings.HasPrefix(p, "test/"):
				tests[p] = true
			case strings.HasPrefix(p, "vectors/"):
				vectors[p] = true
			}
		}
	}
	assert.Len(t, modules, 13, "expected 13 kind modules")
	assert.Len(t, tests, 13, "expected 13 kind tests")
	assert.Len(t, vectors, 13, "expected 13 vector files")
}
