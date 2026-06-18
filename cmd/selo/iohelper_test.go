package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenReaderStdin(t *testing.T) {
	r, closeFn, err := openReader("-")
	require.NoError(t, err)
	assert.Nil(t, closeFn)
	assert.Equal(t, os.Stdin, r)
}

func TestOpenReaderFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docs.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))

	r, closeFn, err := openReader(path)
	require.NoError(t, err)
	require.NotNil(t, closeFn)
	defer closeFn()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	assert.Equal(t, "hello\n", buf.String())
}

func TestOpenReaderMissingFile(t *testing.T) {
	_, _, err := openReader(filepath.Join(t.TempDir(), "nope.txt"))
	assert.Error(t, err)
}

func TestStreamValidate(t *testing.T) {
	in := strings.NewReader("# comment\n\n111\n222\n   333   \n")
	out := new(bytes.Buffer)

	// Treat "222" as the only invalid line; format doubles the value.
	fn := func(value string) (string, bool) {
		if value == "222" {
			return "", false
		}
		return value + value, true
	}

	anyInvalid, err := streamValidate(in, out, fn)
	require.NoError(t, err)
	assert.True(t, anyInvalid)
	assert.Equal(t, "valid\t111111\ninvalid\t222\nvalid\t333333\n", out.String())
}

func TestStreamValidateBareValid(t *testing.T) {
	in := strings.NewReader("abc\n")
	out := new(bytes.Buffer)
	fn := func(value string) (string, bool) { return "", true }

	anyInvalid, err := streamValidate(in, out, fn)
	require.NoError(t, err)
	assert.False(t, anyInvalid)
	assert.Equal(t, "valid\n", out.String())
}
