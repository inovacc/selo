package brdoc

import "testing"

func TestOnlyDigits(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"123.456.789-09", "12345678909"},
		{"12.ABC.345/01DE-35", "123450135"},
		{"  +55 (11) 98888-7777 ", "5511988887777"},
		{"abc", ""},
		{"", ""},
		{"0001-10", "000110"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := onlyDigits(tt.in); got != tt.want {
				t.Fatalf("onlyDigits(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateDocument_DelegatesToDetect(t *testing.T) {
	// Wrapper must agree with Detect+Validate for known kinds.
	if dt, ok := ValidateDocument("123.456.789-09"); dt != "CPF" || !ok {
		t.Fatalf("ValidateDocument(valid CPF) = (%q, %v), want (CPF, true)", dt, ok)
	}
	if dt, ok := ValidateDocument("12.ABC.345/01DE-35"); dt != "CNPJ" || !ok {
		t.Fatalf("ValidateDocument(valid CNPJ) = (%q, %v), want (CNPJ, true)", dt, ok)
	}
	// Invalid-but-CPF-shaped input keeps the CPF label (parity with legacy).
	if dt, ok := ValidateDocument("123.456.789-00"); dt != "CPF" || ok {
		t.Fatalf("ValidateDocument(invalid CPF) = (%q, %v), want (CPF, false)", dt, ok)
	}
	if dt, ok := ValidateDocument("12345"); dt != "UNKNOWN" || ok {
		t.Fatalf("ValidateDocument(garbage) = (%q, %v), want (UNKNOWN, false)", dt, ok)
	}
}
