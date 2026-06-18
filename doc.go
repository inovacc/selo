// Package brdoc provides validation, generation, and formatting for Brazilian fiscal documents.
//
// This package implements official algorithms from SERPRO (Serviço Federal de Processamento de Dados)
// for both CPF (Cadastro de Pessoas Físicas) and alphanumeric CNPJ (Cadastro Nacional de Pessoa Jurídica)
// validation.
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
//	cpf := brdoc.NewCPF()
//	if cpf.Validate("123.456.789-09") {
//	    fmt.Println("Valid CPF")
//	    origin := cpf.CheckOrigin("123.456.789-09")
//	    fmt.Printf("Issued in: %s\n", origin)
//	}
//
// CNPJ validation example:
//
//	cnpj := brdoc.NewCNPJ()
//	if cnpj.Validate("12.ABC.345/01DE-35") {
//	    fmt.Println("Valid CNPJ")
//	}
//
// Auto-detection example:
//
//	docType, isValid := brdoc.ValidateDocument("123.456.789-09")
//	fmt.Printf("Type: %s, Valid: %v\n", docType, isValid)
//
// # Generation
//
// Generate valid documents:
//
//	cpf := brdoc.NewCPF()
//	newCPF := cpf.Generate()  // Returns unformatted CPF
//
//	cnpj := brdoc.NewCNPJ()
//	newCNPJ := cnpj.Generate()  // Returns unformatted alphanumeric CNPJ
//
//	// Legacy numeric-only (14 digits)
//	legacy := cnpj.GenerateLegacy() // Returns unformatted numeric-only CNPJ
//
// # Formatting
//
// Format documents to standard Brazilian format:
//
//	cpf := brdoc.NewCPF()
//	formatted := cpf.Format("12345678909")  // Returns "123.456.789-09"
//
//	cnpj := brdoc.NewCNPJ()
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
