package selo

import (
	"math/rand/v2"
	"strings"
)

func init() { Register(&Renavam{}) }

// RenavamLength is the canonical digit count of a RENAVAM number.
const RenavamLength = 11

// renavamWeights are the mod-11 weights applied to the first 10 digits.
var renavamWeights = [10]int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

// Renavam validates, generates and formats RENAVAM vehicle-registration numbers.
type Renavam struct{}

// NewRenavam returns a Renavam document handler.
func NewRenavam() *Renavam { return &Renavam{} }

// Kind reports the registry identifier for RENAVAM.
func (r *Renavam) Kind() Kind { return KindRenavam }

// Validate reports whether value is a well-formed 11-digit RENAVAM.
// It accepts unformatted input and rejects all-equal sequences.
func (r *Renavam) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != RenavamLength {
		return false
	}
	if renavamAllEqual(d) {
		return false
	}
	return int(d[10]-'0') == renavamCheckDigit(d)
}

// Generate returns a random, valid, 11-digit RENAVAM number.
// It uses math/rand/v2 top-level funcs (goroutine-safe) and rejects all-equal results.
func (r *Renavam) Generate() string {
	for {
		var b [RenavamLength]byte
		for i := 0; i < 10; i++ {
			b[i] = byte('0' + rand.IntN(10))
		}
		b[10] = byte('0' + renavamCheckDigit(string(b[:10])))
		out := string(b[:])
		if !renavamAllEqual(out) {
			return out
		}
	}
}

// Format returns the canonical 11-digit RENAVAM string. RENAVAM has no separator
// mask; shorter inputs (legacy 9-digit forms) are left-padded with zeros to 11 digits.
func (r *Renavam) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) < RenavamLength {
		d = strings.Repeat("0", RenavamLength-len(d)) + d
	}
	return d, nil
}

// renavamCheckDigit computes the (sum*10)%11 check digit over the first 10 digits.
func renavamCheckDigit(d string) int {
	sum := 0
	for i := 0; i < 10; i++ {
		sum += int(d[i]-'0') * renavamWeights[i]
	}
	dv := (sum * 10) % 11
	if dv == 10 {
		dv = 0
	}
	return dv
}

// renavamAllEqual reports whether every byte of d is identical.
func renavamAllEqual(d string) bool {
	for i := 1; i < len(d); i++ {
		if d[i] != d[0] {
			return false
		}
	}
	return true
}
