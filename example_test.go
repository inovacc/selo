package selo

import "fmt"

func ExampleCPF_Validate() {
	fmt.Println(NewCPF().Validate("52998224725"))
	// Output: true
}

func ExampleCNPJ_Validate() {
	fmt.Println(NewCNPJ().Validate("39591842000010"))
	// Output: true
}

func ExampleCNH_Validate() {
	fmt.Println(NewCNH().Validate("12345678900"))
	// Output: true
}

func ExamplePIS_Format() {
	formatted, _ := NewPIS().Format("12001234564")
	fmt.Println(formatted)
	// Output: 120.01234.56-4
}

func ExampleRenavam_Validate() {
	fmt.Println(NewRenavam().Validate("12345678900"))
	// Output: true
}

func ExampleCEP_Origin() {
	origin, _ := NewCEP().Origin("01310-100")
	fmt.Println(origin)
	// Output: SP
}

func ExamplePhone_Origin() {
	origin, _ := NewPhone().Origin("11987654321")
	fmt.Println(origin)
	// Output: SP
}

func ExamplePlate_Validate() {
	fmt.Println(NewPlate().Validate("ABC1D23"))
	// Output: true
}

func ExamplePIX_Validate() {
	// A CPF is a valid PIX key.
	fmt.Println(NewPIX().Validate("52998224725"))
	// Output: true
}

// ExampleDetect shows auto-detecting a document's kind from its raw string.
func ExampleDetect() {
	kind, ok := Detect("529.982.247-25")
	fmt.Println(kind, ok)
	// Output: cpf true
}

// ExampleValidate uses the registry to validate by kind without constructing a
// concrete type.
func ExampleValidate() {
	ok, err := Validate(KindCPF, "52998224725")
	fmt.Println(ok, err)
	// Output: true <nil>
}

// ExampleGeneratePerson generates a deterministic, UF-consistent synthetic
// Brazilian identity suitable for test fixtures. WithSeed makes it reproducible.
func ExampleGeneratePerson() {
	p := GeneratePerson(WithUF(UFSP), WithSeed(42))
	fmt.Println(p.UF)
	// Output: SP
}
