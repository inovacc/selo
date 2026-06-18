package mcp

import (
	"context"
	"encoding/json"
	"github.com/inovacc/selo"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// newTestSession spins up the server and a client over in-memory transports
// and returns a connected client session. Cleanup is registered on t.
func newTestSession(t *testing.T) (context.Context, *mcp.ClientSession) {
	t.Helper()
	ctx := context.Background()

	st, ct := mcp.NewInMemoryTransports()

	srv := NewServer("test")
	ss, err := srv.Connect(ctx, st, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ss.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })

	return ctx, cs
}

// decodeResult unmarshals the first TextContent of a non-error result into v.
// The go-sdk serialises StructuredContent into a JSON TextContent, so this
// works regardless of whether the test reads StructuredContent directly.
func decodeResult(t *testing.T, res *mcp.CallToolResult, v any) {
	t.Helper()
	require.False(t, res.IsError, "tool returned an error result")
	require.NotEmpty(t, res.Content, "result has no content")
	tc, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "first content is not TextContent")
	require.NoError(t, json.Unmarshal([]byte(tc.Text), v))
}

func TestValidateDocumentTool(t *testing.T) {
	ctx, cs := newTestSession(t)

	// A freshly generated CPF is always valid; use the registry to source one.
	cpf := selo.NewCPF().Generate()
	require.True(t, selo.NewCPF().Validate(cpf), "generated CPF must validate")

	tests := []struct {
		name      string
		args      map[string]any
		wantValid bool
		wantErr   bool
	}{
		{
			name:      "valid cpf",
			args:      map[string]any{"kind": "cpf", "value": cpf},
			wantValid: true,
		},
		{
			name:      "invalid cpf all equal",
			args:      map[string]any{"kind": "cpf", "value": "11111111111"},
			wantValid: false,
		},
		{
			name:      "cnpj regression sample 39591842000010",
			args:      map[string]any{"kind": "cnpj", "value": "39591842000010"},
			wantValid: true,
		},
		{
			name:    "unknown kind is error result",
			args:    map[string]any{"kind": "bogus", "value": "x"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := cs.CallTool(ctx, &mcp.CallToolParams{
				Name:      "validate_document",
				Arguments: tt.args,
			})
			require.NoError(t, err)

			if tt.wantErr {
				assert.True(t, res.IsError)
				return
			}

			var out ValidateOutput
			decodeResult(t, res, &out)
			assert.Equal(t, tt.wantValid, out.Valid)
		})
	}
}

func TestGenerateDocumentTool(t *testing.T) {
	ctx, cs := newTestSession(t)

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_document",
		Arguments: map[string]any{"kind": "cpf", "count": 3},
	})
	require.NoError(t, err)

	var out GenerateOutput
	decodeResult(t, res, &out)
	require.Len(t, out.Values, 3)
	for _, v := range out.Values {
		assert.True(t, selo.NewCPF().Validate(v), "generated %q must validate", v)
	}

	// count omitted -> defaults to 1.
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_document",
		Arguments: map[string]any{"kind": "cnpj"},
	})
	require.NoError(t, err)
	var one GenerateOutput
	decodeResult(t, res, &one)
	assert.Len(t, one.Values, 1)

	// unknown kind -> error result.
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_document",
		Arguments: map[string]any{"kind": "bogus"},
	})
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestFormatDocumentTool(t *testing.T) {
	ctx, cs := newTestSession(t)

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "format_document",
		Arguments: map[string]any{"kind": "cpf", "value": "11144477735"},
	})
	require.NoError(t, err)

	var out FormatOutput
	decodeResult(t, res, &out)
	assert.Equal(t, "111.444.777-35", out.Formatted)

	// bad length -> error result (ErrInvalidLength surfaced as message).
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "format_document",
		Arguments: map[string]any{"kind": "cpf", "value": "123"},
	})
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestDetectDocumentTool(t *testing.T) {
	ctx, cs := newTestSession(t)

	tests := []struct {
		name      string
		value     string
		wantKind  string
		wantValid bool
	}{
		{name: "cpf length", value: "11144477735", wantKind: "cpf", wantValid: true},
		{name: "cnpj length", value: "39591842000010", wantKind: "cnpj", wantValid: true},
		{name: "unknown length", value: "12345", wantKind: "", wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := cs.CallTool(ctx, &mcp.CallToolParams{
				Name:      "detect_document",
				Arguments: map[string]any{"value": tt.value},
			})
			require.NoError(t, err)

			var out DetectOutput
			decodeResult(t, res, &out)
			assert.Equal(t, tt.wantKind, out.Kind)
			assert.Equal(t, tt.wantValid, out.Valid)
		})
	}
}

func TestListDocumentTypesTool(t *testing.T) {
	ctx, cs := newTestSession(t)

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_document_types",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)

	var out ListOutput
	decodeResult(t, res, &out)

	want := make([]string, 0, len(selo.Kinds()))
	for _, k := range selo.Kinds() {
		want = append(want, k.String())
	}
	assert.Equal(t, want, out.Kinds)
	assert.Contains(t, out.Kinds, "cpf")
	assert.Contains(t, out.Kinds, "cnpj")
}

func TestKindEnumMatchesRegistry(t *testing.T) {
	enum := kindEnum()
	require.Len(t, enum, len(selo.Kinds()))
	assert.Equal(t, selo.Kinds()[0].String(), enum[0])
}
