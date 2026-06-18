package selo

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// CPF Tests
// ============================================================================

func TestCPF_Generate(t *testing.T) {
	cpf := NewCPF()

	for range 10 {
		generated := cpf.Generate()

		assert.True(t, cpf.Validate(generated), "Generated CPF is invalid: %s", generated)

		_, _ = fmt.Fprintf(os.Stdout, "Generated CPF: %s | Origin: %s\n", generated, cpf.CheckOrigin(generated))
	}
}

func TestCPF_Validate(t *testing.T) {
	tests := []struct {
		name     string
		cpf      string
		expected bool
	}{
		{"Valid formatted CPF", "123.456.789-09", true},
		{"Valid unformatted CPF", "12345678909", true},
		{"Invalid CPF - wrong check digit", "123.456.789-00", false},
		{"Invalid CPF - all zeros", "000.000.000-00", false},
		{"Invalid CPF - all equal digits", "111.111.111-11", false},
		{"Invalid CPF - wrong length", "123.456.789", false},
	}

	cpf := NewCPF()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cpf.Validate(tt.cpf)
			assert.Equal(t, tt.expected, result, "Validate(%s)", tt.cpf)
		})
	}
}

func TestCPF_Format(t *testing.T) {
	cpf := NewCPF()

	input := "12345678909"
	expected := "123.456.789-09"

	result, err := cpf.Format(input)
	require.NoError(t, err)

	assert.Equal(t, expected, result, "Format(%s)", input)
}

func TestCPF_CheckOrigin(t *testing.T) {
	tests := []struct {
		cpf      string
		expected string
	}{
		{"123.456.780-09", IsDigit0},
		{"123.456.788-09", IsDigit8},
		{"123.456.789-09", IsDigit9},
	}

	cpf := NewCPF()
	for _, tt := range tests {
		t.Run(tt.cpf, func(t *testing.T) {
			result := cpf.CheckOrigin(tt.cpf)
			assert.Equal(t, tt.expected, result, "CheckOrigin(%s)", tt.cpf)
		})
	}
}

// Additional real-world valid CPF samples (provided) — all must validate
func TestCPF_Validate_ProvidedValid(t *testing.T) {
	samples := []string{
		"013.723.737-56",
		"260.808.754-03",
		"205.117.448-20",
		"213.872.640-10",
		"722.628.653-02",
		"747.356.416-10",
		"486.158.855-32",
	}

	cpf := NewCPF()

	for _, s := range samples {
		t.Run(s, func(t *testing.T) {
			require.True(t, cpf.Validate(s), "Expected provided CPF to be valid: %s", s)

			formatted, err := cpf.Format(s)
			require.NoError(t, err, "Unexpected formatting error for %s", s)

			require.True(t, cpf.Validate(formatted), "Formatted CPF should be valid: %s", formatted)
		})
	}
}

// Ensure unformatted variants of provided CPFs are also valid and format back correctly
func TestCPF_Format_ProvidedValid_Unformatted(t *testing.T) {
	strip := func(s string) string {
		out := make([]rune, 0, len(s))
		for _, r := range s {
			switch r {
			case '.', '-', ' ':
				// skip
			default:
				out = append(out, r)
			}
		}

		return string(out)
	}

	samples := []string{
		"013.723.737-56",
		"260.808.754-03",
		"205.117.448-20",
		"213.872.640-10",
		"722.628.653-02",
		"747.356.416-10",
		"486.158.855-32",
	}

	cpf := NewCPF()

	for _, formatted := range samples {
		unformatted := strip(formatted)
		t.Run(unformatted, func(t *testing.T) {
			require.True(t, cpf.Validate(unformatted), "Expected unformatted provided CPF to be valid: %s", unformatted)

			got, err := cpf.Format(unformatted)
			require.NoError(t, err, "Unexpected formatting error for %s", unformatted)

			assert.Equal(t, formatted, got, "Format(%s)", unformatted)
		})
	}
}

