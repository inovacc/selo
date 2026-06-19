package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCPF(t *testing.T) {
	out, err := runCmd(t, "detect", "529.982.247-25")
	require.NoError(t, err)
	assert.Equal(t, "cpf\n", out)
}

func TestDetectCNPJ(t *testing.T) {
	out, err := runCmd(t, "detect", "39591842000010")
	require.NoError(t, err)
	assert.Equal(t, "cnpj\n", out)
}

func TestDetectUnknown(t *testing.T) {
	out, err := runCmd(t, "detect", "12345")
	require.Error(t, err)
	assert.Equal(t, "unknown\n", out)
}

func TestDetectRequiresArg(t *testing.T) {
	_, err := runCmd(t, "detect")
	require.Error(t, err)
}
