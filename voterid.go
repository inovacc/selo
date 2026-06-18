package selo

import (
	"fmt"
	"math/rand/v2"
)

func init() {
	Register(NewVoterID())
}

// VoterIDLength is the canonical digit count for a Título Eleitoral.
const VoterIDLength = 12

var (
	voterWeightsDV1 = [8]int{2, 3, 4, 5, 6, 7, 8, 9}
	voterWeightsDV2 = [3]int{7, 8, 9}
)

// VoterID validates, generates, and resolves the origin of a Brazilian
// Título Eleitoral (voter registration). Layout: SSSSSSSS UU D1 D2
// (8 sequence digits + 2 UF code digits + 2 check digits).
type VoterID struct{}

// NewVoterID returns a VoterID document handler.
func NewVoterID() *VoterID { return &VoterID{} }

// Kind reports the document kind.
func (v *VoterID) Kind() Kind { return KindVoterID }

// Validate reports whether value is a well-formed Título Eleitoral.
func (v *VoterID) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != VoterIDLength {
		return false
	}

	if allEqualBytes(d) {
		return false
	}

	ufCode := int(d[8]-'0')*10 + int(d[9]-'0')
	if ufCode < 1 || ufCode > 28 {
		return false
	}

	dv1 := voterDV1(d)
	dv2 := voterDV2(d, dv1)

	return dv1 == int(d[10]-'0') && dv2 == int(d[11]-'0')
}

// voterDV1 computes the first check digit over the 8 sequence digits.
func voterDV1(d string) int {
	sum := 0
	for i := 0; i < 8; i++ {
		sum += int(d[i]-'0') * voterWeightsDV1[i]
	}

	mod := sum % 11
	if mod == 10 || mod == 11 {
		return 0
	}

	return mod
}

// voterDV2 computes the second check digit over the 2 UF digits plus dv1.
func voterDV2(d string, dv1 int) int {
	vals := [3]int{int(d[8] - '0'), int(d[9] - '0'), dv1}

	sum := 0
	for i := 0; i < 3; i++ {
		sum += vals[i] * voterWeightsDV2[i]
	}

	mod := sum % 11
	if mod == 10 || mod == 11 {
		return 0
	}

	return mod
}

// allEqualBytes reports whether every byte in s is identical (and s is non-empty).
func allEqualBytes(s string) bool {
	if len(s) == 0 {
		return false
	}

	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}

	return true
}

// Generate returns a syntactically valid random Título Eleitoral (12 digits).
func (v *VoterID) Generate() string {
	for {
		var d [VoterIDLength]byte

		for i := 0; i < 8; i++ {
			d[i] = byte('0' + rand.IntN(10))
		}

		uf := rand.IntN(28) + 1 // 1..28
		d[8] = byte('0' + uf/10)
		d[9] = byte('0' + uf%10)

		s := string(d[:10])
		dv1 := voterDV1(s)
		d[10] = byte('0' + dv1)
		d[11] = byte('0' + voterDV2(s, dv1))

		out := string(d[:])
		if !allEqualBytes(out) {
			return out
		}
	}
}

// Format returns the voter ID grouped as "SSSS SSSS UUDD".
func (v *VoterID) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != VoterIDLength {
		return "", fmt.Errorf("voter ID must have %d digits, got %d: %w", VoterIDLength, len(d), ErrInvalidLength)
	}

	return d[0:4] + " " + d[4:8] + " " + d[8:12], nil
}

// voterUFNames maps the TSE 2-digit UF code (01..28) to a state/region name.
var voterUFNames = map[int]string{
	1: "São Paulo", 2: "Minas Gerais", 3: "Rio de Janeiro", 4: "Rio Grande do Sul",
	5: "Bahia", 6: "Paraná", 7: "Ceará", 8: "Pernambuco", 9: "Santa Catarina",
	10: "Goiás", 11: "Maranhão", 12: "Paraíba", 13: "Pará", 14: "Espírito Santo",
	15: "Piauí", 16: "Rio Grande do Norte", 17: "Alagoas", 18: "Mato Grosso",
	19: "Mato Grosso do Sul", 20: "Distrito Federal", 21: "Sergipe", 22: "Amazonas",
	23: "Rondônia", 24: "Acre", 25: "Amapá", 26: "Roraima", 27: "Tocantins",
	28: "Exterior",
}

// Origin returns the state/region encoded in the voter ID's UF code.
// It implements OriginResolver.
func (v *VoterID) Origin(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != VoterIDLength {
		return "", fmt.Errorf("voter ID must have %d digits, got %d: %w", VoterIDLength, len(d), ErrInvalidLength)
	}

	ufCode := int(d[8]-'0')*10 + int(d[9]-'0')

	name, ok := voterUFNames[ufCode]
	if !ok {
		return "", fmt.Errorf("voter ID UF code %02d unknown: %w", ufCode, ErrInvalidFormat)
	}

	return name, nil
}
