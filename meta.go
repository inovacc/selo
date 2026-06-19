package selo

// Brand-bearing strings live here in one place so a future rebrand
// (/branding:names) is a near-mechanical change. The Go package name stays
// "brdoc" (a domain term, not a brand).
const (
	// AppName is the human-facing application name.
	AppName = "Selo"
	// CLIUse is the Cobra root command Use field (the binary name).
	CLIUse = "selo"
	// CLIShort is the Cobra root command short description.
	CLIShort = "Brazilian documents utilities (CPF/CNPJ and more)"
	// MCPServerName is the MCP server Implementation Name.
	MCPServerName = "selo"
)
