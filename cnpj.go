package selo

import (
	"fmt"
	"math/rand/v2"
	"strconv"
)

// CnpjLength is the number of characters in a CNPJ.
const CnpjLength = 14

// charToValue maps alphanumeric CNPJ characters to their numeric weights (SERPRO).
// Conversion map for alphanumeric CNPJ (ASCII - 48)
var charToValue = map[rune]int{
	'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
	'A': 17, 'B': 18, 'C': 19, 'D': 20, 'E': 21, 'F': 22, 'G': 23, 'H': 24, 'I': 25,
	'J': 26, 'K': 27, 'L': 28, 'M': 29, 'N': 30, 'O': 31, 'P': 32, 'Q': 33, 'R': 34,
	'S': 35, 'T': 36, 'U': 37, 'V': 38, 'W': 39, 'X': 40, 'Y': 41, 'Z': 42,
}

func init() { Register(&CNPJ{}) }

// ============================================================================
// CNPJ - National Registry of Legal Entities (Alphanumeric)
// Based on the SERPRO specification
// ============================================================================

// CNPJ represents a Brazilian company tax ID validator (alphanumeric format)
type CNPJ struct{}

// NewCNPJ creates a new CNPJ validator instance
func NewCNPJ() *CNPJ {
	return &CNPJ{}
}

// GenerateRand generates a valid alphanumeric CNPJ using the supplied random source.
func (c *CNPJ) GenerateRand(r *rand.Rand) string {
	return c.generateDigitsRand(r, false)
}

// Generate generates a valid alphanumeric CNPJ
func (c *CNPJ) Generate() string { return c.GenerateRand(newRand()) }

// GenerateLegacy generates a valid numeric-only (legacy) CNPJ
// It produces a 14-digit unformatted string where the first 12 positions are digits (0-9)
// and the last two are check digits per modulo 11.
func (c *CNPJ) GenerateLegacy() string { return c.generateDigitsRand(newRand(), true) }

// Validate verifies if an alphanumeric CNPJ is valid per SERPRO specification
func (c *CNPJ) Validate(value string) bool {
	// Remove formatting
	cleaned := c.digits(value)

	if len(cleaned) != CnpjLength {
		return false
	}

	// Reject all-equal inputs (e.g. "00000000000000"); never a real CNPJ.
	if allEqualBytes(cleaned) {
		return false
	}

	// Ensure the last 2 characters are numeric
	ch12 := cleaned[12]
	if ch12 < '0' || ch12 > '9' {
		return false
	}

	dv1 := int(ch12 - '0')

	ch13 := cleaned[13]
	if ch13 < '0' || ch13 > '9' {
		return false
	}

	dv2 := int(ch13 - '0')

	base := cleaned[:12]

	dv1Calc, err := c.calculateDV(base)
	if err != nil {
		return false
	}

	dv2Calc, err := c.calculateDV(base + strconv.Itoa(dv1Calc))
	if err != nil {
		return false
	}

	return dv1Calc == dv1 && dv2Calc == dv2
}

// Format formats a CNPJ to the standard format XX.XXX.XXX/XXXX-XX
func (c *CNPJ) Format(value string) (string, error) {
	cleaned := c.digits(value)

	if len(cleaned) != CnpjLength {
		return "", fmt.Errorf("CNPJ must have 14 characters, got: %d", len(cleaned))
	}

	// Build formatted CNPJ directly into an 18-byte buffer: XX.XXX.XXX/XXXX-XX
	var out [18]byte

	out[2], out[6], out[10], out[15] = '.', '.', '/', '-'

	// Copy pieces
	copy(out[0:2], cleaned[0:2])
	copy(out[3:6], cleaned[2:5])
	copy(out[7:10], cleaned[5:8])
	copy(out[11:15], cleaned[8:12])
	copy(out[16:18], cleaned[12:14])

	return string(out[:]), nil
}

// Kind returns KindCNPJ, identifying this type in the registry.
func (c *CNPJ) Kind() Kind { return KindCNPJ }

// Private CNPJ methods

func (c *CNPJ) generateDigitsRand(r *rand.Rand, legacy bool) string {
	// Build a 12-char base directly into a fixed buffer
	var base [12]byte

	if legacy {
		for i := range 12 {
			base[i] = byte('0' + r.IntN(10))
		}
	} else {
		for i := range 12 {
			if r.IntN(2) == 0 {
				base[i] = byte('0' + r.IntN(10))
			} else {
				base[i] = byte('A' + r.IntN(26))
			}
		}
	}

	cnpjBase := string(base[:])

	// Calculate the two check digits
	dv1, err := c.calculateDV(cnpjBase)
	if err != nil {
		return ""
	}

	dv2, err := c.calculateDV(cnpjBase + strconv.Itoa(dv1))
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s%d%d", cnpjBase, dv1, dv2)
}

// calculateDV calculates a check digit using modulo 11
// Official SERPRO algorithm for alphanumeric CNPJ
func (c *CNPJ) calculateDV(value string) (int, error) {
	weights := []int{2, 3, 4, 5, 6, 7, 8, 9}
	sum := 0
	j := 0

	// Iterate the CNPJ from right to left applying the weights
	for i := len(value) - 1; i >= 0; i-- {
		val, ok := charToValue[rune(value[i])]
		if !ok {
			return 0, fmt.Errorf("invalid character: %c at position %d", value[i], i)
		}

		sum += val * weights[j]
		j = (j + 1) % len(weights) // Restart weights after the 8th element
	}

	remainder := sum % 11

	// Specific rule: if remainder = 0 or 1, DV = 0
	if remainder == 0 || remainder == 1 {
		return 0, nil
	}

	return 11 - remainder, nil
}

// normalizeChar converts lowercase to uppercase and validates alphanumeric characters
func (c *CNPJ) normalizeChar(ch byte) (byte, bool) {
	if ch >= 'a' && ch <= 'z' {
		return ch - 'a' + 'A', true
	}

	if (ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'Z') {
		return ch, true
	}

	return 0, false
}

// processRemainingChars handles the fallback when buffer exceeds capacity
func (c *CNPJ) processRemainingChars(value string, startIdx int, existingData []byte) string {
	result := make([]byte, 0, len(value))
	result = append(result, existingData...)

	for i := startIdx; i < len(value); i++ {
		if normalized, ok := c.normalizeChar(value[i]); ok {
			result = append(result, normalized)
		}
	}

	return string(result)
}

func (c *CNPJ) digits(value string) string {
	// Fast path: uppercase letters and keep only 0-9 and A-Z
	var buf [CnpjLength]byte

	n := 0

	for i := 0; i < len(value); i++ {
		normalized, ok := c.normalizeChar(value[i])
		if !ok {
			continue
		}

		if n >= len(buf) {
			// Switch to fallback allocation for longer inputs
			return c.processRemainingChars(value, i, buf[:n])
		}

		buf[n] = normalized
		n++
	}

	return string(buf[:n])
}
