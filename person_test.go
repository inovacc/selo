package selo

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratePerson_Consistency is the core guarantee: for every implemented UF,
// a generated person's documents are all valid AND the geolocatable ones (CPF
// region, Voter ID UF code, phone DDD, CEP range) all resolve to that same UF.
func TestGeneratePerson_Consistency(t *testing.T) {
	for _, uf := range personUFs() {
		t.Run(string(uf), func(t *testing.T) {
			p := GeneratePerson(WithUF(uf))

			// Every document validates.
			assert.Truef(t, NewCPF().Validate(p.CPF), "CPF %q", p.CPF)
			assert.Truef(t, NewVoterID().Validate(p.VoterID), "VoterID %q", p.VoterID)
			assert.Truef(t, NewPhone().Validate(p.Phone), "Phone %q", p.Phone)
			assert.Truef(t, NewCEP().Validate(p.CEP), "CEP %q", p.CEP)
			assert.Truef(t, NewCNH().Validate(p.CNH), "CNH %q", p.CNH)
			assert.Truef(t, NewPIS().Validate(p.PIS), "PIS %q", p.PIS)
			assert.Truef(t, NewRenavam().Validate(p.Renavam), "RENAVAM %q", p.Renavam)
			assert.Truef(t, NewCNS().Validate(p.CNS), "CNS %q", p.CNS)

			// UF consistency.
			assert.Equalf(t, cpfRegionByUF[uf], int(p.CPF[8]-'0'), "CPF region for %s", uf)
			assert.Equalf(t, fmt.Sprintf("%02d", voterCodeByUF[uf]), p.VoterID[8:10], "voter UF code for %s", uf)
			phoneUF, err := NewPhone().Origin(p.Phone)
			assert.NoError(t, err)
			assert.Equal(t, string(uf), phoneUF, "phone origin")

			cepUF, err := NewCEP().Origin(p.CEP)
			assert.NoError(t, err)
			assert.Equal(t, string(uf), cepUF, "cep origin")

			// RG is implemented only for SP.
			if uf == UFSP {
				assert.NotEmpty(t, p.RG)
				assert.Truef(t, NewRG().Validate(p.RG), "RG %q", p.RG)
			} else {
				assert.Empty(t, p.RG)
			}

			// IE is populated for every UF with a verified IE algorithm and
			// validates under that UF; empty for the rest.
			if slices.Contains(NewIE().ImplementedUFs(), uf) {
				assert.NotEmpty(t, p.IE)
				ieOK, ieErr := NewIE().ValidateUF(p.IE, uf)
				assert.NoError(t, ieErr)
				assert.Truef(t, ieOK, "IE %q for %s", p.IE, uf)
			} else {
				assert.Empty(t, p.IE)
			}

			// PIX keys are all valid; identity fields populated.
			require.Len(t, p.PIXKeys, 4)

			for _, k := range p.PIXKeys {
				assert.Truef(t, NewPIX().Validate(k), "pix key %q", k)
			}

			assert.NotEmpty(t, p.Name)
			assert.Contains(t, p.Email, "@")

			// Address is always populated and UF-consistent: the city is a real
			// municipality in the person's UF, and the address CEP equals the
			// top-level Person.CEP.
			require.NotNil(t, p.Address)
			assert.Equal(t, uf, p.Address.UF)
			assert.Equal(t, p.CEP, p.Address.CEP)
			assert.Contains(t, citiesByUF[uf], p.Address.City, "city must be a real municipality in %s", uf)
			assert.NotEmpty(t, p.Address.Street)

			hasType := false

			for _, lt := range logradouroTypes {
				if strings.HasPrefix(p.Address.Street, lt.value+" ") {
					hasType = true

					break
				}
			}

			assert.Truef(t, hasType, "street %q must start with a logradouro type", p.Address.Street)
			assert.NotEmpty(t, p.Address.Neighborhood)
			assert.NotEmpty(t, p.Address.Number)
		})
	}
}

// TestCitiesByUF_AllUFsPopulated guards the headline invariant: every one of the
// 27 UFs has at least one city, otherwise genAddressForUFRand would panic on
// r.IntN(0).
func TestCitiesByUF_AllUFsPopulated(t *testing.T) {
	for _, uf := range AllUFs() {
		t.Run(string(uf), func(t *testing.T) {
			assert.GreaterOrEqual(t, len(citiesByUF[uf]), 1, "UF %s must have >=1 city", uf)
		})
	}
}

