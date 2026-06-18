package selo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenavam_Validate(t *testing.T) {
	r := NewRenavam()
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid sample one", "12345678900", true},
		{"valid leading zeros", "00000000019", true},
		{"valid high digits", "98765432103", true},
		{"all equal rejected", "11111111111", false},
		{"all zeros rejected", "00000000000", false},
		{"off-by-one dv", "12345678901", false},
		{"too short", "1234567890", false},
		{"too long", "123456789000", false},
		{"non-digit garbage", "abcdefghijk", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, r.Validate(tt.value))
		})
	}
}

func TestRenavam_Generate_RoundTrip(t *testing.T) {
	r := NewRenavam()
	for i := 0; i < 1000; i++ {
		got := r.Generate()
		require.Len(t, got, RenavamLength)
		assert.True(t, r.Validate(got), "generated RENAVAM must validate: %q", got)
		assert.False(t, renavamAllEqual(got), "generated RENAVAM must not be all-equal: %q", got)
	}
}

func TestRenavam_Kind(t *testing.T) {
	assert.Equal(t, KindRenavam, NewRenavam().Kind())
}

func TestRenavam_Format(t *testing.T) {
	r := NewRenavam()
	t.Run("11 digits identity", func(t *testing.T) {
		got, err := r.Format("12345678900")
		require.NoError(t, err)
		assert.Equal(t, "12345678900", got)
	})
	t.Run("short value zero-padded to 11", func(t *testing.T) {
		got, err := r.Format("19")
		require.NoError(t, err)
		assert.Equal(t, "00000000019", got)
	})
}

func BenchmarkRenavam_Validate(b *testing.B) {
	r := NewRenavam()
	const sample = "12345678900"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Validate(sample)
	}
}

func BenchmarkRenavam_Generate(b *testing.B) {
	r := NewRenavam()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Generate()
	}
}
