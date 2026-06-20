package selo

import (
	"fmt"
	"math/rand/v2"
)

func init() { Register(&PIS{}) }

// PisLength is the canonical digit count of a PIS/PASEP/NIS number.
const PisLength = 11

// pisWeights are the fixed mod-11 weights applied to the first 10 digits.
var pisWeights = [10]int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

// PIS validates, generates and formats PIS/PASEP/NIS/NIT numbers,
// which share a single mod-11 check-digit algorithm.
type PIS struct{}

// NewPIS returns a PIS document handler.
func NewPIS() *PIS { return &PIS{} }

// Kind reports the registry identifier for PIS.
func (p *PIS) Kind() Kind { return KindPIS }

// Validate reports whether value is a well-formed PIS/PASEP/NIS number.
// It accepts formatted or unformatted input and rejects all-equal sequences.
func (p *PIS) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != PisLength {
		return false
	}

	if pisAllEqual(d) {
		return false
	}

	return int(d[10]-'0') == pisCheckDigit(d)
}

// GenerateRand returns a valid unformatted PIS number using the supplied random source.
func (p *PIS) GenerateRand(r *rand.Rand) string {
	for {
		var b [PisLength]byte
		for i := range 10 {
			b[i] = byte('0' + r.IntN(10))
		}

		b[10] = byte('0' + pisCheckDigit(string(b[:10])))

		out := string(b[:])
		if !pisAllEqual(out) {
			return out
		}
	}
}

// Generate returns a random, valid, unformatted PIS number.
// It uses math/rand/v2 top-level funcs (goroutine-safe) and rejects all-equal results.
func (p *PIS) Generate() string { return p.GenerateRand(newRand()) }

// Format renders a PIS number with the canonical ###.#####.##-# mask.
// It returns ErrInvalidLength (wrapped with %w) when value has the wrong digit count.
func (p *PIS) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != PisLength {
		return "", fmt.Errorf("pis: got %d digits, want %d: %w", len(d), PisLength, ErrInvalidLength)
	}

	return d[0:3] + "." + d[3:8] + "." + d[8:10] + "-" + d[10:11], nil
}

// pisCheckDigit computes the single mod-11 check digit over the first 10 digits.
func pisCheckDigit(d string) int {
	sum := 0
	for i := range 10 {
		sum += int(d[i]-'0') * pisWeights[i]
	}

	mod := sum % 11
	if mod <= 1 {
		return 0
	}

	return 11 - mod
}

// pisAllEqual reports whether every byte of d is identical (e.g. "11111111111").
func pisAllEqual(d string) bool {
	for i := 1; i < len(d); i++ {
		if d[i] != d[0] {
			return false
		}
	}

	return true
}