// ============================================================================
// Alphanumeric CNPJ Tests
// ============================================================================

func TestCNPJ_ValidateExampleFromPDF(t *testing.T) {
	cnpj := NewCNPJ()

	// Example from SERPRO documentation
	example := "12ABC34501DE35"

	assert.True(t, cnpj.Validate(example), "SERPRO example CNPJ should be valid: %s", example)

	// Also test with formatting
	formattedExample := "12.ABC.345/01DE-35"
	assert.True(t, cnpj.Validate(formattedExample), "Formatted CNPJ should be valid: %s", formattedExample)
}

// Additional real-world valid CNPJ samples (provided) — all must validate
func TestCNPJ_Validate_ProvidedValid(t *testing.T) {
	samples := []string{
		"HR.YUP.H8D/0001-02",
		"48.175.226/0001-50",
		"SE.URZ.76B/0001-02",
		"37.077.670/0001-16",
		"52.311.151/0001-64",
		"64.814.243/0001-46",
		"Z7.BM3.7VE/0001-93",
		"V2.P0M.NVE/0001-07",
	}

	cnpj := NewCNPJ()

	for _, s := range samples {
		t.Run(s, func(t *testing.T) {
			require.True(t, cnpj.Validate(s), "Expected provided CNPJ to be valid: %s", s)

			// Round-trip formatting should keep the same visual representation (uppercase)
			formatted, err := cnpj.Format(s)
			require.NoError(t, err, "Unexpected formatting error for %s", s)

			// Validate formatted too
			require.True(t, cnpj.Validate(formatted), "Formatted CNPJ should be valid: %s", formatted)
		})
	}
}

// Ensure unformatted variants of provided CNPJs are also valid and format back correctly
func TestCNPJ_Format_ProvidedValid_Unformatted(t *testing.T) {
	strip := func(s string) string {
		// remove formatting characters .-/ and spaces
		out := make([]rune, 0, len(s))
		for _, r := range s {
			switch r {
			case '.', '-', '/', ' ':
				// skip
			default:
				out = append(out, r)
			}
		}

		return string(out)
	}

	samples := []string{
		"HR.YUP.H8D/0001-02",
		"48.175.226/0001-50",
		"SE.URZ.76B/0001-02",
		"37.077.670/0001-16",
		"52.311.151/0001-64",
		"64.814.243/0001-46",
		"Z7.BM3.7VE/0001-93",
		"V2.P0M.NVE/0001-07",
	}

	cnpj := NewCNPJ()

	for _, formatted := range samples {
		unformatted := strip(formatted)
		t.Run(unformatted, func(t *testing.T) {
			require.True(t, cnpj.Validate(unformatted), "Expected unformatted provided CNPJ to be valid: %s", unformatted)

			got, err := cnpj.Format(unformatted)
			require.NoError(t, err, "Unexpected formatting error for %s", unformatted)

			// The formatter normalizes to uppercase and standard mask; compare after normalizing expected to uppercase
			expected := strings.ToUpper(formatted)
			assert.Equal(t, expected, got, "Format(%s)", unformatted)
		})
	}
}

func TestCNPJ_Generate(t *testing.T) {
	cnpj := NewCNPJ()

	for range 10 {
		generated := cnpj.Generate()

		assert.True(t, cnpj.Validate(generated), "Generated CNPJ is invalid: %s", generated)

		formatted, err := cnpj.Format(generated)
		assert.NoError(t, err)

		_, _ = fmt.Fprintf(os.Stdout, "Generated CNPJ: %s | Formatted: %s\n", generated, formatted)
	}
}

