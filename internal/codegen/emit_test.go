package codegen_test

import (
	"testing"

	"github.com/inovacc/selo"
	"github.com/inovacc/selo/internal/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSupportedLangs asserts the fixed six-language target set and its string
// form, which the CLI/MCP advertise.
func TestSupportedLangs(t *testing.T) {
	assert.Equal(t,
		[]string{"ts", "js", "ruby", "java", "csharp", "python"},
		codegen.SupportedLangStrings())

	for _, l := range []string{"ts", "js", "ruby", "java", "csharp", "python"} {
		assert.Truef(t, codegen.IsSupportedLang(l), "expected %q supported", l)
	}

	assert.False(t, codegen.IsSupportedLang("rust"))
	assert.False(t, codegen.IsSupportedLang(""))
}

// TestKindStrings asserts KindStrings covers all 13 kinds in sorted order.
func TestKindStrings(t *testing.T) {
	ks := codegen.KindStrings()
	assert.Len(t, ks, 13)

	for _, k := range selo.Kinds() {
		assert.Containsf(t, ks, k.String(), "KindStrings missing %q", k)
	}
	// sorted
	for i := 1; i < len(ks); i++ {
		assert.LessOrEqual(t, ks[i-1], ks[i])
	}
}

// TestEmitterRegistry_AllRegistered asserts every target language's emitter is
// registered (M2–M6: ts, js, ruby, java, csharp, python).
func TestEmitterRegistry_AllRegistered(t *testing.T) {
	for _, l := range []codegen.Lang{
		codegen.LangTS, codegen.LangJS, codegen.LangRuby, codegen.LangJava, codegen.LangCSharp, codegen.LangPython,
	} {
		e, ok := codegen.EmitterFor(l)
		require.Truef(t, ok, "emitter should be registered for %q", l)
		assert.Equalf(t, l, e.Lang(), "EmitterFor(%q).Lang() mismatch", l)
	}
}

// TestGenerate_UnsupportedLang asserts an unknown language is a clean error.
func TestGenerate_UnsupportedLang(t *testing.T) {
	_, err := codegen.Generate(codegen.Lang("bogus"), selo.KindCPF)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported language")
}

// TestGenerate_UnknownKind asserts an unknown kind is a clean error.
func TestGenerate_UnknownKind(t *testing.T) {
	_, err := codegen.Generate(codegen.LangTS, selo.Kind("nope"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown kind")
}
