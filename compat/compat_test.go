package compat

import (
	"github.com/inovacc/selo"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsDigitDocs_ParityWithRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		compat  func(string) bool
		root    func(string) bool
		valid   string // a generated-valid sample produced below
		invalid string
	}{
		{"cpf", IsCPF, func(s string) bool { return selo.NewCPF().Validate(s) }, selo.NewCPF().Generate(), "00000000000"},
		{"cnpj", IsCNPJ, func(s string) bool { return selo.NewCNPJ().Validate(s) }, selo.NewCNPJ().Generate(), "39591842000011"},
		{"cnh", IsCNH, func(s string) bool { return selo.NewCNH().Validate(s) }, selo.NewCNH().Generate(), "11111111111"},
		{"pis", IsPIS, func(s string) bool { return selo.NewPIS().Validate(s) }, selo.NewPIS().Generate(), "00000000001"},
		{"renavam", IsRENAVAM, func(s string) bool { return selo.NewRenavam().Validate(s) }, selo.NewRenavam().Generate(), "00000000001"},
		{"voterid", IsVoterID, func(s string) bool { return selo.NewVoterID().Validate(s) }, selo.NewVoterID().Generate(), "000000000000"},
		{"cns", IsCNS, func(s string) bool { return selo.NewCNS().Validate(s) }, selo.NewCNS().Generate(), "000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, tt.compat(tt.valid), "compat must accept a valid %s", tt.name)
			assert.Equal(t, tt.root(tt.valid), tt.compat(tt.valid), "compat must mirror root on valid %s", tt.name)
			assert.False(t, tt.compat(tt.invalid), "compat must reject invalid %s", tt.name)
			assert.Equal(t, tt.root(tt.invalid), tt.compat(tt.invalid), "compat must mirror root on invalid %s", tt.name)
		})
	}
}
func TestPlateWrappers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		value        string
		wantPlate    bool
		wantNational bool
		wantMercosul bool
	}{
		{"ABC1234", true, true, false},   // legacy national, no dash
		{"ABC-1234", true, true, false},  // legacy national, dashed
		{"ABC1D23", true, false, true},   // Mercosul
		{"AB1234", false, false, false},  // too short
		{"1234ABC", false, false, false}, // wrong order
		{"ABCD123", false, false, false}, // 4 letters
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantPlate, IsPlate(tt.value), "IsPlate(%q)", tt.value)
			assert.Equal(t, tt.wantNational, IsNationalPlate(tt.value), "IsNationalPlate(%q)", tt.value)
			assert.Equal(t, tt.wantMercosul, IsMercosulPlate(tt.value), "IsMercosulPlate(%q)", tt.value)
		})
	}
}

func TestPlateWrappers_ParityWithRoot(t *testing.T) {
	t.Parallel()
	p := selo.NewPlate()
	for _, v := range []string{"ABC1234", "ABC-1234", "ABC1D23", "AB1234"} {
		assert.Equal(t, p.Validate(v), IsPlate(v), "IsPlate parity %q", v)
		assert.Equal(t, p.ValidateNational(v), IsNationalPlate(v), "IsNationalPlate parity %q", v)
		assert.Equal(t, p.ValidateMercosul(v), IsMercosulPlate(v), "IsMercosulPlate parity %q", v)
	}
}
func TestIsCEP(t *testing.T) {
	t.Parallel()
	// Generate a valid CEP and resolve its expected UF from the root type so
	// the test stays correct regardless of which range the generator picked.
	cep := selo.NewCEP()
	valid := cep.Generate()
	wantUF, err := cep.Origin(valid)
	assert.NoError(t, err, "root must resolve origin for its own generated CEP")

	ok, gotUF := IsCEP(valid)
	assert.True(t, ok, "IsCEP must accept a valid CEP %q", valid)
	assert.Equal(t, selo.UF(wantUF), gotUF, "IsCEP must return the same UF as root.Origin")

	// Invalid CEP -> (false, zero UF).
	badOK, badUF := IsCEP("000")
	assert.False(t, badOK, "IsCEP must reject a too-short value")
	assert.Equal(t, UF(""), badUF, "invalid CEP must return the zero UF")
}

