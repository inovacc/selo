package compat

import (
	"github.com/inovacc/brdoc"
)

// Package compat provides drop-in replacements for the public API of
// github.com/paemuri/brdoc/v3. Every function is a thin wrapper over the
// root github.com/inovacc/brdoc package, so a paemuri user can migrate by
// changing a single import path. No validation logic lives here.

// UF aliases the root brdoc.UF type so that the wrapper signatures below
// match paemuri/brdoc v3 exactly (e.g. func IsCEP(s string) (bool, UF)).
type UF = brdoc.UF

// IsCPF reports whether s is a valid CPF. Mirrors paemuri/brdoc.IsCPF.
func IsCPF(s string) bool { return brdoc.NewCPF().Validate(s) }

// IsCNPJ reports whether s is a valid CNPJ. Mirrors paemuri/brdoc.IsCNPJ.
func IsCNPJ(s string) bool { return brdoc.NewCNPJ().Validate(s) }

// IsCNH reports whether s is a valid CNH. Mirrors paemuri/brdoc.IsCNH.
func IsCNH(s string) bool { return brdoc.NewCNH().Validate(s) }

// IsPIS reports whether s is a valid PIS/PASEP/NIS/NIT. Mirrors paemuri/brdoc.IsPIS.
func IsPIS(s string) bool { return brdoc.NewPIS().Validate(s) }

// IsRENAVAM reports whether s is a valid RENAVAM. Mirrors paemuri/brdoc.IsRENAVAM.
func IsRENAVAM(s string) bool { return brdoc.NewRenavam().Validate(s) }

// IsVoterID reports whether s is a valid Título Eleitoral. Mirrors paemuri/brdoc.IsVoterID.
func IsVoterID(s string) bool { return brdoc.NewVoterID().Validate(s) }

// IsCNS reports whether s is a valid CNS (health card). Mirrors paemuri/brdoc.IsCNS.
func IsCNS(s string) bool { return brdoc.NewCNS().Validate(s) }

// IsPlate reports whether s is a valid vehicle plate (national OR Mercosul).
// Mirrors paemuri/brdoc.IsPlate.
func IsPlate(s string) bool { return brdoc.NewPlate().Validate(s) }

// IsNationalPlate reports whether s is a valid legacy national plate (ABC-1234).
// Mirrors paemuri/brdoc.IsNationalPlate.
func IsNationalPlate(s string) bool { return brdoc.NewPlate().ValidateNational(s) }

// IsMercosulPlate reports whether s is a valid Mercosul plate (ABC1D23).
// Mirrors paemuri/brdoc.IsMercosulPlate.
func IsMercosulPlate(s string) bool { return brdoc.NewPlate().ValidateMercosul(s) }

// IsCEP reports whether s is a valid CEP and, if so, the UF it maps to.
// On invalid input it returns (false, ""). Mirrors paemuri/brdoc.IsCEP.
func IsCEP(s string) (bool, UF) {
	c := brdoc.NewCEP()
	if !c.Validate(s) {
		return false, UF("")
	}
	origin, err := c.Origin(s)
	if err != nil {
		return false, UF("")
	}
	return true, UF(origin)
}

// IsCEPFrom reports whether s is a valid CEP whose UF is one of ufs.
// With no ufs it behaves like the bool part of IsCEP. Mirrors paemuri/brdoc.IsCEPFrom.
func IsCEPFrom(s string, ufs ...UF) bool {
	ok, uf := IsCEP(s)
	if !ok {
		return false
	}
	if len(ufs) == 0 {
		return true
	}
	for _, want := range ufs {
		if uf == want {
			return true
		}
	}
	return false
}

// IsPhone reports whether s is a valid Brazilian phone number and, if so, the
// UF its DDD maps to. On invalid input it returns (false, ""). Mirrors
// paemuri/brdoc.IsPhone.
func IsPhone(s string) (bool, UF) {
	p := brdoc.NewPhone()
	if !p.Validate(s) {
		return false, UF("")
	}
	origin, err := p.Origin(s)
	if err != nil {
		return false, UF("")
	}
	return true, UF(origin)
}

// IsPhoneFrom reports whether s is a valid phone whose UF is one of ufs.
// With no ufs it behaves like the bool part of IsPhone. Mirrors
// paemuri/brdoc.IsPhoneFrom.
func IsPhoneFrom(s string, ufs ...UF) bool {
	ok, uf := IsPhone(s)
	if !ok {
		return false
	}
	if len(ufs) == 0 {
		return true
	}
	for _, want := range ufs {
		if uf == want {
			return true
		}
	}
	return false
}

// IsRG reports whether s is a valid RG for the given UF. The returned error is
// non-nil (wrapping brdoc.ErrUFNotImplemented) when uf has no implemented
// algorithm; a well-formed but invalid RG returns (false, nil).
// Mirrors paemuri/brdoc.IsRG.
func IsRG(s string, uf UF) (bool, error) {
	return brdoc.NewRG().ValidateUF(s, uf)
}
