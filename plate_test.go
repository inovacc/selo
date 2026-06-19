package selo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlateKindAndRegistry(t *testing.T) {
	p := NewPlate()
	assert.Equal(t, KindPlate, p.Kind())

	got, ok := Get(KindPlate)
	require.True(t, ok, "Plate must self-register")
	assert.Equal(t, KindPlate, got.Kind())
}

func TestPlateHelpers(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		national, merc bool
	}{
		{"national with dash", "ABC-1234", true, false},
		{"national no dash", "ABC1234", true, false},
		{"national lowercase", "abc1234", true, false},
		{"mercosul", "ABC1D23", false, true},
		{"mercosul lowercase", "abc1d23", false, true},
		{"garbage", "AB-1234", false, false},
		{"too long", "ABCD1234", false, false},
		{"empty", "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.national, IsNationalPlate(tt.value), "national")
			assert.Equal(t, tt.merc, IsMercosulPlate(tt.value), "mercosul")
			assert.Equal(t, tt.national || tt.merc, IsPlate(tt.value), "any")
			// Method forms (consumed by compat/ MC-2) must mirror the helpers.
			p := NewPlate()
			assert.Equal(t, tt.national, p.ValidateNational(tt.value), "ValidateNational")
			assert.Equal(t, tt.merc, p.ValidateMercosul(tt.value), "ValidateMercosul")
		})
	}
}

func TestPlateFormat(t *testing.T) {
	p := NewPlate()

	got, err := p.Format("ABC1234")
	require.NoError(t, err)
	assert.Equal(t, "ABC-1234", got)

	got, err = p.Format("ABC-1234")
	require.NoError(t, err)
	assert.Equal(t, "ABC-1234", got)

	got, err = p.Format("abc1234")
	require.NoError(t, err)
	assert.Equal(t, "ABC-1234", got)

	got, err = p.Format("ABC1D23")
	require.NoError(t, err)
	assert.Equal(t, "ABC1D23", got)

	_, err = p.Format("AB-1234")
	assert.ErrorIs(t, err, ErrInvalidFormat)
}

func TestPlateGenerateRoundTrip(t *testing.T) {
	nat := &Plate{Mercosul: false}
	for range 300 {
		got := nat.Generate()
		assert.Len(t, got, 7, "national plate is 7 chars unformatted")
		assert.True(t, IsNationalPlate(got), "generated national plate %q must match", got)
		assert.True(t, nat.Validate(got))
	}

	merc := &Plate{Mercosul: true}
	for range 300 {
		got := merc.Generate()
		assert.Len(t, got, 7, "mercosul plate is 7 chars")
		assert.True(t, IsMercosulPlate(got), "generated mercosul plate %q must match", got)
		assert.True(t, merc.Validate(got))
	}
}

func TestPlateViaRegistry(t *testing.T) {
	gen, err := Generate(KindPlate)
	require.NoError(t, err)
	require.Len(t, gen, 7)

	ok, err := Validate(KindPlate, gen)
	require.NoError(t, err)
	assert.True(t, ok)

	formatted, err := Format(KindPlate, gen)
	require.NoError(t, err)
	assert.NotEmpty(t, formatted)
}

func BenchmarkPlateValidate(b *testing.B) {
	p := NewPlate()

	b.ReportAllocs()

	for b.Loop() {
		_ = p.Validate("ABC1D23")
	}
}

func BenchmarkPlateGenerate(b *testing.B) {
	p := &Plate{Mercosul: true}

	b.ReportAllocs()

	for b.Loop() {
		_ = p.Generate()
	}
}
