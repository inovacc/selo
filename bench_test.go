package selo

import "testing"

// Package-level sinks prevent the compiler from eliminating benchmarked calls.
var (
	benchBool   bool
	benchStr    string
	benchKind   Kind
	benchOK     bool
	benchPerson Person
)

func BenchmarkCPFValidate(b *testing.B) {
	cpf := NewCPF()

	b.ReportAllocs()

	for b.Loop() {
		benchBool = cpf.Validate("529.982.247-25")
	}
}

func BenchmarkCPFFormat(b *testing.B) {
	cpf := NewCPF()

	b.ReportAllocs()

	for b.Loop() {
		benchStr, _ = cpf.Format("52998224725")
	}
}

func BenchmarkCPFGenerate(b *testing.B) {
	cpf := NewCPF()

	b.ReportAllocs()

	for b.Loop() {
		benchStr = cpf.Generate()
	}
}

func BenchmarkCNPJValidate(b *testing.B) {
	cnpj := NewCNPJ()

	b.ReportAllocs()

	for b.Loop() {
		benchBool = cnpj.Validate("39.591.842/0000-10")
	}
}

func BenchmarkDetect(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		benchKind, benchOK = Detect("529.982.247-25")
	}
}

func BenchmarkRegistryValidate(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		benchBool, _ = Validate(KindCPF, "52998224725")
	}
}

func BenchmarkGenerateRand(b *testing.B) {
	r := NewSeededRand(1)

	b.ReportAllocs()

	for b.Loop() {
		benchStr, _ = GenerateRand(KindCPF, r)
	}
}

func BenchmarkGeneratePerson(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		benchPerson = GeneratePerson(WithUF(UFSP))
	}
}

func BenchmarkGeneratePersonSeeded(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		benchPerson = GeneratePerson(WithUF(UFSP), WithSeed(42))
	}
}
