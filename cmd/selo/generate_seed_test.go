package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerate_Deterministic verifies that --seed makes --generate / --bulk
// reproducible while a seeded batch still yields distinct values.
func TestGenerate_Deterministic(t *testing.T) {
	// Same seed → byte-identical output across runs.
	out1, err := runCmd(t, "cpf", "--generate", "--seed", "42", "--count", "5")
	require.NoError(t, err)
	out2, err := runCmd(t, "cpf", "--generate", "--seed", "42", "--count", "5")
	require.NoError(t, err)
	assert.Equal(t, out1, out2, "same seed must produce identical output")

	// A different seed → different output.
	out3, err := runCmd(t, "cpf", "--generate", "--seed", "7", "--count", "5")
	require.NoError(t, err)
	assert.NotEqual(t, out1, out3, "a different seed should produce different output")

	// A seeded batch yields distinct values (shared advancing stream).
	lines := strings.Split(strings.TrimSpace(out1), "\n")
	require.Len(t, lines, 5)
	assert.NotEqual(t, lines[0], lines[1], "a seeded batch must yield distinct values")

	// --bulk honours --seed too.
	b1, err := runCmd(t, "cnpj", "--bulk", "3", "--seed", "99")
	require.NoError(t, err)
	b2, err := runCmd(t, "cnpj", "--bulk", "3", "--seed", "99")
	require.NoError(t, err)
	assert.Equal(t, b1, b2, "same seed must produce identical --bulk output")
}
