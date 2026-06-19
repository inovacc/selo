package selo

import "testing"

func TestCNPJ_Registered(t *testing.T) {
	d, ok := Get(KindCNPJ)
	if !ok {
		t.Fatal("CNPJ is not registered in the registry")
	}

	if d.Kind() != KindCNPJ {
		t.Fatalf("registered CNPJ Kind = %q, want %q", d.Kind(), KindCNPJ)
	}
}

func TestCNPJ_SatisfiesDocument(t *testing.T) {
	var _ Document = (*CNPJ)(nil)
}

func TestCNPJ_RejectsAllEqual(t *testing.T) {
	c := NewCNPJ()
	for _, bad := range []string{
		"00000000000000",
		"00.000.000/0000-00",
		"11111111111111",
	} {
		if c.Validate(bad) {
			t.Errorf("Validate(%q) = true, want false (all-equal must be rejected)", bad)
		}
	}

	if !c.Validate("39591842000010") {
		t.Error("Validate(\"39591842000010\") = false, want true (known-valid regression sample)")
	}
}

func TestCNPJ_RegressionSample(t *testing.T) {
	// paemuri #26/#27: this valid legacy numeric CNPJ was once falsely rejected.
	c := NewCNPJ()

	const sample = "39591842000010"
	if !c.Validate(sample) {
		t.Fatalf("regression: CNPJ %q must validate true", sample)
	}

	formatted, err := c.Format(sample)
	if err != nil {
		t.Fatalf("Format(%q) error: %v", sample, err)
	}

	const wantMask = "39.591.842/0000-10"
	if formatted != wantMask {
		t.Fatalf("Format(%q) = %q, want %q", sample, formatted, wantMask)
	}

	if !c.Validate(formatted) {
		t.Fatalf("formatted regression CNPJ %q must validate true", formatted)
	}
}
