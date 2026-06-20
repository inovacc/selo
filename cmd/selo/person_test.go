package main

import (
	"encoding/json"
	"strings"
	"testing"

	sdk "github.com/inovacc/selo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersonCmd_JSONSingle(t *testing.T) {
	out, err := runCmd(t, "person", "--uf", "SP", "--json")
	require.NoError(t, err)

	var p sdk.Person
	require.NoError(t, json.Unmarshal([]byte(out), &p))
	assert.Equal(t, sdk.UFSP, p.UF)
	assert.True(t, sdk.NewCPF().Validate(p.CPF), "CPF %q must validate", p.CPF)
	cepUF, err := sdk.NewCEP().Origin(p.CEP)
	require.NoError(t, err)
	assert.Equal(t, "SP", cepUF)
}

func TestPersonCmd_JSONArray(t *testing.T) {
	out, err := runCmd(t, "person", "--count", "3", "--uf", "MG", "--json")
	require.NoError(t, err)

	var people []sdk.Person
	require.NoError(t, json.Unmarshal([]byte(out), &people))
	require.Len(t, people, 3)

	for _, p := range people {
		assert.Equal(t, sdk.UFMG, p.UF)
	}
}

func TestPersonCmd_TextWithExtras(t *testing.T) {
	out, err := runCmd(t, "person", "--uf", "RJ", "--with-vehicle", "--with-company")
	require.NoError(t, err)
	assert.Contains(t, out, "RJ")
	assert.Contains(t, out, "Vehicle:")
	assert.Contains(t, out, "Company:")
	assert.Contains(t, strings.ToLower(out), "cpf:")
}

func TestPersonCmd_InvalidUF(t *testing.T) {
	_, err := runCmd(t, "person", "--uf", "ZZ")
	require.Error(t, err)
}

func TestPersonCmd_TextAddress(t *testing.T) {
	out, err := runCmd(t, "person", "--uf", "SP")
	require.NoError(t, err)
	assert.Contains(t, out, "Address:")
	assert.Contains(t, out, "SP")

	// The printed city must be one of the real SP municipalities.
	spCities := []string{
		"São Paulo", "Guarulhos", "Campinas", "São Bernardo do Campo",
		"Santo André", "Osasco", "Ribeirão Preto", "Sorocaba",
		"Santos", "São José dos Campos",
	}

	found := false

	for _, c := range spCities {
		if strings.Contains(out, c) {
			found = true

			break
		}
	}

	assert.True(t, found, "Address line must name a real SP city, got: %s", out)
}

func TestPersonCmd_Deterministic(t *testing.T) {
	// Same seed → byte-identical output across runs.
	out1, err := runCmd(t, "person", "--seed", "42", "--uf", "SP", "--count", "3", "--json")
	require.NoError(t, err)
	out2, err := runCmd(t, "person", "--seed", "42", "--uf", "SP", "--count", "3", "--json")
	require.NoError(t, err)
	assert.Equal(t, out1, out2, "same seed must produce identical output")

	// A different seed → different output.
	out3, err := runCmd(t, "person", "--seed", "99", "--uf", "SP", "--count", "3", "--json")
	require.NoError(t, err)
	assert.NotEqual(t, out1, out3, "a different seed should produce different output")

	// Within one seeded batch the people are distinct (shared advancing stream),
	// not three copies of the same person.
	var people []sdk.Person
	require.NoError(t, json.Unmarshal([]byte(out1), &people))
	require.Len(t, people, 3)
	assert.NotEqual(t, people[0].CPF, people[1].CPF, "seeded batch must yield distinct people")
	assert.NotEqual(t, people[1].CPF, people[2].CPF, "seeded batch must yield distinct people")
}