func TestIsCEPFrom(t *testing.T) {
	t.Parallel()
	cep := selo.NewCEP()
	valid := cep.Generate()
	originStr, err := cep.Origin(valid)
	assert.NoError(t, err)
	uf := selo.UF(originStr)

	assert.True(t, IsCEPFrom(valid, uf), "IsCEPFrom must accept when uf matches")
	assert.True(t, IsCEPFrom(valid), "IsCEPFrom with no UFs must behave like IsCEP (valid -> true)")
	// Pick a UF guaranteed different from the resolved one.
	other := selo.UFSP
	if uf == selo.UFSP {
		other = selo.UFRJ
	}
	assert.False(t, IsCEPFrom(valid, other), "IsCEPFrom must reject when uf does not match")
	assert.False(t, IsCEPFrom("000", uf), "IsCEPFrom must reject an invalid CEP")
}
func TestIsPhone(t *testing.T) {
	t.Parallel()
	phone := selo.NewPhone()
	valid := phone.Generate()
	wantUF, err := phone.Origin(valid)
	assert.NoError(t, err, "root must resolve origin for its own generated phone")

	ok, gotUF := IsPhone(valid)
	assert.True(t, ok, "IsPhone must accept a valid phone %q", valid)
	assert.Equal(t, selo.UF(wantUF), gotUF, "IsPhone must return the same UF as root.Origin")

	badOK, badUF := IsPhone("123")
	assert.False(t, badOK, "IsPhone must reject a too-short value")
	assert.Equal(t, UF(""), badUF, "invalid phone must return the zero UF")
}

func TestIsPhoneFrom(t *testing.T) {
	t.Parallel()
	phone := selo.NewPhone()
	valid := phone.Generate()
	originStr, err := phone.Origin(valid)
	assert.NoError(t, err)
	uf := selo.UF(originStr)

	assert.True(t, IsPhoneFrom(valid, uf), "IsPhoneFrom must accept when uf matches")
	assert.True(t, IsPhoneFrom(valid), "IsPhoneFrom with no UFs must behave like IsPhone (valid -> true)")
	other := selo.UFSP
	if uf == selo.UFSP {
		other = selo.UFRJ
	}
	assert.False(t, IsPhoneFrom(valid, other), "IsPhoneFrom must reject when uf does not match")
	assert.False(t, IsPhoneFrom("123", uf), "IsPhoneFrom must reject an invalid phone")
}
func TestIsRG(t *testing.T) {
	t.Parallel()
	rg := selo.NewRG()

	// Valid SP RG sample: 33.962.657-4 (mod-11 weights 2..9, DV=11-(sum%11)=4).
	const validSP = "33.962.657-4"
	wantOK, wantErr := rg.ValidateUF(validSP, selo.UFSP)
	gotOK, gotErr := IsRG(validSP, selo.UFSP)
	assert.Equal(t, wantOK, gotOK, "IsRG must mirror root validity for a valid SP RG")
	assert.Equal(t, wantErr, gotErr, "IsRG must mirror root error for a valid SP RG")
	assert.True(t, gotOK, "valid SP RG must pass")
	assert.NoError(t, gotErr, "valid SP RG must not error")

	// Wrong check digit for SP -> (false, nil) (well-formed but invalid).
	badOK, badErr := IsRG("33.962.657-0", selo.UFSP)
	assert.False(t, badOK, "off-by-one check digit must fail")
	assert.NoError(t, badErr, "an invalid-but-wellformed RG is (false, nil), not an error")

	// Unimplemented UF -> error wrapping ErrUFNotImplemented.
	ufOK, ufErr := IsRG("12345678", selo.UFAC)
	assert.False(t, ufOK, "unimplemented UF must not validate")
	assert.ErrorIs(t, ufErr, selo.ErrUFNotImplemented, "unimplemented UF must wrap ErrUFNotImplemented")
}

// TestSignatureParity is a compile-time guard: each assignment fails to build
// if a wrapper's signature drifts from paemuri/brdoc v3. It also exercises the
// values at runtime so the test counts toward coverage.
func TestSignatureParity(t *testing.T) {
	t.Parallel()
	var (
		_ func(string) bool              = IsCPF
		_ func(string) bool              = IsCNPJ
		_ func(string) bool              = IsCNH
		_ func(string) bool              = IsPIS
		_ func(string) bool              = IsRENAVAM
		_ func(string) bool              = IsVoterID
		_ func(string) bool              = IsCNS
		_ func(string) bool              = IsPlate
		_ func(string) bool              = IsNationalPlate
		_ func(string) bool              = IsMercosulPlate
		_ func(string) (bool, UF)        = IsCEP
		_ func(string, ...UF) bool       = IsCEPFrom
		_ func(string) (bool, UF)        = IsPhone
		_ func(string, ...UF) bool       = IsPhoneFrom
		_ func(string, UF) (bool, error) = IsRG
	)
	// UF must be the SAME type as selo.UF (alias, not a defined type), so a
	// selo.UF is assignable to compat.UF with no conversion.
	var u UF = selo.UFSP
	assert.Equal(t, "SP", string(u))
}
