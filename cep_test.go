package brdoc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCEPKindAndRegistry(t *testing.T) {
	c := NewCEP()
	assert.Equal(t, KindCEP, c.Kind())

	got, ok := Get(KindCEP)
	require.True(t, ok, "CEP must self-register")
	assert.Equal(t, KindCEP, got.Kind())
}

func TestCEPValidate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid SP formatted", "01310-100", true},
		{"valid SP unformatted", "01310100", true},
		{"valid RS top of range", "90000-000", true},
		{"valid RJ", "20040-002", true},
		{"valid MG", "30140-071", true},
		{"valid AM secondary block", "69400-000", true},
		{"too short", "0131010", false},
		{"too long", "013101000", false},
		{"non digit", "0131A100", false},
		{"prefix below first range", "00900-000", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewCEP().Validate(tt.value))
		})
	}
}

func TestCEPFormat(t *testing.T) {
	c := NewCEP()

	got, err := c.Format("01310100")
	require.NoError(t, err)
	assert.Equal(t, "01310-100", got)

	got, err = c.Format("01310-100")
	require.NoError(t, err)
	assert.Equal(t, "01310-100", got)

	_, err = c.Format("0131010")
	assert.ErrorIs(t, err, ErrInvalidLength)

	_, err = c.Format("0131A100")
	assert.ErrorIs(t, err, ErrInvalidLength)
}

func TestCEPOrigin(t *testing.T) {
	c := NewCEP()

	uf, err := c.Origin("01310-100")
	require.NoError(t, err)
	assert.Equal(t, "SP", uf)

	uf, err = c.Origin("90000-000")
	require.NoError(t, err)
	assert.Equal(t, "RS", uf)

	uf, err = c.Origin("69400-000")
	require.NoError(t, err)
	assert.Equal(t, "AM", uf)

	_, err = c.Origin("0131010")
	assert.ErrorIs(t, err, ErrInvalidLength)

	_, err = c.Origin("00900000")
	assert.ErrorIs(t, err, ErrInvalidFormat)
}

func TestCEPGenerateRoundTrip(t *testing.T) {
	c := NewCEP()
	for i := 0; i < 500; i++ {
		got := c.Generate()
		assert.Len(t, got, CepLength, "Generate must emit 8 raw digits")
		assert.True(t, c.Validate(got), "generated CEP %q must validate", got)
		_, err := c.Origin(got)
		assert.NoError(t, err, "generated CEP %q must resolve an origin", got)
	}
}

func TestCEPViaRegistry(t *testing.T) {
	gen, err := Generate(KindCEP)
	require.NoError(t, err)
	require.Len(t, gen, CepLength)

	ok, err := Validate(KindCEP, gen)
	require.NoError(t, err)
	assert.True(t, ok)

	formatted, err := Format(KindCEP, gen)
	require.NoError(t, err)
	assert.Len(t, formatted, 9) // #####-###
}

// Verify CEP satisfies OriginResolver interface at compile time.
var _ OriginResolver = (*CEP)(nil)

// Verify unused import is exercised (errors package used in test).
var _ = errors.New

func BenchmarkCEPValidate(b *testing.B) {
	c := NewCEP()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Validate("01310-100")
	}
}

func BenchmarkCEPGenerate(b *testing.B) {
	c := NewCEP()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Generate()
	}
}
