package main

import (
	"errors"
	"strings"
	"testing"

	sdk "github.com/inovacc/selo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	out, err := runCmd(t, "version")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, sdk.AppName+" "), "got %q", out)
	assert.NotEmpty(t, strings.TrimSpace(strings.TrimPrefix(out, sdk.AppName+" ")))
}

func TestVersionFuncDefault(t *testing.T) {
	assert.NotEmpty(t, version())
}

func TestRootHasCompletionCmd(t *testing.T) {
	root := newRootCmd()
	assert.NotNil(t, findCmd(root, "completion"))
}

func TestCompletionBashGenerates(t *testing.T) {
	out, err := runCmd(t, "completion", "bash")
	require.NoError(t, err)
	assert.Contains(t, out, "# bash completion")
	assert.Contains(t, out, sdk.CLIUse)
}

func TestCompletionRejectsUnknownShell(t *testing.T) {
	_, err := runCmd(t, "completion", "tcsh")
	require.Error(t, err)
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
