package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	out, err := runCmd(t, "version")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, "brdoc "), "got %q", out)
	assert.NotEmpty(t, strings.TrimSpace(strings.TrimPrefix(out, "brdoc ")))
}

func TestVersionFuncDefault(t *testing.T) {
	assert.NotEmpty(t, version())
}

func TestRootHasNoCompletionCmd(t *testing.T) {
	root := newRootCmd()
	assert.Nil(t, findCmd(root, "completion"))
}

func TestRootSilences(t *testing.T) {
	root := newRootCmd()
	assert.True(t, root.SilenceUsage)
	assert.True(t, root.SilenceErrors)
}

func TestInvalidInputSentinel(t *testing.T) {
	_, err := runCmd(t, "cpf", "--validate", "000.000.000-00")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errInvalidInput))
}
