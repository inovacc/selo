package selo

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDoc is a controllable Document for registry tests.
type fakeDoc struct {
	kind  Kind
	valid bool
	gen   string
	fmtd  string
	fmErr error
}

func (f fakeDoc) Kind() Kind           { return f.kind }
func (f fakeDoc) Validate(string) bool { return f.valid }
func (f fakeDoc) Generate() string     { return f.gen }
func (f fakeDoc) Format(string) (string, error) {
	return f.fmtd, f.fmErr
}

func TestRegistry_RegisterGetDispatch(t *testing.T) {
	// Use a kind value that no real type registers, to avoid collisions.
	const testKind Kind = "test_fake"
	Register(fakeDoc{kind: testKind, valid: true, gen: "GEN", fmtd: "FMT"})

	got, ok := Get(testKind)
	if !ok {
		t.Fatal("Get returned ok=false for a registered kind")
	}

	if got.Kind() != testKind {
		t.Fatalf("Get returned wrong kind: %q", got.Kind())
	}

	ok, err := Validate(testKind, "x")
	if err != nil || !ok {
		t.Fatalf("Validate = (%v, %v), want (true, nil)", ok, err)
	}

	g, err := Generate(testKind)
	if err != nil || g != "GEN" {
		t.Fatalf("Generate = (%q, %v), want (GEN, nil)", g, err)
	}

	fm, err := Format(testKind, "x")
	if err != nil || fm != "FMT" {
		t.Fatalf("Format = (%q, %v), want (FMT, nil)", fm, err)
	}
}

func TestRegistry_UnknownKind(t *testing.T) {
	const missing Kind = "does_not_exist"
	if _, ok := Get(missing); ok {
		t.Fatal("Get returned ok=true for an unregistered kind")
	}

	if _, err := Validate(missing, "x"); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("Validate err = %v, want ErrUnknownKind", err)
	}

	if _, err := Generate(missing); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("Generate err = %v, want ErrUnknownKind", err)
	}

	if _, err := Format(missing, "x"); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("Format err = %v, want ErrUnknownKind", err)
	}
}

func TestRegistry_KindsSorted(t *testing.T) {
	ks := Kinds()
	for i := 1; i < len(ks); i++ {
		if ks[i-1] >= ks[i] {
			t.Fatalf("Kinds() not strictly sorted at %d: %q >= %q", i, ks[i-1], ks[i])
		}
	}
}

func TestRegistry_M2A_Registered(t *testing.T) {
	for _, k := range []Kind{KindPIS, KindRenavam, KindCNH} {
		t.Run(string(k), func(t *testing.T) {
			gen, err := Generate(k)
			require.NoError(t, err)
			ok, err := Validate(k, gen)
			require.NoError(t, err)
			assert.True(t, ok, "registry-generated %s must validate: %q", k, gen)
			formatted, err := Format(k, gen)
			require.NoError(t, err)
			assert.NotEmpty(t, formatted)
		})
	}
}

func TestRegistry_M2A_KindsListed(t *testing.T) {
	got := Kinds()
	for _, want := range []Kind{KindPIS, KindRenavam, KindCNH} {
		assert.Contains(t, got, want)
	}
}
