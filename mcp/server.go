package mcp

import (
	"context"
	"fmt"
	brdoc "github.com/inovacc/brdoc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"log/slog"
	"os"
)

// Package mcp adapts the brdoc registry to a Model Context Protocol server.
//
// It exposes five tools (validate_document, generate_document,
// format_document, detect_document, list_document_types) over stdio.
// Every tool is derived from the brdoc registry, so adding a new document
// type to the registry automatically widens the MCP surface with no edits
// here.

// ValidateInput is the typed input for the validate_document tool.
type ValidateInput struct {
	Kind  string `json:"kind" jsonschema:"document kind, e.g. cpf or cnpj"`
	Value string `json:"value" jsonschema:"the document value to validate"`
	UF    string `json:"uf,omitempty" jsonschema:"federative unit, only used for kind rg, e.g. SP"`
}

// ValidateOutput is the typed output for the validate_document tool.
type ValidateOutput struct {
	Valid  bool   `json:"valid" jsonschema:"true when the value is a valid document of the given kind"`
	Origin string `json:"origin,omitempty" jsonschema:"geographic origin when the kind supports it (cpf region, cep/phone/voter_id UF)"`
}

// GenerateInput is the typed input for the generate_document tool.
type GenerateInput struct {
	Kind  string `json:"kind" jsonschema:"document kind to generate"`
	Count int    `json:"count,omitempty" jsonschema:"how many values to generate, defaults to 1"`
}

// GenerateOutput is the typed output for the generate_document tool.
type GenerateOutput struct {
	Values []string `json:"values" jsonschema:"the generated, valid document values"`
}

// FormatInput is the typed input for the format_document tool.
type FormatInput struct {
	Kind  string `json:"kind" jsonschema:"document kind"`
	Value string `json:"value" jsonschema:"the document value to format with its canonical mask"`
}

// FormatOutput is the typed output for the format_document tool.
type FormatOutput struct {
	Formatted string `json:"formatted" jsonschema:"the value rendered with its canonical mask"`
}

// DetectInput is the typed input for the detect_document tool.
type DetectInput struct {
	Value string `json:"value" jsonschema:"a document value of unknown kind"`
}

// DetectOutput is the typed output for the detect_document tool.
type DetectOutput struct {
	Kind  string `json:"kind" jsonschema:"the detected document kind, empty when unknown"`
	Valid bool   `json:"valid" jsonschema:"true when a kind was detected and the value validates"`
}

// ListInput is the (empty) typed input for the list_document_types tool.
type ListInput struct{}

// ListOutput is the typed output for the list_document_types tool.
type ListOutput struct {
	Kinds []string `json:"kinds" jsonschema:"all document kinds the server supports"`
}

// kindEnum returns one enum value per registered kind, for the jsonschema
// "kind" field. Sourced from the registry so it stays in sync automatically.
func kindEnum() []any {
	kinds := brdoc.Kinds()
	out := make([]any, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, k.String())
	}
	return out
}

// errResult builds a tool result flagged as an error with a human-readable
// message. The typed Out zero value is returned alongside.
func errResult[Out any](msg string) (*mcp.CallToolResult, Out, error) {
	var zero Out
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, zero, nil
}

func validateHandler(_ context.Context, _ *mcp.CallToolRequest, in ValidateInput) (*mcp.CallToolResult, ValidateOutput, error) {
	kind := brdoc.Kind(in.Kind)
	doc, ok := brdoc.Get(kind)
	if !ok {
		return errResult[ValidateOutput](fmt.Sprintf("unknown document kind %q", in.Kind))
	}

	var out ValidateOutput
	if in.UF != "" {
		scoped, isScoped := doc.(brdoc.UFScoped)
		if !isScoped {
			return errResult[ValidateOutput](fmt.Sprintf("kind %q does not accept a uf", in.Kind))
		}
		valid, err := scoped.ValidateUF(in.Value, brdoc.UF(in.UF))
		if err != nil {
			return errResult[ValidateOutput](err.Error())
		}
		out.Valid = valid
	} else {
		valid, err := brdoc.Validate(kind, in.Value)
		if err != nil {
			return errResult[ValidateOutput](err.Error())
		}
		out.Valid = valid
	}

	if res, hasOrigin := doc.(brdoc.OriginResolver); hasOrigin && out.Valid {
		if origin, err := res.Origin(in.Value); err == nil {
			out.Origin = origin
		}
	}

	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

func generateHandler(_ context.Context, _ *mcp.CallToolRequest, in GenerateInput) (*mcp.CallToolResult, GenerateOutput, error) {
	count := in.Count
	if count <= 0 {
		count = 1
	}

	values := make([]string, 0, count)
	for i := 0; i < count; i++ {
		v, err := brdoc.Generate(brdoc.Kind(in.Kind))
		if err != nil {
			return errResult[GenerateOutput](err.Error())
		}
		values = append(values, v)
	}

	out := GenerateOutput{Values: values}
	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

func formatHandler(_ context.Context, _ *mcp.CallToolRequest, in FormatInput) (*mcp.CallToolResult, FormatOutput, error) {
	formatted, err := brdoc.Format(brdoc.Kind(in.Kind), in.Value)
	if err != nil {
		return errResult[FormatOutput](err.Error())
	}
	out := FormatOutput{Formatted: formatted}
	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

func detectHandler(_ context.Context, _ *mcp.CallToolRequest, in DetectInput) (*mcp.CallToolResult, DetectOutput, error) {
	kind, ok := brdoc.Detect(in.Value)
	out := DetectOutput{Kind: kind.String(), Valid: ok}
	if !ok {
		out.Kind = ""
	}
	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

func listHandler(_ context.Context, _ *mcp.CallToolRequest, _ ListInput) (*mcp.CallToolResult, ListOutput, error) {
	kinds := brdoc.Kinds()
	names := make([]string, 0, len(kinds))
	for _, k := range kinds {
		names = append(names, k.String())
	}
	out := ListOutput{Kinds: names}
	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

// NewServer builds an MCP server with all five brdoc tools registered.
// version is stamped into the server Implementation (use build info).
func NewServer(version string) *mcp.Server {
	if version == "" {
		version = "dev"
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := mcp.NewServer(
		&mcp.Implementation{Name: brdoc.MCPServerName, Version: version},
		&mcp.ServerOptions{Logger: logger},
	)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "validate_document",
		Description: "Validate a Brazilian document of a given kind; returns valid and optional origin.",
	}, validateHandler)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "generate_document",
		Description: "Generate one or more valid Brazilian documents of a given kind.",
	}, generateHandler)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "format_document",
		Description: "Format a Brazilian document with its canonical mask.",
	}, formatHandler)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "detect_document",
		Description: "Detect the kind of an unknown Brazilian document value.",
	}, detectHandler)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_document_types",
		Description: "List every Brazilian document kind this server supports.",
	}, listHandler)

	return srv
}

// Serve runs the MCP server over stdio until the context is cancelled or
// stdin closes. The logger writes to stderr because stdout carries the
// JSON-RPC stream.
func Serve(ctx context.Context, version string) error {
	srv := NewServer(version)
	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("brdoc mcp: %w", err)
	}
	return nil
}
