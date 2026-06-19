package selo

import "testing"

func TestKind_String(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindCPF, "cpf"},
		{KindCNPJ, "cnpj"},
		{KindCNH, "cnh"},
		{KindPIS, "pis"},
		{KindRenavam, "renavam"},
		{KindVoterID, "voter_id"},
		{KindCEP, "cep"},
		{KindPhone, "phone"},
		{KindPlate, "plate"},
		{KindCNS, "cns"},
		{KindRG, "rg"},
		{KindPIX, "pix"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Fatalf("Kind.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// stubDoc proves the Document/OriginResolver/UFScoped method sets compile as declared.
type stubDoc struct{}

func (stubDoc) Kind() Kind                          { return KindCPF }
func (stubDoc) Validate(string) bool                { return false }
func (stubDoc) Generate() string                    { return "" }
func (stubDoc) Format(string) (string, error)       { return "", nil }
func (stubDoc) Origin(string) (string, error)       { return "", nil }
func (stubDoc) ValidateUF(string, UF) (bool, error) { return false, nil }
func (stubDoc) ImplementedUFs() []UF                { return nil }

var (
	_ Document       = stubDoc{}
	_ OriginResolver = stubDoc{}
	_ UFScoped       = stubDoc{}
)
