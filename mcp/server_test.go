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

	// A freshly generated RG is always valid for SP.
	rg := selo.NewRG().Generate()

	tests := []struct {
		name       string
		args       map[string]any
		wantValid  bool
		wantErr    bool
		wantOrigin string // non-empty means we assert the origin field
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
		// uf supplied but kind is NOT UFScoped → error result.
		{
			name:    "uf on non-uf-scoped kind is error",
			args:    map[string]any{"kind": "cpf", "value": cpf, "uf": "SP"},
			wantErr: true,
		},
		// uf supplied, kind IS UFScoped, ValidateUF succeeds.
		{
			name:      "rg valid with uf SP",
			args:      map[string]any{"kind": "rg", "value": rg, "uf": "SP"},
			wantValid: true,
		},
		// uf supplied, kind IS UFScoped, ValidateUF returns ErrUFNotImplemented.
		{
			name:    "rg with unimplemented uf MG is error",
			args:    map[string]any{"kind": "rg", "value": rg, "uf": "MG"},
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

			if tt.wantOrigin != "" {
				assert.Equal(t, tt.wantOrigin, out.Origin)
			}
		})
	}
}

// TestValidateDocumentOriginResolver exercises the OriginResolver path: a valid
// CPF must come back with a non-empty Origin field.
func TestValidateDocumentOriginResolver(t *testing.T) {
	ctx, cs := newTestSession(t)

	cpf := selo.NewCPF().Generate()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "validate_document",
		Arguments: map[string]any{"kind": "cpf", "value": cpf},
	})
	require.NoError(t, err)

	var out ValidateOutput
	decodeResult(t, res, &out)
	assert.True(t, out.Valid)
	assert.NotEmpty(t, out.Origin, "valid CPF should have an Origin")
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

	// unknown kind -> error result.
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "format_document",
		Arguments: map[string]any{"kind": "bogus", "value": "11144477735"},
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

func TestGeneratePersonTool(t *testing.T) {
	ctx, cs := newTestSession(t)

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_person",
		Arguments: map[string]any{"uf": "SP", "count": 2, "with_vehicle": true, "with_company": true},
	})
	require.NoError(t, err)

	var out PersonOutput
	decodeResult(t, res, &out)
	require.Len(t, out.People, 2)

	for _, p := range out.People {
		assert.Equal(t, selo.UFSP, p.UF)
		assert.Truef(t, selo.NewCPF().Validate(p.CPF), "CPF %q", p.CPF)
		cepUF, err := selo.NewCEP().Origin(p.CEP)
		assert.NoError(t, err)
		assert.Equal(t, "SP", cepUF)
		require.NotNil(t, p.Vehicle)
		require.NotNil(t, p.Company)
		assert.True(t, selo.NewCNPJ().Validate(p.Company.CNPJ))

		// Address serializes through StructuredContent and is UF-consistent.
		require.NotNil(t, p.Address)
		assert.Equal(t, selo.UFSP, p.Address.UF)
		assert.Equal(t, p.CEP, p.Address.CEP)
		assert.NotEmpty(t, p.Address.City)
	}

	// Invalid UF yields an error result (not a transport error).
	bad, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_person",
		Arguments: map[string]any{"uf": "ZZ"},
	})
	require.NoError(t, err)
	assert.True(t, bad.IsError, "invalid uf should yield an error result")

	// formatted=true exercises the Formatted() option branch.
	fmtRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_person",
		Arguments: map[string]any{"count": 1, "formatted": true},
	})
	require.NoError(t, err)

	var fmtOut PersonOutput
	decodeResult(t, fmtRes, &fmtOut)
	require.Len(t, fmtOut.People, 1)
}

func TestGeneratePersonToolDeterministic(t *testing.T) {
	ctx, cs := newTestSession(t)

	call := func(seed int) PersonOutput {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      "generate_person",
			Arguments: map[string]any{"uf": "SP", "count": 3, "seed": seed},
		})
		require.NoError(t, err)

		var out PersonOutput
		decodeResult(t, res, &out)

		return out
	}

	a := call(42)
	b := call(42)

	require.Len(t, a.People, 3)
	assert.Equal(t, a.People, b.People, "same seed must produce identical people")

	c := call(99)
	assert.NotEqual(t, a.People, c.People, "a different seed should produce different people")

	// Distinct within the seeded batch (shared advancing stream).
	assert.NotEqual(t, a.People[0].CPF, a.People[1].CPF, "seeded batch must yield distinct people")
	assert.NotEqual(t, a.People[1].CPF, a.People[2].CPF, "seeded batch must yield distinct people")
}

func TestGenerateCodeTool(t *testing.T) {
	ctx, cs := newTestSession(t)

	// M2: the TypeScript emitter is registered, so ts/cpf returns real files.
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_code",
		Arguments: map[string]any{"lang": "ts", "kind": "cpf"},
	})
	require.NoError(t, err)
	require.Falsef(t, res.IsError, "M2 generate_code(ts, cpf) should succeed: %+v", res.Content)

	var codeOut GenerateCodeOutput
	decodeResult(t, res, &codeOut)
	require.NotEmpty(t, codeOut.Files, "ts/cpf should produce files")

	var hasModule bool

	for _, f := range codeOut.Files {
		if f.Path == "src/cpf.ts" {
			hasModule = true

			assert.Contains(t, f.Content, "export function validateCPF")
		}
	}

	assert.True(t, hasModule, "ts/cpf should include src/cpf.ts")

	// All five language emitters are registered (M2–M6), so ruby/cpf also
	// returns real files via MCP.
	rubyRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_code",
		Arguments: map[string]any{"lang": "ruby", "kind": "cpf"},
	})
	require.NoError(t, err)
	require.Falsef(t, rubyRes.IsError, "generate_code(ruby, cpf) should succeed: %+v", rubyRes.Content)

	var rubyOut GenerateCodeOutput
	decodeResult(t, rubyRes, &rubyOut)
	require.NotEmpty(t, rubyOut.Files, "ruby/cpf should produce files")

	// Unsupported language is a clean error result.
	bad, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_code",
		Arguments: map[string]any{"lang": "bogus", "kind": "cpf"},
	})
	require.NoError(t, err)
	assert.True(t, bad.IsError, "unsupported lang should yield an error result")

	// Unknown kind is a clean error result.
	badKind, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "generate_code",
		Arguments: map[string]any{"lang": "ts", "kind": "nope"},
	})
	require.NoError(t, err)
	assert.True(t, badKind.IsError, "unknown kind should yield an error result")
}
