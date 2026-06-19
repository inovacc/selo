package selo

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPIS_Validate(t *testing.T) {
	p := NewPIS()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid unformatted", "12001234564", true},
		{"valid second sample", "12345678900", true},
		{"valid leading zeros", "00000000019", true},
		{"valid formatted", "120.01234.56-4", true},
		{"all equal rejected", "11111111111", false},
		{"all zeros rejected", "00000000000", false},
		{"off-by-one dv", "12001234565", false},
		{"too short", "1200123456", false},
		{"too long", "120012345644", false},
		{"non-digit garbage", "abcdefghijk", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, p.Validate(tt.value))
		})
	}
}

func TestPIS_Generate_RoundTrip(t *testing.T) {
	p := NewPIS()
	for range 1000 {
		got := p.Generate()
		require.Len(t, got, PisLength)
		assert.True(t, p.Validate(got), "generated PIS must validate: %q", got)
		assert.False(t, pisAllEqual(got), "generated PIS must not be all-equal: %q", got)
	}
}

func TestPIS_Kind(t *testing.T) {
	assert.Equal(t, KindPIS, NewPIS().Kind())
}

func TestPIS_Format(t *testing.T) {
	p := NewPIS()

	t.Run("unformatted to mask", func(t *testing.T) {
		got, err := p.Format("12001234564")
		require.NoError(t, err)
		assert.Equal(t, "120.01234.56-4", got)
	})
	t.Run("already formatted is idempotent", func(t *testing.T) {
		got, err := p.Format("120.01234.56-4")
		require.NoError(t, err)
		assert.Equal(t, "120.01234.56-4", got)
	})
	t.Run("wrong length errors", func(t *testing.T) {
		_, err := p.Format("123")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidLength))
	})
}

func BenchmarkPIS_Validate(b *testing.B) {
	p := NewPIS()

	const sample = "12001234564"

	b.ReportAllocs()

	for b.Loop() {
		_ = p.Validate(sample)
	}
}

func BenchmarkPIS_Generate(b *testing.B) {
	p := NewPIS()

	b.ReportAllocs()

	for b.Loop() {
		_ = p.Generate()
	}
}
