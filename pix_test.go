package selo

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectPIXKind(t *testing.T) {
	t.Parallel()

	// Real valid samples (generated/known-good for each kind).
	const (
		validCPF   = "52998224725"    // passes CPF check digits
		validCNPJ  = "39591842000010" // paemuri regression sample
		validEmail = "joao.silva@example.com.br"
		validPhone = "+5511998765432"                       // +55, DDD 11, 9-digit mobile
		validEVP   = "123e4567-e89b-42d3-a456-426614174000" // UUIDv4 shape
	)

	tests := []struct {
		name     string
		value    string
		wantKind string
		wantOK   bool
	}{
		{"cpf key", validCPF, PIXKindCPF, true},
		{"cpf key formatted", "529.982.247-25", PIXKindCPF, true},
		{"cnpj key", validCNPJ, PIXKindCNPJ, true},
		{"cnpj key formatted", "39.591.842/0000-10", PIXKindCNPJ, true},
		{"email key", validEmail, PIXKindEmail, true},
		{"email key simple", "a@b.co", PIXKindEmail, true},
		{"phone key 9 digit", validPhone, PIXKindPhone, true},
		{"phone key 8 digit", "+551133224455", PIXKindPhone, true},
		{"evp key", validEVP, PIXKindEVP, true},
		{"evp key uppercase", "123E4567-E89B-42D3-A456-426614174000", PIXKindEVP, true},
		{"empty", "", "", false},
		{"all-equal cpf rejected", "11111111111", "", false},
		{"off-by-one cpf dv", "52998224724", "", false},
		{"email no domain dot", "joao@example", "", false},
		{"email double at", "a@@b.co", "", false},
		{"phone no plus55", "11998765432", "", false},
		{"phone wrong country", "+1198765432", "", false},
		{"phone too short", "+55119987", "", false},
		{"evp wrong version", "123e4567-e89b-12d3-a456-426614174000", "", false},
		{"evp wrong variant", "123e4567-e89b-42d3-c456-426614174000", "", false},
		{"random junk", "not-a-key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotKind, gotOK := DetectPIXKind(tt.value)
			assert.Equal(t, tt.wantOK, gotOK)
			assert.Equal(t, tt.wantKind, gotKind)
		})
	}
}

// compile-time anchor so the regexp import is used before impl lands.
var _ = regexp.MustCompile

func TestPIXKind(t *testing.T) {
	t.Parallel()
	assert.Equal(t, KindPIX, NewPIX().Kind())
	assert.Equal(t, "pix", NewPIX().Kind().String())
}

func TestPIXValidate(t *testing.T) {
	t.Parallel()

	p := NewPIX()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid cpf", "52998224725", true},
		{"valid cnpj regression", "39591842000010", true},
		{"valid email", "joao.silva@example.com.br", true},
		{"valid phone e164", "+5511998765432", true},
		{"valid evp uuidv4", "123e4567-e89b-42d3-a456-426614174000", true},
		{"invalid cpf dv", "52998224724", false},
		{"invalid email", "joao@example", false},
		{"invalid phone", "11998765432", false},
		{"invalid evp version", "123e4567-e89b-12d3-a456-426614174000", false},
		{"empty", "", false},
		{"junk", "totally-not-a-pix-key", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Validate(tt.value))
		})
	}
}

func TestPIXFormat(t *testing.T) {
	t.Parallel()

	p := NewPIX()

	t.Run("identity on valid key", func(t *testing.T) {
		t.Parallel()

		out, err := p.Format("  joao.silva@example.com.br  ")
		assert.NoError(t, err)
		assert.Equal(t, "joao.silva@example.com.br", out) // trimmed, otherwise verbatim
	})

	t.Run("identity on valid cpf key keeps mask", func(t *testing.T) {
		t.Parallel()

		out, err := p.Format("529.982.247-25")
		assert.NoError(t, err)
		assert.Equal(t, "529.982.247-25", out) // PIX has no canonical mask; verbatim
	})

	t.Run("error on invalid key", func(t *testing.T) {
		t.Parallel()

		_, err := p.Format("not-a-pix-key")
		assert.ErrorIs(t, err, ErrInvalidLength)
	})
}
func TestPIXGenerate(t *testing.T) {
	t.Parallel()

	p := NewPIX()

	// Round-trip: every generated key is a valid PIX key, and a valid EVP UUIDv4.
	for range 200 {
		key := p.Generate()

		assert.True(t, p.Validate(key), "generated key must validate: %q", key)
		assert.Truef(t, pixEVPRe.MatchString(key), "generated key must be UUIDv4: %q", key)

		kind, ok := DetectPIXKind(key)
		assert.True(t, ok)
		assert.Equal(t, PIXKindEVP, kind)

		// Canonical UUIDv4: 36 chars, version '4', lowercase hex.
		assert.Len(t, key, 36)
		assert.Equal(t, byte('4'), key[14], "version nibble must be 4: %q", key)
		assert.Equal(t, strings.ToLower(key), key, "generated key must be lowercase: %q", key)
	}
}

func TestPIXGenerateUnique(t *testing.T) {
	t.Parallel()

	p := NewPIX()
	seen := make(map[string]struct{}, 1000)

	for range 1000 {
		key := p.Generate()
		_, dup := seen[key]
		assert.Falsef(t, dup, "generated duplicate EVP key: %q", key)
		seen[key] = struct{}{}
	}
}

func TestPIXRegistered(t *testing.T) {
	t.Parallel()

	doc, ok := Get(KindPIX)
	assert.True(t, ok, "PIX must self-register")
	assert.Equal(t, KindPIX, doc.Kind())

	// Round-trip through the registry dispatcher.
	valid, err := Validate(KindPIX, "52998224725")
	assert.NoError(t, err)
	assert.True(t, valid)

	valid, err = Validate(KindPIX, "not-a-key")
	assert.NoError(t, err)
	assert.False(t, valid)
}
func FuzzPIXValidate(f *testing.F) {
	seeds := []string{
		"52998224725",
		"39591842000010",
		"joao.silva@example.com.br",
		"+5511998765432",
		"123e4567-e89b-42d3-a456-426614174000",
		"",
		"not-a-key",
		"@@@",
		"+55",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	p := NewPIX()

	f.Fuzz(func(t *testing.T, value string) {
		// Must never panic on arbitrary input.
		_ = p.Validate(value)
		_, _ = DetectPIXKind(value)

		// A Generate-produced key must always validate (round-trip invariant).
		key := p.Generate()
		if !p.Validate(key) {
			t.Fatalf("generated key failed to validate: %q", key)
		}
	})
}
