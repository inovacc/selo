package selo_test

import (
	"math/rand/v2"
	"testing"

	"github.com/inovacc/selo"
)

// seedRand returns two *rand.Rand instances seeded identically.
func seedRand(seed uint64) (*rand.Rand, *rand.Rand) {
	r1 := rand.New(rand.NewPCG(seed, seed+1))
	r2 := rand.New(rand.NewPCG(seed, seed+1))

	return r1, r2
}

// TestRandGeneratorDeterminism verifies that every registered Kind that
// implements RandGenerator produces identical output from identical seeds,
// and that the output still validates.
func TestRandGeneratorDeterminism(t *testing.T) {
	t.Parallel()

	for _, kind := range selo.Kinds() {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()

			r1, r2 := seedRand(42)

			v1, err := selo.GenerateRand(kind, r1)
			if err != nil {
				t.Skipf("kind %q does not implement RandGenerator: %v", kind, err)
			}

			v2, err := selo.GenerateRand(kind, r2)
			if err != nil {
				t.Fatalf("second GenerateRand failed: %v", err)
			}

			if v1 != v2 {
				t.Errorf("GenerateRand not deterministic: got %q and %q", v1, v2)
			}

			// Round-trip: generated value must validate.
			ok, err := selo.Validate(kind, v1)
			if err != nil {
				t.Fatalf("Validate error: %v", err)
			}

			if !ok {
				t.Errorf("GenerateRand(%q) = %q does not validate", kind, v1)
			}
		})
	}
}

// TestGenerateRandUnknownKind verifies ErrUnknownKind is returned for unknown kinds.
func TestGenerateRandUnknownKind(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewPCG(1, 2))

	_, err := selo.GenerateRand("nonexistent_kind", r)
	if err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
}

// TestGeneratePersonWithSeedDeterminism verifies that two GeneratePerson calls
// with the same seed produce deeply equal Person structs.
func TestGeneratePersonWithSeedDeterminism(t *testing.T) {
	t.Parallel()

	p1 := selo.GeneratePerson(selo.WithSeed(42))
	p2 := selo.GeneratePerson(selo.WithSeed(42))

	fields := []struct {
		name string
		a, b string
	}{
		{"Name", p1.Name, p2.Name},
		{"CPF", p1.CPF, p2.CPF},
		{"RG", p1.RG, p2.RG},
		{"CNH", p1.CNH, p2.CNH},
		{"PIS", p1.PIS, p2.PIS},
		{"Renavam", p1.Renavam, p2.Renavam},
		{"VoterID", p1.VoterID, p2.VoterID},
		{"CNS", p1.CNS, p2.CNS},
		{"CEP", p1.CEP, p2.CEP},
		{"Phone", p1.Phone, p2.Phone},
		{"Email", p1.Email, p2.Email},
		{"UF", string(p1.UF), string(p2.UF)},
	}

	for _, f := range fields {
		if f.a != f.b {
			t.Errorf("%s mismatch: %q vs %q", f.name, f.a, f.b)
		}
	}

	if len(p1.PIXKeys) != len(p2.PIXKeys) {
		t.Fatalf("PIXKeys length mismatch: %d vs %d", len(p1.PIXKeys), len(p2.PIXKeys))
	}

	for i := range p1.PIXKeys {
		if p1.PIXKeys[i] != p2.PIXKeys[i] {
			t.Errorf("PIXKeys[%d] mismatch: %q vs %q", i, p1.PIXKeys[i], p2.PIXKeys[i])
		}
	}
}

// TestGeneratePersonDifferentSeedsDiffer verifies different seeds produce different people.
func TestGeneratePersonDifferentSeedsDiffer(t *testing.T) {
	t.Parallel()

	p1 := selo.GeneratePerson(selo.WithSeed(7))
	p2 := selo.GeneratePerson(selo.WithSeed(99999))

	// Very unlikely all fields match; check at least one differs.
	if p1.CPF == p2.CPF && p1.CNH == p2.CNH && p1.Name == p2.Name {
		t.Error("different seeds produced identical persons (extremely unlikely)")
	}
}

// TestGeneratePersonDocumentsValid verifies a seeded person's documents all validate.
func TestGeneratePersonDocumentsValid(t *testing.T) {
	t.Parallel()

	p := selo.GeneratePerson(selo.WithSeed(7))

	checks := []struct {
		name  string
		kind  selo.Kind
		value string
	}{
		{"CPF", selo.KindCPF, p.CPF},
		{"CNH", selo.KindCNH, p.CNH},
		{"PIS", selo.KindPIS, p.PIS},
		{"Renavam", selo.KindRenavam, p.Renavam},
		{"VoterID", selo.KindVoterID, p.VoterID},
		{"CNS", selo.KindCNS, p.CNS},
		{"CEP", selo.KindCEP, p.CEP},
		{"Phone", selo.KindPhone, p.Phone},
	}

	for _, tc := range checks {
		ok, err := selo.Validate(tc.kind, tc.value)
		if err != nil {
			t.Errorf("%s Validate error: %v", tc.name, err)
			continue
		}

		if !ok {
			t.Errorf("%s %q does not validate", tc.name, tc.value)
		}
	}
}

// TestWithRandOption verifies that WithRand works the same as WithSeed for a
// caller-owned source.
func TestWithRandOption(t *testing.T) {
	t.Parallel()

	r1 := rand.New(rand.NewPCG(123, 456))
	r2 := rand.New(rand.NewPCG(123, 456))

	p1 := selo.GeneratePerson(selo.WithRand(r1))
	p2 := selo.GeneratePerson(selo.WithRand(r2))

	if p1.CPF != p2.CPF {
		t.Errorf("WithRand not deterministic: CPF %q vs %q", p1.CPF, p2.CPF)
	}
}
