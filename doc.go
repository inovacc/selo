// Package selo provides validation, generation, formatting, and geolocation for
// Brazilian documents — CPF, CNPJ (incl. alphanumeric), CNH, PIS/PASEP/NIS,
// RENAVAM, Título Eleitoral, CEP, phone, license plate, CNS, RG (SP),
// Inscrição Estadual (SP), and PIX keys — behind a common Document interface and
// a self-registering type registry. The CLI and MCP server derive their surfaces
// from that registry; the compat subpackage mirrors paemuri/brdoc for drop-in
// migration.
//
// This package implements official algorithms from SERPRO (Serviço Federal de Processamento de Dados)
// for both CPF (Cadastro de Pessoas Físicas) and alphanumeric CNPJ (Cadastro Nacional de Pessoa Jurídica)
// validation, alongside the per-type check-digit algorithms for the remaining kinds.
//
// # CPF Features
//
// The CPF validator supports:
//   - Validation with check digit verification
//   - Generation of valid random CPFs
//   - Formatting (XXX.XXX.XXX-XX)
//   - State/region identification
//   - Detection of invalid patterns (all same digits)
//
// # CNPJ Features
//
// The CNPJ validator supports alphanumeric format per SERPRO specification:
//   - Validation of alphanumeric CNPJs (12 alphanumeric + 2 numeric check digits)
//   - Generation of valid random alphanumeric CNPJs
//   - Optional generation of legacy numeric-only CNPJs (14 digits)
//   - Formatting (XX.XXX.XXX/XXXX-XX)
//   - Modulo 11 check digit calculation
//
// # Basic Usage
//
// CPF validation example:
//
//	cpf := selo.NewCPF()
//	if cpf.Validate("123.456.789-09") {
//	    fmt.Println("Valid CPF")
//	    origin := cpf.CheckOrigin("123.456.789-09")
//	    fmt.Printf("Issued in: %s\n", origin)
//	}
//
// CNPJ validation example:
//
//	cnpj := selo.NewCNPJ()
//	if cnpj.Validate("12.ABC.345/01DE-35") {
//	    fmt.Println("Valid CNPJ")
//	}
//
// Auto-detection example (ValidateDocument is deprecated; use Detect + Validate):
//
//	kind, ok := selo.Detect("123.456.789-09")
//	if ok {
//	    valid, _ := selo.Validate(kind, "123.456.789-09")
//	    fmt.Printf("Type: %s, Valid: %v\n", kind, valid)
//	}
//
// # Generation
//
// Generate valid documents:
//
//	cpf := selo.NewCPF()
//	newCPF := cpf.Generate()  // Returns unformatted CPF
//
//	cnpj := selo.NewCNPJ()
//	newCNPJ := cnpj.Generate()  // Returns unformatted alphanumeric CNPJ
//
//	// Legacy numeric-only (14 digits)
//	legacy := cnpj.GenerateLegacy() // Returns unformatted numeric-only CNPJ
//
// # Formatting
//
// Format documents to standard Brazilian format:
//
//	cpf := selo.NewCPF()
//	formatted := cpf.Format("12345678909")  // Returns "123.456.789-09"
//
//	cnpj := selo.NewCNPJ()
//	formatted, err := cnpj.Format("12ABC34501DE35")  // Returns "12.ABC.345/01DE-35"
//
// # CNPJ Alphanumeric Specification
//
// The library implements the official SERPRO specification:
//   - Character mapping: 0-9 → 0-9, A-Z → 17-42 (ASCII - 48)
//   - Weight distribution: 2-9, repeating from right to left
//   - Check digit calculation: modulo 11
//   - Special rule: If remainder is 0 or 1, check digit is 0
//
// For more information, see: https://github.com/inovacc/selo
package selo
