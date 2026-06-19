package selo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhoneKindAndRegistry(t *testing.T) {
	p := NewPhone()
	assert.Equal(t, KindPhone, p.Kind())

	got, ok := Get(KindPhone)
	require.True(t, ok, "Phone must self-register")
	assert.Equal(t, KindPhone, got.Kind())
}

func TestPhoneValidate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"mobile SP plain", "11987654321", true},
		{"mobile SP with +55", "+5511987654321", true},
		{"mobile SP with 0055", "005511987654321", true},
		{"mobile SP formatted", "(11) 98765-4321", true},
		{"landline SP 8 digit", "1133224455", true},
		{"landline RJ formatted", "(21) 3322-4455", true},
		{"mobile RS", "51999887766", true},
		{"unknown DDD 20", "20987654321", false},
		{"unknown DDD 00", "00987654321", false},
		{"too short", "1198765", false},
		{"too long", "119876543210", false},
		{"non digit", "11ABCDEFGHI", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewPhone().Validate(tt.value))
		})
	}
}

func TestPhoneFormat(t *testing.T) {
	p := NewPhone()

	got, err := p.Format("11987654321")
	require.NoError(t, err)
	assert.Equal(t, "(11) 98765-4321", got)

	got, err = p.Format("+5511987654321")
	require.NoError(t, err)
	assert.Equal(t, "(11) 98765-4321", got)

	got, err = p.Format("1133224455")
	require.NoError(t, err)
	assert.Equal(t, "(11) 3322-4455", got)

	_, err = p.Format("1198765")
	assert.ErrorIs(t, err, ErrInvalidLength)

	_, err = p.Format("20987654321")
	assert.ErrorIs(t, err, ErrInvalidFormat)
}

func TestPhoneOrigin(t *testing.T) {
	p := NewPhone()

	uf, err := p.Origin("11987654321")
	require.NoError(t, err)
	assert.Equal(t, "SP", uf)

	uf, err = p.Origin("+552133224455")
	require.NoError(t, err)
	assert.Equal(t, "RJ", uf)

	uf, err = p.Origin("51999887766")
	require.NoError(t, err)
	assert.Equal(t, "RS", uf)

	_, err = p.Origin("1198765")
	assert.ErrorIs(t, err, ErrInvalidLength)

	_, err = p.Origin("20987654321")
	assert.ErrorIs(t, err, ErrInvalidFormat)
}

func TestPhoneGenerateRoundTrip(t *testing.T) {
	p := NewPhone()
	for range 500 {
		got := p.Generate()
		assert.True(t, len(got) == 10 || len(got) == 11, "Generate must emit 10 or 11 raw digits, got %q", got)
		assert.True(t, p.Validate(got), "generated phone %q must validate", got)
		_, err := p.Origin(got)
		assert.NoError(t, err, "generated phone %q must resolve an origin", got)
	}
}

func TestPhoneViaRegistry(t *testing.T) {
	gen, err := Generate(KindPhone)
	require.NoError(t, err)
	require.True(t, len(gen) == 10 || len(gen) == 11)

	ok, err := Validate(KindPhone, gen)
	require.NoError(t, err)
	assert.True(t, ok)

	_, err = Format(KindPhone, gen)
	require.NoError(t, err)
}

func BenchmarkPhoneValidate(b *testing.B) {
	p := NewPhone()

	b.ReportAllocs()

	for b.Loop() {
		_ = p.Validate("11987654321")
	}
}

func BenchmarkPhoneGenerate(b *testing.B) {
	p := NewPhone()

	b.ReportAllocs()

	for b.Loop() {
		_ = p.Generate()
	}
}
