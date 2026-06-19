package mcp

import (
	"context"
	"fmt"
	"github.com/inovacc/selo"
	"github.com/inovacc/selo/internal/codegen"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"log/slog"
	"os"
	"strings"
)

// Package mcp adapts the selo registry to a Model Context Protocol server.
//
// It exposes seven tools (validate_document, generate_document,
// format_document, detect_document, list_document_types, generate_person,
// generate_code) over stdio.
// Every tool is derived from the selo registry, so adding a new document
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

// PersonInput is the typed input for the generate_person tool.
type PersonInput struct {
	UF          string `json:"uf,omitempty" jsonschema:"federative unit to pin (e.g. SP); random if omitted"`
	Count       int    `json:"count,omitempty" jsonschema:"how many people to generate, defaults to 1"`
	WithVehicle bool   `json:"with_vehicle,omitempty" jsonschema:"also generate a linked vehicle (plate + renavam)"`
	WithCompany bool   `json:"with_company,omitempty" jsonschema:"also generate a linked company (cnpj)"`
	Formatted   bool   `json:"formatted,omitempty" jsonschema:"return documents in canonical masked form"`
}

// PersonOutput is the typed output for the generate_person tool.
type PersonOutput struct {
	People []selo.Person `json:"people" jsonschema:"synthetic identities; all documents valid and UF-consistent"`
}

// GenerateCodeInput is the typed input for the generate_code tool.
type GenerateCodeInput struct {
	Lang string `json:"lang" jsonschema:"target language: ts, js, ruby, java, or csharp"`
	Kind string `json:"kind" jsonschema:"document kind to generate code for, e.g. cpf"`
}

// GenerateCodeFile is one generated artifact: a relative path and its contents.
type GenerateCodeFile struct {
	Path    string `json:"path" jsonschema:"path of the generated file, relative to the output root"`
	Content string `json:"content" jsonschema:"the generated file contents"`
}

// GenerateCodeOutput is the typed output for the generate_code tool.
type GenerateCodeOutput struct {
	Files []GenerateCodeFile `json:"files" jsonschema:"the generated module/test/vector files"`
}

// kindEnum returns one enum value per registered kind, for the jsonschema
// "kind" field. Sourced from the registry so it stays in sync automatically.
func kindEnum() []any {
	kinds := selo.Kinds()

	out := make([]any, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, k.String())
	}

	return out
}

// errResult builds a tool result flagged as an error with a human-readable
// message. The typed Out zero value is returned alongside to satisfy the
// three-value handler signature required by mcp.AddTool.
func errResult[Out any](msg string) (*mcp.CallToolResult, Out, error) { //nolint:unparam // Out is always zero but required by the generic handler signature
	var zero Out

	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, zero, nil
}

