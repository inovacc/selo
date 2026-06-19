package codegen_test

import (
	"testing"

	"github.com/inovacc/selo"
	"github.com/inovacc/selo/internal/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSupportedLangs asserts the fixed five-language target set and its string
// form, which the CLI/MCP advertise.
func TestSupportedLangs(t *testing.T) {
	assert.Equal(t,
		[]string{"ts", "js", "ruby", "java", "csharp"},
		codegen.SupportedLangStrings())

	for _, l := range []string{"ts", "js", "ruby", "java", "csharp"} {
		assert.Truef(t, codegen.IsSupportedLang(l), "expected %q supported", l)
	}
	assert.False(t, codegen.IsSupportedLang("python"))
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

// TestEmitterRegistry_EmptyInM1 asserts no emitter is registered yet, so
// Generate fails cleanly for a supported language.
func TestEmitterRegistry_EmptyInM1(t *testing.T) {
	for _, l := range codegen.SupportedLangs() {
		_, ok := codegen.EmitterFor(l)
		assert.Falsef(t, ok, "no emitter should be registered for %q in M1", l)
	}

	_, err := codegen.Generate(codegen.LangTS, selo.KindCPF)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet registered")
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
