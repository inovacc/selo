package selo

import (
	"fmt"
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

			// RG and IE only for the implemented UF (SP).
			if uf == UFSP {
				assert.NotEmpty(t, p.RG)
				assert.Truef(t, NewRG().Validate(p.RG), "RG %q", p.RG)
				assert.NotEmpty(t, p.IE)
				ieOK, ieErr := NewIE().ValidateUF(p.IE, UFSP)
				assert.NoError(t, ieErr)
				assert.Truef(t, ieOK, "IE %q", p.IE)
			} else {
				assert.Empty(t, p.RG)
				assert.Empty(t, p.IE)
			}

			// PIX keys are all valid; identity fields populated.
			require.Len(t, p.PIXKeys, 4)

			for _, k := range p.PIXKeys {
				assert.Truef(t, NewPIX().Validate(k), "pix key %q", k)
			}

			assert.NotEmpty(t, p.Name)
			assert.Contains(t, p.Email, "@")
		})
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
