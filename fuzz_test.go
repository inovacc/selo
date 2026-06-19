package selo

import "testing"

// fuzzNoPanic seeds a fuzz target with representative inputs and asserts that
// Validate and Format never panic on arbitrary input. The per-type tests cover
// the Generate->Validate round-trip invariant; this guards robustness.
func fuzzNoPanic(f *testing.F, d Document, seeds ...string) {
	f.Helper()

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(_ *testing.T, s string) {
		_ = d.Validate(s)
		_, _ = d.Format(s)
	})
}

func FuzzCPFValidate(f *testing.F) {
	fuzzNoPanic(f, NewCPF(), "52998224725", "529.982.247-25", "00000000000", "abc", "")
}

func FuzzCNPJValidate(f *testing.F) {
	fuzzNoPanic(f, NewCNPJ(), "39591842000010", "39.591.842/0000-10", "00000000000000", "12.ABC.345/01DE-35", "")
}

func FuzzCNHValidate(f *testing.F) {
	fuzzNoPanic(f, NewCNH(), "12345678900", "11111111111", "abc", "")
}

func FuzzPISValidate(f *testing.F) {
	fuzzNoPanic(f, NewPIS(), "12001234564", "120.01234.56-4", "0", "")
}

func FuzzRENAVAMValidate(f *testing.F) {
	fuzzNoPanic(f, NewRenavam(), "12345678900", "00000000001", "abc", "")
}
