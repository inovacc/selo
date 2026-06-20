package selo

import "math/rand/v2"

// Kind is the stable identifier for a document type, e.g. "cpf".
type Kind string

// Document kind identifiers. Values are stable and used by the CLI and MCP adapters.
const (
	KindCPF     Kind = "cpf"
	KindCNPJ    Kind = "cnpj"
	KindCNH     Kind = "cnh"
	KindPIS     Kind = "pis"
	KindRenavam Kind = "renavam"
	KindVoterID Kind = "voter_id" // Título Eleitoral
	KindCEP     Kind = "cep"
	KindPhone   Kind = "phone"
	KindPlate   Kind = "plate"
	KindCNS     Kind = "cns"
	KindRG      Kind = "rg"
	KindPIX     Kind = "pix"
	KindIE      Kind = "ie" // Inscrição Estadual (state tax registration)
)

// String returns the stable string identifier of the Kind.
func (k Kind) String() string { return string(k) }

// UF is a Brazilian federative unit (state) two-letter code, e.g. "SP".
// The full constant set and helpers are defined in uf.go.
type UF string

// Document is implemented by every document type in the toolkit.
type Document interface {
	// Kind returns the stable identifier of this document type.
	Kind() Kind
	// Validate reports whether value is a well-formed document of this Kind
	// (formatted or unformatted input is accepted).
	Validate(value string) bool
	// Generate returns a freshly generated valid, unformatted document.
	Generate() string
	// Format returns value in the canonical masked representation for this Kind,
	// or a sentinel error (see errors.go) when value cannot be formatted.
	Format(value string) (string, error)
}

// OriginResolver is the optional capability for types that can resolve a
// geographic origin (CPF region, CEP/phone/voter UF). Discovered via type assertion.
type OriginResolver interface {
	Origin(value string) (string, error)
}

// UFScoped is the optional capability for types whose validation depends on a
// federative unit (notably RG). Discovered via type assertion.
type UFScoped interface {
	ValidateUF(value string, uf UF) (bool, error)
	ImplementedUFs() []UF
}

// RandGenerator is the optional capability for types that can generate from a
// caller-supplied random source (for deterministic fixtures). Discovered via type assertion.
type RandGenerator interface {
	GenerateRand(r *rand.Rand) string
}
