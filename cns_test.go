package brdoc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCNSValidate(t *testing.T) {
	c := NewCNS()

	// Build valid samples constructively so the test is self-consistent
	// regardless of which literal samples remain valid over time.
	def := c.Generate() // definitive/provisional valid CNS
	require.Len(t, def, CNSLength)

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "generated valid", input: def, want: true},
		{name: "wrong length short", input: "29807085064000", want: false},
		{name: "wrong length long", input: "2980708506400070", want: false},
		{name: "all equal", input: "111111111111111", want: false},
		{name: "bad prefix class 3", input: "300000000000000", want: false},
		{name: "bad prefix class 0", input: "000000000000000", want: false},
		{name: "non multiple of 11", input: "100000000000001", want: false},
		{name: "empty", input: "", want: false},
		{name: "non-digit", input: "29807085064000A", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, c.Validate(tt.input))
		})
	}
}

func TestCNSGenerate(t *testing.T) {
	c := NewCNS()

	for i := 0; i < 2000; i++ {
		got := c.Generate()
		require.Len(t, got, CNSLength, "generated CNS must be 15 digits")
		assert.True(t, cnsValidPrefix(got[0]), "generated CNS %q must have a valid prefix", got)
		assert.Equal(t, 0, cnsWeightedSum(got)%11, "generated CNS %q sum must be divisible by 11", got)
		assert.True(t, c.Validate(got), "generated CNS %q must validate", got)
	}
}

func TestCNSFormat(t *testing.T) {
	c := NewCNS()
	sample := c.Generate()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "identity from generated", input: sample, want: sample},
		{name: "strips separators", input: sample[0:3] + " " + sample[3:], want: sample},
		{name: "too short", input: "29807085064000", wantErr: ErrInvalidLength},
		{name: "too long", input: "2980708506400070", wantErr: ErrInvalidLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Format(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCNSImplementsDocument(t *testing.T) {
	var _ Document = NewCNS()
}

func TestVoterIDImplementsDocument(t *testing.T) {
	var _ Document = NewVoterID()
}

func BenchmarkCNSValidate(b *testing.B) {
	c := NewCNS()
	sample := c.Generate()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Validate(sample)
	}
}

func BenchmarkCNSGenerate(b *testing.B) {
	c := NewCNS()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Generate()
	}
}
