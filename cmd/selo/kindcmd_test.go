package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	sdk "github.com/inovacc/selo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := newRootCmd()
	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs(args)
	err := root.Execute()

	return out.String(), err
}

func TestKindCmdGenerateCPF(t *testing.T) {
	out, err := runCmd(t, "cpf", "--generate")
	require.NoError(t, err)

	got := strings.TrimSpace(out)
	assert.True(t, sdk.NewCPF().Validate(got), "generated CPF must validate: %q", got)
}

func TestKindCmdGenerateCount(t *testing.T) {
	out, err := runCmd(t, "cpf", "--generate", "--count", "3")
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 3)
}

func TestKindCmdValidateValidCPF(t *testing.T) {
	// 529.982.247-25 is a well-known valid CPF.
	out, err := runCmd(t, "cpf", "--validate", "529.982.247-25")
	require.NoError(t, err)
	assert.Equal(t, "valid\t529.982.247-25\n", out)
}

func TestKindCmdValidateInvalidCPF(t *testing.T) {
	out, err := runCmd(t, "cpf", "--validate", "123.456.789-00")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errInvalidInput), "expected errInvalidInput, got %v", err)
	assert.Equal(t, "invalid\n", out)
}

func TestKindCmdFromBulkWithInvalid(t *testing.T) {
	// Write a temp file containing one valid and one invalid CPF.
	f, err := os.CreateTemp(t.TempDir(), "cpfs-*.txt")
	require.NoError(t, err)

	_, _ = f.WriteString("529.982.247-25\n123.456.789-00\n")
	require.NoError(t, f.Close())

	out, cmdErr := runCmd(t, "cpf", "--from", f.Name())
	require.Error(t, cmdErr)
	assert.True(t, errors.Is(cmdErr, errInvalidInput), "expected errInvalidInput, got %v", cmdErr)
	// Both lines must still be printed.
	assert.Contains(t, out, "valid")
	assert.Contains(t, out, "invalid")
}

func TestKindCmdFromBulkAllValid(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "cpfs-*.txt")
	require.NoError(t, err)

	_, _ = f.WriteString("529.982.247-25\n")
	require.NoError(t, f.Close())

	out, cmdErr := runCmd(t, "cpf", "--from", f.Name())
	require.NoError(t, cmdErr)
	assert.Contains(t, out, "valid")
}

func TestKindCmdFormatCNPJ(t *testing.T) {
	// 39591842000010 is a valid CNPJ; canonical format is 39.591.842/0000-10.
	out, err := runCmd(t, "cnpj", "--format", "39591842000010")
	require.NoError(t, err)
	assert.Equal(t, "39.591.842/0000-10\n", out)
}

func TestKindCmdOriginOnlyForResolver(t *testing.T) {
	cpf := newKindCmd(sdk.KindCPF)
	assert.NotNil(t, cpf.Flags().Lookup("origin"), "cpf must expose --origin (OriginResolver)")

	cnpj := newKindCmd(sdk.KindCNPJ)
	assert.Nil(t, cnpj.Flags().Lookup("origin"), "cnpj must NOT expose --origin")
}

func TestKindCmdUFOnlyForUFScoped(t *testing.T) {
	cpf := newKindCmd(sdk.KindCPF)
	assert.Nil(t, cpf.Flags().Lookup("uf"), "cpf must NOT expose --uf")
}

func TestKindCmdOriginCPF(t *testing.T) {
	out, err := runCmd(t, "cpf", "--origin", "529.982.247-25")
	require.NoError(t, err)
	assert.NotEmpty(t, strings.TrimSpace(out))
}

func TestKindCmdNoFlags(t *testing.T) {
	_, err := runCmd(t, "cpf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either")
}

func TestKindCmdGenerateConflictsValidate(t *testing.T) {
	_, err := runCmd(t, "cpf", "--generate", "--validate", "529.982.247-25")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used with")
}

func TestRegisterKindCommands(t *testing.T) {
	root := newRootCmd()
	for _, k := range sdk.Kinds() {
		assert.NotNil(t, findCmd(root, k.String()), "missing subcommand for %s", k)
	}
}

func findCmd(root *cobra.Command, name string) *cobra.Command {
	for _, c := range root.Commands() {
		if c.Name() == name {
			return c
		}
	}

	return nil
}
