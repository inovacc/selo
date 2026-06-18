package brdoc

import (
	"fmt"
	"math/rand/v2"
)

func init() { Register(&CNH{}) }

// CnhLength is the canonical digit count of a CNH number.
const CnhLength = 11

// CNH validates, generates and formats Carteira Nacional de Habilitação
// (Brazilian driver's-license) numbers. It uses two mod-11 check digits with a
// -2 base offset carried from DV1 into DV2.
type CNH struct{}

// NewCNH returns a CNH document handler.
func NewCNH() *CNH { return &CNH{} }

// Kind reports the registry identifier for CNH.
func (c *CNH) Kind() Kind { return KindCNH }

// Validate reports whether value is a well-formed 11-digit CNH number.
// It accepts unformatted input and rejects all-equal sequences.
func (c *CNH) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != CnhLength {
		return false
	}
	if cnhAllEqual(d) {
		return false
	}
	dv1, dv2 := cnhCheckDigits(d[:9])
	return dv1 == int(d[9]-'0') && dv2 == int(d[10]-'0')
}

// Generate returns a random, valid, 11-digit CNH number.
// It uses math/rand/v2 top-level funcs (goroutine-safe) and rejects all-equal results.
func (c *CNH) Generate() string {
	for {
		var b [CnhLength]byte
		for i := 0; i < 9; i++ {
			b[i] = byte('0' + rand.IntN(10))
		}
		dv1, dv2 := cnhCheckDigits(string(b[:9]))
		b[9] = byte('0' + dv1)
		b[10] = byte('0' + dv2)
		out := string(b[:])
		if !cnhAllEqual(out) {
			return out
		}
	}
}

// Format returns the cleaned 11-digit CNH string (CNH has no official separator mask).
// It returns ErrInvalidLength (wrapped with %w) when value has the wrong digit count.
func (c *CNH) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != CnhLength {
		return "", fmt.Errorf("cnh: got %d digits, want %d: %w", len(d), CnhLength, ErrInvalidLength)
	}
	return d, nil
}

// cnhCheckDigits computes both check digits over the 9-digit base.
// DV1 uses descending weights 9..1; if its raw remainder is >= 10 the digit is 0
// and an offset (dsc=2) is carried into DV2. DV2 uses ascending weights 1..9
// with the carried offset subtracted before the mod-11 fold.
func cnhCheckDigits(base string) (dv1, dv2 int) {
	dsc := 0

	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(base[i]-'0') * (9 - i)
	}
	r := sum % 11
	if r >= 10 {
		dv1 = 0
		dsc = 2
	} else {
		dv1 = r
	}

	sum = 0
	for i := 0; i < 9; i++ {
		sum += int(base[i]-'0') * (1 + i)
	}
	r = (sum % 11) - dsc
	if r < 0 {
		r += 11
	}
	if r >= 10 {
		dv2 = 0
	} else {
		dv2 = r
	}
	return dv1, dv2
}

// cnhAllEqual reports whether every byte of d is identical.
func cnhAllEqual(d string) bool {
	for i := 1; i < len(d); i++ {
		if d[i] != d[0] {
			return false
		}
	}
	return true
}
