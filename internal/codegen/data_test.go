package codegen_test

import (
	"sort"
	"testing"

	"github.com/inovacc/selo"
	"github.com/inovacc/selo/internal/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cepUFForPrefix scans the extracted CEP table (first match wins, mirroring the
// selo validator) and returns the UF whose range contains prefix.
func cepUFForPrefix(prefix int) (string, bool) {
	for _, r := range codegen.CEPRanges() {
		if prefix >= r.From && prefix <= r.To {
			return r.UF, true
		}
	}

	return "", false
}

// TestCEPRanges_MatchSeloOrigin asserts the extracted CEP table resolves known
// prefixes to the same UF that selo's CEP Origin reports.
func TestCEPRanges_MatchSeloOrigin(t *testing.T) {
	cep, ok := selo.Get(selo.KindCEP)
	require.True(t, ok)

	origin := cep.(selo.OriginResolver)

	samples := []struct {
		value string
		uf    string
	}{
		{"01310-100", "SP"}, // Av. Paulista
		{"20040-002", "RJ"},
		{"40010-000", "BA"},
		{"80010-000", "PR"},
		{"90010-000", "RS"},
		{"70040-010", "DF"},
	}
	for _, s := range samples {
		want, err := origin.Origin(s.value)
		require.NoErrorf(t, err, "selo origin for %s", s.value)
		assert.Equalf(t, s.uf, want, "selo origin for %s", s.value)

		prefix := int(s.value[0]-'0')*100 + int(s.value[1]-'0')*10 + int(s.value[2]-'0')
		got, found := cepUFForPrefix(prefix)
		require.Truef(t, found, "extracted table missing prefix %03d", prefix)
		assert.Equalf(t, want, got, "extracted CEP UF for %s", s.value)
	}
}

// TestDDDtoUF_MatchSeloOrigin asserts the extracted DDD map agrees with selo's
// phone Origin for known area codes.
func TestDDDtoUF_MatchSeloOrigin(t *testing.T) {
	phone, ok := selo.Get(selo.KindPhone)
	require.True(t, ok)

	origin := phone.(selo.OriginResolver)

	ddd := codegen.DDDtoUF()

	samples := []struct {
		ddd  string
		full string
		uf   string
	}{
		{"11", "11999990000", "SP"},
		{"21", "21999990000", "RJ"},
		{"31", "31999990000", "MG"},
		{"51", "51999990000", "RS"},
		{"61", "61999990000", "DF"},
		{"71", "71999990000", "BA"},
	}
	for _, s := range samples {
		want, err := origin.Origin(s.full)
		require.NoErrorf(t, err, "selo origin for %s", s.full)
		assert.Equalf(t, s.uf, want, "selo origin for %s", s.full)
		assert.Equalf(t, selo.UF(want), ddd[s.ddd], "extracted DDD UF for %s", s.ddd)
	}
}

// TestDDDList_SortedAndComplete asserts DDDList is sorted ascending and covers
// the same set as the map.
func TestDDDList_SortedAndComplete(t *testing.T) {
	list := codegen.DDDList()
	require.NotEmpty(t, list)
	assert.True(t, sort.IntsAreSorted(list), "DDDList must be sorted")
	assert.Equal(t, len(codegen.DDDtoUF()), len(list), "DDDList length must equal map size")
}

// TestCPFRegions_MatchSeloOrigin asserts the extracted CPF region map agrees with
// selo's CPF Origin for every ninth digit.
func TestCPFRegions_MatchSeloOrigin(t *testing.T) {
	cpf, ok := selo.Get(selo.KindCPF)
	require.True(t, ok)

	origin := cpf.(selo.OriginResolver)

	regions := codegen.CPFRegions()
	require.Len(t, regions, 10)

	for digit := 0; digit <= 9; digit++ {
		// Build a CPF whose ninth digit is `digit`. We only need Origin, which
		// reads position 8 regardless of validity, so any 9+ digit string works.
		base := make([]byte, 11)
		for i := range base {
			base[i] = '0'
		}

		base[8] = byte('0' + digit)
		want, err := origin.Origin(string(base))
		require.NoErrorf(t, err, "selo cpf origin for digit %d", digit)
		assert.Equalf(t, want, regions[digit], "extracted CPF region for digit %d", digit)
	}
}

// TestVoterUFNames_MatchSeloOrigin spot-checks the voter UF-code region map.
func TestVoterUFNames_MatchSeloOrigin(t *testing.T) {
	names := codegen.VoterUFNames()
	assert.Equal(t, "São Paulo", names[1])
	assert.Equal(t, "Minas Gerais", names[2])
	assert.Equal(t, "Exterior", names[28])
	assert.Len(t, names, 28)
}
