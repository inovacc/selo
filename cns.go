package selo

import (
	"fmt"
	"math/rand/v2"
)

func init() {
	Register(NewCNS())
}

// CNSLength is the canonical digit count for a Cartão Nacional de Saúde.
const CNSLength = 15

// CNS validates and generates a Brazilian Cartão Nacional de Saúde
// (national health-card number). Definitive cards begin with 1 or 2;
// provisional cards begin with 7, 8, or 9.
type CNS struct{}

// NewCNS returns a CNS document handler.
func NewCNS() *CNS { return &CNS{} }

// Kind reports the document kind.
func (c *CNS) Kind() Kind { return KindCNS }

// Validate reports whether value is a well-formed CNS.
func (c *CNS) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != CNSLength {
		return false
	}

	if allEqualBytes(d) {
		return false
	}

	if !cnsValidPrefix(d[0]) {
		return false
	}

	return cnsWeightedSum(d)%11 == 0
}

// Generate returns a syntactically valid random CNS (15 digits, sum % 11 == 0).
func (c *CNS) Generate() string {
	prefixes := [5]byte{'1', '2', '7', '8', '9'}

	for {
		var d [CNSLength]byte

		d[0] = prefixes[rand.IntN(len(prefixes))]

		for i := 1; i < CNSLength-1; i++ {
			d[i] = byte('0' + rand.IntN(10))
		}

		// Sum of the first 14 positions (weights 15..2); position 15 has weight 1.
		partial := 0
		for i := range CNSLength - 1 {
			partial += int(d[i]-'0') * (CNSLength - i)
		}

		last := (11 - (partial % 11)) % 11
		if last == 10 {
			// Not solvable with a single digit; retry with new digits.
			continue
		}

		d[CNSLength-1] = byte('0' + last)

		out := string(d[:])
		if !allEqualBytes(out) {
			return out
		}
	}
}

// Format returns the cleaned 15-digit CNS (identity; CNS has no official mask).
func (c *CNS) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != CNSLength {
		return "", fmt.Errorf("CNS must have %d digits, got %d: %w", CNSLength, len(d), ErrInvalidLength)
	}

	return d, nil
}

// cnsValidPrefix reports whether the leading byte denotes a valid CNS class.
func cnsValidPrefix(b byte) bool {
	switch b {
	case '1', '2', '7', '8', '9':
		return true
	default:
		return false
	}
}

// cnsWeightedSum computes Σ dᵢ·wᵢ with descending weights 15..1.
func cnsWeightedSum(d string) int {
	sum := 0
	for i := range CNSLength {
		sum += int(d[i]-'0') * (CNSLength - i)
	}

	return sum
}
