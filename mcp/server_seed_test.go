package mcp

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateToolDeterministic verifies the generate_document tool's seed
// parameter produces reproducible-yet-distinct output.
func TestGenerateToolDeterministic(t *testing.T) {
	ctx, cs := newTestSession(t)

	gen := func(seed int) []string {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      "generate_document",
			Arguments: map[string]any{"kind": "cpf", "count": 4, "seed": seed},
		})
		require.NoError(t, err)

		var out GenerateOutput
		decodeResult(t, res, &out)

		return out.Values
	}

	a := gen(42)
	b := gen(42)

	require.Len(t, a, 4)
	assert.Equal(t, a, b, "same seed must produce identical values")

	c := gen(7)
	assert.NotEqual(t, a, c, "a different seed should produce different values")
	assert.NotEqual(t, a[0], a[1], "a seeded batch must yield distinct values")
}
