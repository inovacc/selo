package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBulkFromFileCPF writes a file containing a comment, blank line, one valid
// CPF and one invalid CPF, then asserts that:
//   - the command returns errInvalidInput (exit 1 contract),
//   - every non-skipped line is printed (valid first, then invalid).
func TestBulkFromFileCPF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cpfs.txt")
	content := strings.Join([]string{
		"# header comment",
		"",
		"529.982.247-25", // valid
		"123.456.789-00", // invalid
		"   ",            // blank
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content+"\n"), 0o600))

	out, err := runCmd(t, "cpf", "--from", path)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errInvalidInput))
	assert.Equal(t, "valid\t529.982.247-25\ninvalid\t123.456.789-00\n", out)
}

// TestBulkFromAllValidCNPJ passes a file with a single known-valid CNPJ and
// asserts exit 0 + formatted output.
func TestBulkFromAllValidCNPJ(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cnpjs.txt")
	// 39591842000010 formats to 39.591.842/0000-10 (matches TestKindCmdFormatCNPJ).
	require.NoError(t, os.WriteFile(path, []byte("39591842000010\n"), 0o600))

	out, err := runCmd(t, "cnpj", "--from", path)
	require.NoError(t, err)
	assert.Equal(t, "valid\t39.591.842/0000-10\n", out)
}

// TestBulkFromMissingFile asserts that a non-existent path returns an I/O error,
// NOT errInvalidInput.
func TestBulkFromMissingFile(t *testing.T) {
	_, err := runCmd(t, "cpf", "--from", filepath.Join(t.TempDir(), "nope.txt"))
	require.Error(t, err)
	assert.False(t, errors.Is(err, errInvalidInput), "missing-file error is an I/O error, not invalid-input")
}

// TestBulkFromAcrossKinds iterates every registered kind, generates one value
// via the CLI, writes it to a temp file, and asserts that --from validates it
// with exit 0. This locks in that the shared streamValidate path is reused
// across all kinds without any kind-specific wiring gaps.
func TestBulkFromAcrossKinds(t *testing.T) {
	t.Helper()

	root := newRootCmd()
	for _, sub := range root.Commands() {
		name := sub.Name()
		// Only exercise document-kind subcommands (skip detect, version, etc.).
		if sub.Flags().Lookup("generate") == nil {
			continue
		}

		t.Run(name, func(t *testing.T) {
			// Generate one value.
			genOut, err := runCmd(t, name, "--generate")
			require.NoError(t, err, "generate %s", name)
			value := strings.TrimSpace(genOut)
			require.NotEmpty(t, value)

			// Write to a temp file.
			dir := t.TempDir()
			path := filepath.Join(dir, name+".txt")
			require.NoError(t, os.WriteFile(path, []byte(value+"\n"), 0o600))

			// Validate via --from; generated values must be all-valid → exit 0.
			out, fromErr := runCmd(t, name, "--from", path)
			require.NoError(t, fromErr, "--from %s exited non-zero; output: %q", name, out)
			assert.True(t, strings.HasPrefix(out, "valid\t"), "expected valid\t prefix, got %q", out)
		})
	}
}
