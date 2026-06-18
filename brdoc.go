package brdoc

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
)

const (
	CpfLength  = 11
	CnpjLength = 14

	IsDigit0 = "Rio Grande do Sul"
	IsDigit1 = "Federal District, Goiás, Mato Grosso do Sul, and Tocantins"
	IsDigit2 = "Pará, Amazonas, Acre, Amapá, Rondônia, and Roraima"
	IsDigit3 = "Ceará, Maranhão, and Piauí"
	IsDigit4 = "Pernambuco, Rio Grande do Norte, Paraíba, and Alagoas"
	IsDigit5 = "Bahia and Sergipe"
	IsDigit6 = "Minas Gerais"
	IsDigit7 = "Rio de Janeiro and Espírito Santo"
	IsDigit8 = "São Paulo"
	IsDigit9 = "Paraná and Santa Catarina"
)

var notAcceptedCPF []string

// Conversion map for alphanumeric CNPJ (ASCII - 48)
var charToValue = map[rune]int{
	'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
	'A': 17, 'B': 18, 'C': 19, 'D': 20, 'E': 21, 'F': 22, 'G': 23, 'H': 24, 'I': 25,
	'J': 26, 'K': 27, 'L': 28, 'M': 29, 'N': 30, 'O': 31, 'P': 32, 'Q': 33, 'R': 34,
	'S': 35, 'T': 36, 'U': 37, 'V': 38, 'W': 39, 'X': 40, 'Y': 41, 'Z': 42,
}

func init() {
	// Initialize non-accepted CPFs (all digits equal)
	notAcceptedCPF = make([]string, 0, 10)

	for i := range 10 {
		value := strings.Repeat(strconv.Itoa(i), 11)
		notAcceptedCPF = append(notAcceptedCPF, value)
	}

	// Self-register CPF and CNPJ as Document singletons.
	Register(&CPF{})
	Register(&CNPJ{})
}

// ============================================================================
// CPF - Individual Taxpayer Registry
// ============================================================================

// CPF represents a Brazilian individual tax ID validator
type CPF struct {
	cpfNumber []int
}

// NewCPF creates a new CPF validator instance
func NewCPF() *CPF {
	return &CPF{}
}

// Generate generates a valid random CPF with unformatting
func (c *CPF) Generate() string {
	number := []int{0, 0, 0, 0, 0, 0, 0, 0, 0}

	for i := range 9 {
		number[i] = rand.IntN(10)
	}

	number = append(number, c.calculateFirstDigit(number))
	number = append(number, c.calculateSecondDigit(number))

	var sb strings.Builder

	for _, item := range number {
		sb.WriteString(strconv.Itoa(item))
	}

	return c.digits(sb.String())
}

// Validate validates a CPF number (with or without formatting)
func (c *CPF) Validate(value string) bool {
	c.clean(value)

	return c.isAccepted(value) && c.length(c.cpfNumber) && c.validate(c.cpfNumber)
}

// Format formats a CPF string to the standard format XXX.XXX.XXX-XX
func (c *CPF) Format(value string) (string, error) {
	c.clean(value)

	if !c.isAccepted(value) {
		return "", fmt.Errorf("CPF is not valid")
	}

	if len(c.cpfNumber) != CpfLength {
		return "", fmt.Errorf("CPF must have %d digits, got: %d", CpfLength, len(c.cpfNumber))
	}

	return c.maskCPF(c.cpfNumber), nil
}

// CheckOrigin returns the Brazilian state/region where the CPF was issued
// based on the 9th digit
func (c *CPF) CheckOrigin(value string) string {
	c.clean(value)

	if len(c.cpfNumber) < 9 {
		return ""
	}

	switch c.cpfNumber[8] {
	case 0:
		return IsDigit0
	case 1:
		return IsDigit1
	case 2:
		return IsDigit2
	case 3:
		return IsDigit3
	case 4:
		return IsDigit4
	case 5:
		return IsDigit5
	case 6:
		return IsDigit6
	case 7:
		return IsDigit7
	case 8:
		return IsDigit8
	case 9:
		return IsDigit9
	default:
		return ""
	}
}

// Kind returns KindCPF, identifying this type in the registry.
func (c *CPF) Kind() Kind { return KindCPF }

// Origin returns the issuing region for value, satisfying OriginResolver.
// It wraps CheckOrigin; an empty origin (value too short) yields ErrInvalidLength.
func (c *CPF) Origin(value string) (string, error) {
	origin := c.CheckOrigin(value)
	if origin == "" {
		return "", ErrInvalidLength
	}
	return origin, nil
}

// Private CPF methods

func (c *CPF) maskCPF(value []int) string {
	// Build formatted CPF directly into a 14-byte buffer: XXX.XXX.XXX-XX
	var out [14]byte

	// Map digits into positions
	out[3], out[7], out[11] = '.', '.', '-'
	out[0] = byte('0' + value[0])
	out[1] = byte('0' + value[1])
	out[2] = byte('0' + value[2])
	out[4] = byte('0' + value[3])
	out[5] = byte('0' + value[4])
	out[6] = byte('0' + value[5])
	out[8] = byte('0' + value[6])
	out[9] = byte('0' + value[7])
	out[10] = byte('0' + value[8])
	out[12] = byte('0' + value[9])
	out[13] = byte('0' + value[10])

	return string(out[:])
}

