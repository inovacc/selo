package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/inovacc/selo"
	"github.com/inovacc/selo/internal/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenCmdRegistered asserts the gen command is wired into the root command.
func TestGenCmdRegistered(t *testing.T) {
	root := newRootCmd()
	assert.NotNil(t, findCmd(root, "gen"))
}

// TestGenHelpListsLangsAndKinds is the Task 4 gate: `gen --help` advertises the
// five supported languages and all 13 kinds.
func TestGenHelpListsLangsAndKinds(t *testing.T) {
	out, err := runCmd(t, "gen", "--help")
	require.NoError(t, err)

	for _, lang := range []string{"ts", "js", "ruby", "java", "csharp"} {
		assert.Containsf(t, out, lang, "gen --help should list lang %q", lang)
	}
	for _, k := range sdk.Kinds() {
		assert.Containsf(t, out, k.String(), "gen --help should list kind %q", k)
	}
}

// TestGenBogusLangExitsNonZero asserts an unsupported --lang errors (non-zero).
func TestGenBogusLangExitsNonZero(t *testing.T) {
	_, err := runCmd(t, "gen", "--lang", "bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// TestGenMissingLangExitsNonZero asserts --lang is required.
func TestGenMissingLangExitsNonZero(t *testing.T) {
	_, err := runCmd(t, "gen")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--lang is required")
}

// TestGenSupportedLangNoEmitter asserts that a supported language whose emitter
// is not registered yet (e.g. js, pending its milestone) fails cleanly rather
// than silently succeeding.
func TestGenSupportedLangNoEmitter(t *testing.T) {
	_, err := runCmd(t, "gen", "--lang", "js", "--kind", "cpf", "--out", t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet registered")
}

// TestGenTSWritesFiles asserts the registered TypeScript emitter (M2) writes the
// expected module/test/vector/scaffold files for a kind.
func TestGenTSWritesFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmd(t, "gen", "--lang", "ts", "--kind", "cpf", "--out", dir)
	require.NoError(t, err)
	for _, rel := range []string{
		"src/cpf.ts", "test/cpf.test.ts", "vectors/cpf.json",
		"src/mod11.ts", "package.json", "tsconfig.json", "vitest.config.ts",
	} {
		_, statErr := os.Stat(filepath.Join(dir, rel))
		assert.NoErrorf(t, statErr, "expected generated file %q", rel)
	}
}

// TestGenUnknownKindExitsNonZero asserts an unknown --kind errors.
func TestGenUnknownKindExitsNonZero(t *testing.T) {
	_, err := runCmd(t, "gen", "--lang", "ts", "--kind", "nope", "--out", t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown --kind")
}

// TestGenAllSupportedLangsKnown keeps the CLI's notion of supported languages in
// lockstep with the codegen package.
func TestGenAllSupportedLangsKnown(t *testing.T) {
	assert.Equal(t,
		[]string{"ts", "js", "ruby", "java", "csharp"},
		codegen.SupportedLangStrings())
}

// TestResolveKinds_All expands "all" and "" to every kind.
func TestResolveKinds_All(t *testing.T) {
	for _, in := range []string{"all", ""} {
		ks, err := resolveKinds(in)
		require.NoError(t, err)
		assert.Len(t, ks, len(sdk.Kinds()))
	}
}

// TestResolveKinds_Single resolves a single kind.
func TestResolveKinds_Single(t *testing.T) {
	ks, err := resolveKinds("cpf")
	require.NoError(t, err)
	require.Len(t, ks, 1)
	assert.Equal(t, sdk.KindCPF, ks[0])
}

// TestGenHelpHasAllString asserts the help mentions the 'all' shortcut.
func TestGenHelpHasAllString(t *testing.T) {
	out, err := runCmd(t, "gen", "--help")
	require.NoError(t, err)
	assert.True(t, strings.Contains(out, "all"), "help should mention 'all'")
}
