package brdoc

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
)

var (
	nationalPlateRE = regexp.MustCompile(`^[A-Z]{3}-?[0-9]{4}$`)
	mercosulPlateRE = regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z][0-9]{2}$`)
)

func init() {
	Register(&Plate{})
}

// Plate validates, generates, and formats Brazilian vehicle license plates.
// Mercosul steers Generate toward the Mercosul pattern; Validate accepts both.
type Plate struct {
	Mercosul bool
}

// NewPlate creates a new national-pattern Plate instance.
func NewPlate() *Plate { return &Plate{} }

// Kind returns KindPlate.
func (p *Plate) Kind() Kind { return KindPlate }

// IsNationalPlate reports whether value matches the legacy national pattern
// ABC-1234 / ABC1234 (case-insensitive).
func IsNationalPlate(value string) bool {
	return nationalPlateRE.MatchString(strings.ToUpper(strings.TrimSpace(value)))
}

// IsMercosulPlate reports whether value matches the Mercosul pattern ABC1D23
// (case-insensitive).
func IsMercosulPlate(value string) bool {
	return mercosulPlateRE.MatchString(strings.ToUpper(strings.TrimSpace(value)))
}

// IsPlate reports whether value is either a national or a Mercosul plate.
func IsPlate(value string) bool {
	return IsNationalPlate(value) || IsMercosulPlate(value)
}

// Validate reports whether value is a valid plate (national or Mercosul).
func (p *Plate) Validate(value string) bool {
	return IsPlate(value)
}

// ValidateNational reports whether value is a valid legacy national plate
// (ABC-1234 / ABC1234). It is a method form of IsNationalPlate consumed by the
// compat/ paemuri drop-in (Task MC-2).
func (p *Plate) ValidateNational(value string) bool {
	return IsNationalPlate(value)
}

// ValidateMercosul reports whether value is a valid Mercosul plate (ABC1D23).
// It is a method form of IsMercosulPlate consumed by the compat/ paemuri
// drop-in (Task MC-2).
func (p *Plate) ValidateMercosul(value string) bool {
	return IsMercosulPlate(value)
}

// Generate returns a random valid plate. When Mercosul is true it emits the
// ABC1D23 pattern; otherwise the national ABC1234 pattern (no dash).
func (p *Plate) Generate() string {
	var sb strings.Builder
	for i := 0; i < 3; i++ {
		sb.WriteByte(byte('A' + rand.IntN(26)))
	}
	if p.Mercosul {
		sb.WriteByte(byte('0' + rand.IntN(10)))
		sb.WriteByte(byte('A' + rand.IntN(26)))
		sb.WriteByte(byte('0' + rand.IntN(10)))
		sb.WriteByte(byte('0' + rand.IntN(10)))
		return sb.String()
	}
	for i := 0; i < 4; i++ {
		sb.WriteByte(byte('0' + rand.IntN(10)))
	}
	return sb.String()
}

// Format canonicalizes a plate: national plates gain the dash (ABC-1234),
// Mercosul plates are returned uppercased without a dash (ABC1D23). Returns
// ErrInvalidFormat when value is neither pattern.
func (p *Plate) Format(value string) (string, error) {
	v := strings.ToUpper(strings.TrimSpace(value))
	if mercosulPlateRE.MatchString(v) {
		return v, nil
	}
	if nationalPlateRE.MatchString(v) {
		v = strings.ReplaceAll(v, "-", "")
		return v[0:3] + "-" + v[3:7], nil
	}
	return "", fmt.Errorf("brdoc: %q is not a valid plate: %w", value, ErrInvalidFormat)
}