func (c *CPF) clean(value string) {
	// Always reset and parse fresh to avoid stale state across calls
	c.cpfNumber = c.cpfNumber[:0]

	// Ensure we have the capacity to avoid reallocation across calls
	if cap(c.cpfNumber) < CpfLength {
		c.cpfNumber = make([]int, 0, CpfLength)
	}

	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch >= '0' && ch <= '9' {
			c.cpfNumber = append(c.cpfNumber, int(ch-'0'))
		}
	}
}

// isDigit checks if a character is a numeric digit
func (c *CPF) isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// processRemainingDigits handles the fallback when the buffer exceeds capacity
func (c *CPF) processRemainingDigits(value string, startIdx int, existingData []byte) string {
	result := make([]byte, 0, len(value))
	result = append(result, existingData...)

	for i := startIdx; i < len(value); i++ {
		if c.isDigit(value[i]) {
			result = append(result, value[i])
		}
	}

	return string(result)
}

func (c *CPF) digits(value string) string {
	// Fast filter to keep only digits; avoids regexp allocation per call
	var (
		buf [CpfLength]byte
		n   int
	)

	for i := 0; i < len(value); i++ {
		ch := value[i]
		if !c.isDigit(ch) {
			continue
		}

		if n >= len(buf) {
			// Fallback for unexpected longer inputs with many digits
			return c.processRemainingDigits(value, i, buf[:n])
		}

		buf[n] = ch
		n++
	}

	return string(buf[:n])
}

func (c *CPF) calculateFirstDigit(value []int) int {
	sum := 0
	for i, v := range value {
		sum += v * (10 - i)
	}

	rest := (sum * 10) % 11
	if rest == 10 || rest == 11 {
		rest = 0
	}

	return rest
}

func (c *CPF) calculateSecondDigit(value []int) int {
	sum := 0
	for i, v := range value {
		sum += v * (11 - i)
	}

	rest := (sum * 10) % 11
	if rest == 10 || rest == 11 {
		rest = 0
	}

	return rest
}

func (c *CPF) validate(value []int) bool {
	if len(value) != CpfLength {
		return false
	}

	// Calculate using base slices: first 9 for DV1, first 10 for DV2
	dv1 := c.calculateFirstDigit(value[:9])
	dv2 := c.calculateSecondDigit(append(value[:9], dv1))

	return dv1 == value[9] && dv2 == value[10]
}

func (c *CPF) isAccepted(value string) bool {
	// Reject CPFs with all equal digits
	return !slices.Contains(notAcceptedCPF, c.digits(value))
}

func (c *CPF) length(value []int) bool {
	return len(value) == CpfLength
}

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

// Generate generates a valid alphanumeric CNPJ
func (c *CNPJ) Generate() string {
	return c.generateDigits(false)
}

// GenerateLegacy generates a valid numeric-only (legacy) CNPJ
// It produces a 14-digit unformatted string where the first 12 positions are digits (0-9)
// and the last two are check digits per modulo 11.
func (c *CNPJ) GenerateLegacy() string {
	return c.generateDigits(true)
}

// Validate verifies if an alphanumeric CNPJ is valid per SERPRO specification
func (c *CNPJ) Validate(value string) bool {
	// Remove formatting
	cleaned := c.digits(value)

	if len(cleaned) != CnpjLength {
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

func (c *CNPJ) generateDigits(legacy bool) string {
	// Build a 12-char base directly into a fixed buffer
	var base [12]byte

	if legacy {
		for i := range 12 {
			base[i] = byte('0' + rand.IntN(10))
		}
	} else {
		for i := range 12 {
			if rand.IntN(2) == 0 {
				base[i] = byte('0' + rand.IntN(10))
			} else {
				base[i] = byte('A' + rand.IntN(26))
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

// ============================================================================
// Utility Functions
// ============================================================================

// ValidateDocument automatically identifies and validates a CPF or CNPJ.
//
// Deprecated: use Detect to identify the Kind and Validate(kind, value) to
// validate, e.g. k, ok := brdoc.Detect(s); the typed Kind ("cpf"/"cnpj") is
// richer than the legacy "CPF"/"CNPJ"/"UNKNOWN" strings. This wrapper will be
// removed after 2026-07-18.
func ValidateDocument(doc string) (docType string, isValid bool) {
	// Preserve legacy length-based labelling: an 11/14-length value keeps its
	// label even when invalid (parity with the original implementation).
	// CPF length is measured over digits only; CNPJ length over the
	// alphanumeric (0-9, A-Z) cleaned form, matching the legacy semantics.
	if len(onlyDigits(doc)) == CpfLength {
		ok, _ := Validate(KindCPF, doc)
		return "CPF", ok
	}
	if len((&CNPJ{}).digits(doc)) == CnpjLength {
		ok, _ := Validate(KindCNPJ, doc)
		return "CNPJ", ok
	}
	return "UNKNOWN", false
}
