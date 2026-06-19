package selo

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
)

const (
	CpfLength = 11

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

func init() {
	// Initialize non-accepted CPFs (all digits equal)
	notAcceptedCPF = make([]string, 0, 10)

	for i := range 10 {
		value := strings.Repeat(strconv.Itoa(i), 11)
		notAcceptedCPF = append(notAcceptedCPF, value)
	}

	// Self-register CPF as Document singleton.
	Register(&CPF{})
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
// Utility Functions
// ============================================================================

// ValidateDocument automatically identifies and validates a CPF or CNPJ.
//
// Deprecated: use Detect to identify the Kind and Validate(kind, value) to
// validate, e.g. k, ok := selo.Detect(s); the typed Kind ("cpf"/"cnpj") is
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