func TestCNPJ_GenerateLegacy(t *testing.T) {
	cnpj := NewCNPJ()
	for range 10 {
		generated := cnpj.GenerateLegacy()
		require.Len(t, generated, CnpjLength)

		// ensure numeric-only
		for _, r := range generated {
			require.True(t, r >= '0' && r <= '9', "expected numeric-only, got %q", generated)
		}

		// legacy must also validate with the alphanumeric validator
		assert.True(t, cnpj.Validate(generated), "Generated CNPJ is invalid: %s", generated)

		// formatting should work
		formatted, err := cnpj.Format(generated)
		require.NoError(t, err)
		require.Len(t, formatted, 18) // mask length: 18 with separators

		_, _ = fmt.Fprintf(os.Stdout, "Generated CNPJ: %s | Formatted: %s\n", generated, formatted)
	}
}

func TestCNPJ_Validate(t *testing.T) {
	tests := []struct {
		name     string
		cnpj     string
		expected bool
	}{
		{"Valid CNPJ - SERPRO example", "12ABC34501DE35", true},
		{"Valid formatted CNPJ", "12.ABC.345/01DE-35", true},
		{"Invalid CNPJ - wrong check digits", "12ABC34501DE00", false},
		{"Invalid CNPJ - wrong length", "12ABC345", false},
		{"Invalid CNPJ - non-numeric check digits", "12ABC34501DEAA", false},
	}

	cnpj := NewCNPJ()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cnpj.Validate(tt.cnpj)
			assert.Equal(t, tt.expected, result, "Validate(%s)", tt.cnpj)
		})
	}
}

func TestCNPJ_Format(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			"Valid CNPJ without formatting",
			"12ABC34501DE35",
			"12.ABC.345/01DE-35",
			false,
		},
		{
			"CNPJ already formatted",
			"12.ABC.345/01DE-35",
			"12.ABC.345/01DE-35",
			false,
		},
		{
			"CNPJ invalid length",
			"12ABC345",
			"",
			true,
		},
	}

	cnpj := NewCNPJ()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cnpj.Format(tt.input)

			if tt.hasError {
				assert.Error(t, err, "Expected error for input: %s", tt.input)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result, "Format(%s)", tt.input)
			}
		})
	}
}

func TestCNPJ_CalculateDV_Manual(t *testing.T) {
	cnpj := NewCNPJ()

	// Manual test of SERPRO example: 12ABC34501DE
	base := "12ABC34501DE"

	dv1, err := cnpj.calculateDV(base)
	require.NoError(t, err)
	assert.Equal(t, 3, dv1, "DV1 calculated")

	dv2, err := cnpj.calculateDV(base + "3")
	require.NoError(t, err)
	assert.Equal(t, 5, dv2, "DV2 calculated")

	_, _ = fmt.Fprintf(os.Stdout, "✓ Check digits calculated correctly: %d%d\n", dv1, dv2)
}

// ============================================================================
// Utility Functions Tests
// ============================================================================

func TestValidateDocument(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		docType string
		isValid bool
	}{
		{"Valid CPF", "123.456.789-09", "CPF", true},
		{"Valid CNPJ", "12.ABC.345/01DE-35", "CNPJ", true},
		{"Invalid CPF", "123.456.789-00", "CPF", false},
		{"Invalid CNPJ", "12.ABC.345/01DE-00", "CNPJ", false},
		{"Unknown document", "12345", "UNKNOWN", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docType, isValid := ValidateDocument(tt.doc)

			assert.Equal(t, tt.docType, docType, "doc type for %s", tt.doc)
			assert.Equal(t, tt.isValid, isValid, "validation for %s", tt.doc)
		})
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkCPF_Generate(b *testing.B) {
	cpf := NewCPF()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = cpf.Generate()
	}
}

func BenchmarkCPF_Validate(b *testing.B) {
	cpf := NewCPF()

	testCPF := "123.456.789-09"

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = cpf.Validate(testCPF)
	}
}

func BenchmarkCNPJ_Generate(b *testing.B) {
	cnpj := NewCNPJ()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = cnpj.Generate()
	}
}

func BenchmarkCNPJ_Validate(b *testing.B) {
	cnpj := NewCNPJ()

	testCNPJ := "12.ABC.345/01DE-35"

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = cnpj.Validate(testCNPJ)
	}
}
