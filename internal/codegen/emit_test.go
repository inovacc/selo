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

// TestEmitterRegistry_TSRegistered asserts the TypeScript emitter is registered
// (M2) and the Ruby emitter is registered (M4), while the remaining languages
// stay unregistered until their milestones.
func TestEmitterRegistry_TSRegistered(t *testing.T) {
	ts, ok := codegen.EmitterFor(codegen.LangTS)
	require.True(t, ok, "TypeScript emitter should be registered in M2")
	assert.Equal(t, codegen.LangTS, ts.Lang())

	ruby, ok := codegen.EmitterFor(codegen.LangRuby)
	require.True(t, ok, "Ruby emitter should be registered in M4")
	assert.Equal(t, codegen.LangRuby, ruby.Lang())

	for _, l := range []codegen.Lang{codegen.LangJS, codegen.LangJava, codegen.LangCSharp} {
		_, regd := codegen.EmitterFor(l)
		assert.Falsef(t, regd, "no emitter should be registered for %q yet", l)

		_, err := codegen.Generate(l, selo.KindCPF)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not yet registered")
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
