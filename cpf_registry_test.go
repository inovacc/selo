package selo

import "testing"

func TestCPF_Registered(t *testing.T) {
	d, ok := Get(KindCPF)
	if !ok {
		t.Fatal("CPF is not registered in the registry")
	}
	if d.Kind() != KindCPF {
		t.Fatalf("registered CPF Kind = %q, want %q", d.Kind(), KindCPF)
	}
}

func TestCPF_SatisfiesInterfaces(t *testing.T) {
	var _ Document = (*CPF)(nil)
	var _ OriginResolver = (*CPF)(nil)
}

func TestCPF_Origin(t *testing.T) {
	c := NewCPF()
	got, err := c.Origin("123.456.789-09")
	if err != nil {
		t.Fatalf("Origin returned error: %v", err)
	}
	if got != IsDigit9 {
		t.Fatalf("Origin = %q, want %q", got, IsDigit9)
	}
}