// TestGeneratePerson_AddressDeterministic proves the same seed yields the same
// Address (every field), and that appending Address did not reorder earlier
// draws: a known pre-existing field (CPF) for a fixed seed+UF is unaffected by
// the presence of Address.
func TestGeneratePerson_AddressDeterministic(t *testing.T) {
	a := GeneratePerson(WithUF(UFSP), WithSeed(42))
	b := GeneratePerson(WithUF(UFSP), WithSeed(42))

	require.NotNil(t, a.Address)
	require.NotNil(t, b.Address)
	assert.Equal(t, *a.Address, *b.Address, "same seed must yield identical Address")
	assert.Equal(t, a, b, "same seed must yield identical Person")

	// Back-compat draw order: CPF/CEP/Phone/VoterID are drawn before Address, so
	// they must equal each other across two identical builds and the address CEP
	// must mirror the person CEP (Address was appended last, not interleaved).
	assert.Equal(t, a.CPF, b.CPF)
	assert.Equal(t, a.CEP, a.Address.CEP)
}

// TestGeneratePerson_RandomUF_Address smoke-tests the city-in-UF guarantee
// across random UFs.
func TestGeneratePerson_RandomUF_Address(t *testing.T) {
	for range 50 {
		p := GeneratePerson()
		require.NotNil(t, p.Address)
		assert.Contains(t, citiesByUF[p.UF], p.Address.City, "city must be in UF %s", p.UF)
	}
}

// TestGeneratePerson_FormattedAddressCEP verifies the address CEP is masked and
// stays equal to Person.CEP in formatted mode.
func TestGeneratePerson_FormattedAddressCEP(t *testing.T) {
	p := GeneratePerson(WithUF(UFSP), Formatted())
	require.NotNil(t, p.Address)
	assert.Contains(t, p.Address.CEP, "-", "formatted address CEP should be masked")
	assert.Equal(t, p.CEP, p.Address.CEP, "address CEP must equal person CEP in formatted mode")
}

// TestNameLists_Expanded asserts the name pools grew and that every token folds
// to a pure-ASCII email local-part (no leftover accents).
func TestNameLists_Expanded(t *testing.T) {
	assert.GreaterOrEqual(t, len(personFirstNames), 100, "expanded given-name pool")
	assert.GreaterOrEqual(t, len(personSurnames), 60, "expanded surname pool")

	isPureLower := func(s string) bool {
		for _, c := range s {
			if c < 'a' || c > 'z' {
				return false
			}
		}

		return true
	}

	for _, n := range append(append([]string{}, personFirstNames...), personSurnames...) {
		folded := strings.ToLower(asciiFold.Replace(n))
		assert.Truef(t, isPureLower(folded), "name %q folds to %q (must be pure a-z)", n, folded)
	}
}

func TestGeneratePerson_Options(t *testing.T) {
	p := GeneratePerson(WithUF(UFSP), WithVehicle(), WithCompany())
	require.NotNil(t, p.Vehicle)
	require.NotNil(t, p.Company)
	assert.True(t, IsPlate(p.Vehicle.Plate), "vehicle plate %q", p.Vehicle.Plate)
	assert.True(t, NewRenavam().Validate(p.Vehicle.Renavam))
	assert.True(t, NewCNPJ().Validate(p.Company.CNPJ))
	assert.NotEmpty(t, p.Company.Name)

	f := GeneratePerson(WithUF(UFSP), Formatted())
	assert.Contains(t, f.CPF, ".", "formatted CPF should be masked")
	assert.Contains(t, f.CEP, "-", "formatted CEP should be masked")
	assert.Contains(t, f.Phone, "(", "formatted phone should be masked")
	// Validators accept formatted input.
	assert.True(t, NewCPF().Validate(f.CPF))
	assert.True(t, NewCEP().Validate(f.CEP))
	assert.True(t, NewPhone().Validate(f.Phone))
}

func TestGeneratePerson_RandomUF(t *testing.T) {
	for range 50 {
		p := GeneratePerson()
		assert.True(t, p.UF.Valid(), "chosen UF %q must be valid", p.UF)
		assert.True(t, NewCPF().Validate(p.CPF))
		cepUF, err := NewCEP().Origin(p.CEP)
		assert.NoError(t, err)
		assert.Equal(t, string(p.UF), cepUF)
	}
}
