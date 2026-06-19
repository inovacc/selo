package selo

import "testing"

func TestUF_Valid(t *testing.T) {
	tests := []struct {
		uf   UF
		want bool
	}{
		{UFSP, true},
		{UFRJ, true},
		{UFDF, true},
		{UFRS, true},
		{UF("XX"), false},
		{UF(""), false},
		{UF("sp"), false}, // case-sensitive: constants are upper-case
	}
	for _, tt := range tests {
		t.Run(string(tt.uf), func(t *testing.T) {
			if got := tt.uf.Valid(); got != tt.want {
				t.Fatalf("UF(%q).Valid() = %v, want %v", tt.uf, got, tt.want)
			}
		})
	}
}

func TestAllUFs_Count(t *testing.T) {
	got := AllUFs()
	if len(got) != 27 {
		t.Fatalf("AllUFs() returned %d UFs, want 27", len(got))
	}
	// stable order: sorted ascending
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Fatalf("AllUFs() not strictly sorted at %d: %q >= %q", i, got[i-1], got[i])
		}
	}
	// returned slice must be a copy (mutating it must not affect later calls)
	got[0] = UF("ZZ")

	again := AllUFs()
	if again[0] == UF("ZZ") {
		t.Fatal("AllUFs() leaks internal backing array")
	}
}

func TestStubTables(t *testing.T) {
	// cepRanges is populated by cep.go init(); expect one entry per UF
	// (26 primary ranges — AM has two blocks but only the first is stored).
	if len(cepRanges) == 0 {
		t.Fatal("cepRanges must not be empty after CEP init()")
	}
	// dddToUF is populated by phone.go init() — one entry per valid DDD.
	if len(dddToUF) == 0 {
		t.Fatal("dddToUF must not be empty after Phone init()")
	}
}
