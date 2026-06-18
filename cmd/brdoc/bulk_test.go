package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/inovacc/brdoc"
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

// TestBulkFromAcrossKinds iterates every registered kind via sdk.Kinds(),
// generates one value using the registry, writes it to a temp file, and asserts
// that --from validates it with exit 0. This locks in that the shared
// streamValidate path is reused across all kinds without any kind-specific
// wiring gaps, and future kinds cannot be silently skipped.
func TestBulkFromAcrossKinds(t *testing.T) {
	for _, kind := range sdk.Kinds() {
		kind := kind // capture for parallel subtest
		name := kind.String()

		t.Run(name, func(t *testing.T) {
			// Generate one value via the registry (no CLI round-trip needed).
			value, err := sdk.Generate(kind)
			require.NoError(t, err, "sdk.Generate(%s)", name)
			require.NotEmpty(t, value)

			// Write raw (unformatted) value to a temp file.
			dir := t.TempDir()
			path := filepath.Join(dir, name+".txt")
			require.NoError(t, os.WriteFile(path, []byte(value+"\n"), 0o600))

			// Validate via --from; generated values must be all-valid → exit 0.
			out, fromErr := runCmd(t, name, "--from", path)
			require.NoError(t, fromErr, "--from %s exited non-zero; output: %q", name, out)

			// Determine expected formatted value; fall back to raw if Format errors.
			formatted, fmtErr := sdk.Format(kind, value)
			if fmtErr != nil {
				formatted = value
			}
			assert.Equal(t, "valid\t"+formatted+"\n", out, "unexpected --from output for %s", name)
		})
	}
}
