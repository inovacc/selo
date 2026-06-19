package selo

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCNH_Validate(t *testing.T) {
	c := NewCNH()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid offset path", "02345678929", true},   // base 023456789, DV1=2 DV2=9
		{"valid no offset", "12345678900", true},     // base 123456789, DV1=0 DV2=0
		{"valid leading zeros", "00000000119", true}, // base 000000001, DV1=1 DV2=9
		{"valid sample four", "64040501110", true},   // base 640405011, DV1=1 DV2=0
		{"all equal rejected", "11111111111", false},
		{"all zeros rejected", "00000000000", false},
		{"off-by-one dv2", "02345678920", false},
		{"too short", "0234567892", false},
		{"too long", "023456789299", false},
		{"non-digit garbage", "abcdefghijk", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, c.Validate(tt.value))
		})
	}
}

func TestCNH_Generate_RoundTrip(t *testing.T) {
	c := NewCNH()
	for range 1000 {
		got := c.Generate()
		require.Len(t, got, CnhLength)
		assert.True(t, c.Validate(got), "generated CNH must validate: %q", got)
		assert.False(t, cnhAllEqual(got), "generated CNH must not be all-equal: %q", got)
	}
}

func TestCNH_Kind(t *testing.T) {
	assert.Equal(t, KindCNH, NewCNH().Kind())
}

func TestCNH_Format(t *testing.T) {
	c := NewCNH()

	t.Run("11 digits identity", func(t *testing.T) {
		got, err := c.Format("02345678929")
		require.NoError(t, err)
		assert.Equal(t, "02345678929", got)
	})
	t.Run("strips formatting to 11 digits", func(t *testing.T) {
		got, err := c.Format("023-456-789-29")
		require.NoError(t, err)
		assert.Equal(t, "02345678929", got)
	})
	t.Run("wrong length errors", func(t *testing.T) {
		_, err := c.Format("123")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidLength))
	})
}

func BenchmarkCNH_Validate(b *testing.B) {
	c := NewCNH()

	const sample = "02345678929"

	b.ReportAllocs()

	for b.Loop() {
		_ = c.Validate(sample)
	}
}

func BenchmarkCNH_Generate(b *testing.B) {
	c := NewCNH()

	b.ReportAllocs()

	for b.Loop() {
		_ = c.Generate()
	}
}