func validateHandler(_ context.Context, _ *mcp.CallToolRequest, in ValidateInput) (*mcp.CallToolResult, ValidateOutput, error) {
	kind := selo.Kind(in.Kind)

	doc, ok := selo.Get(kind)
	if !ok {
		return errResult[ValidateOutput](fmt.Sprintf("unknown document kind %q", in.Kind))
	}

	var out ValidateOutput

	if in.UF != "" {
		scoped, isScoped := doc.(selo.UFScoped)
		if !isScoped {
			return errResult[ValidateOutput](fmt.Sprintf("kind %q does not accept a uf", in.Kind))
		}

		valid, err := scoped.ValidateUF(in.Value, selo.UF(in.UF))
		if err != nil {
			return errResult[ValidateOutput](err.Error())
		}

		out.Valid = valid
	} else {
		valid, err := selo.Validate(kind, in.Value)
		if err != nil {
			return errResult[ValidateOutput](err.Error())
		}

		out.Valid = valid
	}

	if res, hasOrigin := doc.(selo.OriginResolver); hasOrigin && out.Valid {
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
		v, err := selo.Generate(selo.Kind(in.Kind))
		if err != nil {
			return errResult[GenerateOutput](err.Error())
		}

		values = append(values, v)
	}

	out := GenerateOutput{Values: values}

	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

func formatHandler(_ context.Context, _ *mcp.CallToolRequest, in FormatInput) (*mcp.CallToolResult, FormatOutput, error) {
	formatted, err := selo.Format(selo.Kind(in.Kind), in.Value)
	if err != nil {
		return errResult[FormatOutput](err.Error())
	}

	out := FormatOutput{Formatted: formatted}

	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

func detectHandler(_ context.Context, _ *mcp.CallToolRequest, in DetectInput) (*mcp.CallToolResult, DetectOutput, error) {
	kind, ok := selo.Detect(in.Value)

	out := DetectOutput{Kind: kind.String(), Valid: ok}
	if !ok {
		out.Kind = ""
	}

	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

func listHandler(_ context.Context, _ *mcp.CallToolRequest, _ ListInput) (*mcp.CallToolResult, ListOutput, error) {
	kinds := selo.Kinds()

	names := make([]string, 0, len(kinds))
	for _, k := range kinds {
		names = append(names, k.String())
	}

	out := ListOutput{Kinds: names}

	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

func personHandler(_ context.Context, _ *mcp.CallToolRequest, in PersonInput) (*mcp.CallToolResult, PersonOutput, error) {
	count := in.Count
	if count <= 0 {
		count = 1
	}

	var opts []selo.PersonOption

	if in.UF != "" {
		uf := selo.UF(in.UF)
		if !uf.Valid() {
			return errResult[PersonOutput](fmt.Sprintf("invalid uf %q", in.UF))
		}

		opts = append(opts, selo.WithUF(uf))
	}

	if in.WithVehicle {
		opts = append(opts, selo.WithVehicle())
	}

	if in.WithCompany {
		opts = append(opts, selo.WithCompany())
	}

	if in.Formatted {
		opts = append(opts, selo.Formatted())
	}

	people := make([]selo.Person, count)
	for i := range people {
		people[i] = selo.GeneratePerson(opts...)
	}

	out := PersonOutput{People: people}

	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

// generateCodeHandler is the generate_code tool. It validates the requested
// language/kind and asks the codegen package to render the file set. In M1 no
// language emitters are registered, so it reports "emitter not yet registered"
// cleanly (as a tool error) rather than failing the RPC. M2+ register emitters
// and this returns real files with no handler change here.
func generateCodeHandler(_ context.Context, _ *mcp.CallToolRequest, in GenerateCodeInput) (*mcp.CallToolResult, GenerateCodeOutput, error) {
	if !codegen.IsSupportedLang(in.Lang) {
		return errResult[GenerateCodeOutput](fmt.Sprintf(
			"unsupported language %q (supported: %s)", in.Lang, strings.Join(codegen.SupportedLangStrings(), ", ")))
	}

	kind := selo.Kind(in.Kind)
	if _, ok := codegen.PlanFor(kind); !ok {
		return errResult[GenerateCodeOutput](fmt.Sprintf("unknown document kind %q", in.Kind))
	}

	files, err := codegen.Generate(codegen.Lang(in.Lang), kind)
	if err != nil {
		// Includes the M1 "emitter for %q not yet registered" message.
		return errResult[GenerateCodeOutput](err.Error())
	}

	out := GenerateCodeOutput{Files: make([]GenerateCodeFile, 0, len(files))}
	for _, f := range files {
		out.Files = append(out.Files, GenerateCodeFile{Path: f.Path, Content: string(f.Content)})
	}

	return &mcp.CallToolResult{StructuredContent: out}, out, nil
}

// NewServer builds an MCP server with all selo tools registered.
// version is stamped into the server Implementation (use build info).
func NewServer(version string) *mcp.Server {
	if version == "" {
		version = "dev"
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := mcp.NewServer(
		&mcp.Implementation{Name: selo.MCPServerName, Version: version},
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

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "generate_person",
		Description: "Generate coherent synthetic Brazilian identities: all documents valid and UF-consistent. Synthetic test data only — never real PII.",
	}, personHandler)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "generate_code",
		Description: "Generate validate/format/origin code (with golden vectors and a test) for a Brazilian document kind in a target language (ts, js, ruby, java, csharp).",
	}, generateCodeHandler)

	return srv
}

// Serve runs the MCP server over stdio until the context is cancelled or
// stdin closes. The logger writes to stderr because stdout carries the
// JSON-RPC stream.
func Serve(ctx context.Context, version string) error {
	srv := NewServer(version)
	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("selo mcp: %w", err)
	}

	return nil
}
