package selo

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// Person is a coherent synthetic Brazilian identity: every document is valid and
// the geolocatable ones (CPF region, Voter ID UF code, phone DDD, CEP range) all
// agree on the same federative unit (UF). Generated for testing/fixtures only —
// never real PII (LGPD: synthetic data).
type Person struct {
	Name    string
	Email   string
	UF      UF
	CPF     string
	RG      string // only when UF is SP or RJ (the implemented RG algorithms); else ""
	CNH     string
	PIS     string
	Renavam string
	VoterID string
	CNS     string
	CEP     string
	Phone   string
	PIXKeys []string
	Vehicle *Vehicle // populated only with WithVehicle()
	Company *Company // populated only with WithCompany()
}

// Vehicle is a synthetic vehicle linked to a Person.
type Vehicle struct {
	Plate   string
	Renavam string
}

// Company is a synthetic company (CNPJ) linked to a Person.
type Company struct {
	Name string
	CNPJ string
}

type personOpts struct {
	uf          UF
	ufSet       bool
	withVehicle bool
	withCompany bool
	formatted   bool
}

// PersonOption configures GeneratePerson.
type PersonOption func(*personOpts)

// WithUF pins the person's federative unit (drives all geo-consistent documents).
// When omitted, a random implemented UF is chosen.
func WithUF(uf UF) PersonOption { return func(o *personOpts) { o.uf = uf; o.ufSet = true } }

// WithVehicle also generates a linked Vehicle (plate + RENAVAM).
func WithVehicle() PersonOption { return func(o *personOpts) { o.withVehicle = true } }

// WithCompany also generates a linked Company (CNPJ + name).
func WithCompany() PersonOption { return func(o *personOpts) { o.withCompany = true } }

// Formatted returns every document in its canonical masked form instead of raw digits.
func Formatted() PersonOption { return func(o *personOpts) { o.formatted = true } }

// cpfRegionByUF maps a UF to the CPF 9th-digit fiscal region (Receita Federal).
var cpfRegionByUF = map[UF]int{
	UFRS: 0,
	UFDF: 1, UFGO: 1, UFMS: 1, UFMT: 1, UFTO: 1,
	UFAC: 2, UFAM: 2, UFAP: 2, UFPA: 2, UFRO: 2, UFRR: 2,
	UFCE: 3, UFMA: 3, UFPI: 3,
	UFAL: 4, UFPB: 4, UFPE: 4, UFRN: 4,
	UFBA: 5, UFSE: 5,
	UFMG: 6,
	UFES: 7, UFRJ: 7,
	UFSP: 8,
	UFPR: 9, UFSC: 9,
}

// voterCodeByUF maps a UF to its Título Eleitoral UF code (TSE ordering).
var voterCodeByUF = map[UF]int{
	UFSP: 1, UFMG: 2, UFRJ: 3, UFRS: 4, UFBA: 5, UFPR: 6, UFCE: 7, UFPE: 8, UFSC: 9,
	UFGO: 10, UFMA: 11, UFPB: 12, UFPA: 13, UFES: 14, UFPI: 15, UFRN: 16, UFAL: 17,
	UFMT: 18, UFMS: 19, UFDF: 20, UFSE: 21, UFAM: 22, UFRO: 23, UFAC: 24, UFAP: 25,
	UFRR: 26, UFTO: 27,
}

var personFirstNames = []string{
	"João", "Maria", "José", "Ana", "Carlos", "Beatriz", "Pedro", "Juliana",
	"Lucas", "Fernanda", "Rafael", "Camila", "Bruno", "Larissa", "Gabriel", "Mariana",
}

var personSurnames = []string{
	"Silva", "Santos", "Oliveira", "Souza", "Lima", "Pereira", "Ferreira",
	"Costa", "Rodrigues", "Almeida", "Nascimento", "Carvalho",
}

var personCompanySuffixes = []string{"ME", "LTDA", "S.A.", "EIRELI", "Comércio", "Serviços"}

// asciiFold replaces the accented characters used in the name lists with their
// ASCII equivalents, for building email local-parts.
var asciiFold = strings.NewReplacer(
	"ã", "a", "á", "a", "â", "a", "à", "a",
	"é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c",
)

// GeneratePerson returns a coherent synthetic Person whose geolocatable documents
// all resolve to the same UF. By default the UF is random and documents are raw
// (unformatted). Use the options to pin the UF, add a vehicle/company, or format.
func GeneratePerson(opts ...PersonOption) Person {
	o := personOpts{}
	for _, fn := range opts {
		fn(&o)
	}

	uf := o.uf
	if !o.ufSet {
		impl := personUFs()
		uf = impl[rand.IntN(len(impl))]
	}

	first := personFirstNames[rand.IntN(len(personFirstNames))]
	last := personSurnames[rand.IntN(len(personSurnames))]
	name := first + " " + last
	email := fmt.Sprintf("%s.%s%d@example.com.br",
		strings.ToLower(asciiFold.Replace(first)),
		strings.ToLower(asciiFold.Replace(last)),
		rand.IntN(1000))

	cpf := genCPFForUF(uf)
	phone := genPhoneForUF(uf)
	cep := genCEPForUF(uf)
	voter := genVoterIDForUF(uf)

	cnh := NewCNH().Generate()
	pis := NewPIS().Generate()
	renavam := NewRenavam().Generate()
	cns := NewCNS().Generate()

	rg := ""
	if _, ok := rgImplemented[uf]; ok {
		rg = NewRG().Generate() // already masked
	}

	pix := []string{
		cpf,                       // CPF key
		"+55" + onlyDigits(phone), // phone key (E.164)
		email,                     // email key
		NewPIX().Generate(),       // a random EVP (UUIDv4) key
	}

	p := Person{
		Name: name, Email: email, UF: uf,
		CPF: cpf, RG: rg, CNH: cnh, PIS: pis, Renavam: renavam,
		VoterID: voter, CNS: cns, CEP: cep, Phone: phone, PIXKeys: pix,
	}

	if o.withVehicle {
		p.Vehicle = &Vehicle{Plate: (&Plate{Mercosul: true}).Generate(), Renavam: NewRenavam().Generate()}
	}
	if o.withCompany {
		p.Company = &Company{
			Name: last + " " + personCompanySuffixes[rand.IntN(len(personCompanySuffixes))],
			CNPJ: NewCNPJ().Generate(),
		}
	}

	if o.formatted {
		formatPerson(&p)
	}
	return p
}

// personUFs returns the UFs for which a fully consistent person can be built
// (those present in every geo table: CPF region, voter code, CEP, DDD). All 27
// qualify; sorted for determinism.
func personUFs() []UF {
	out := make([]UF, 0, len(voterCodeByUF))
	for _, uf := range AllUFs() {
		if _, ok := voterCodeByUF[uf]; ok {
			out = append(out, uf)
		}
	}
	return out
}

// genCPFForUF returns a valid CPF whose 9th digit matches uf's fiscal region.
func genCPFForUF(uf UF) string {
	region := cpfRegionByUF[uf]
	c := NewCPF()
	for {
		v := c.Generate()
		if len(v) == CpfLength && int(v[8]-'0') == region {
			return v
		}
	}
}

// genVoterIDForUF returns a valid Título Eleitoral whose embedded UF code matches uf.
func genVoterIDForUF(uf UF) string {
	code := fmt.Sprintf("%02d", voterCodeByUF[uf])
	v := NewVoterID()
	for {
		got := v.Generate()
		if len(got) == 12 && got[8:10] == code {
			return got
		}
	}
}

// genPhoneForUF builds a valid 9-digit mobile for one of uf's DDD area codes.
func genPhoneForUF(uf UF) string {
	ddds := dddsForUF(uf)
	ddd := ddds[rand.IntN(len(ddds))]
	var sb strings.Builder
	fmt.Fprintf(&sb, "%02d9", ddd) // DDD + mobile leading 9
	for i := 0; i < 8; i++ {
		sb.WriteByte(byte('0' + rand.IntN(10)))
	}
	return sb.String()
}

// dddsForUF returns the sorted DDD codes belonging to uf.
func dddsForUF(uf UF) []int {
	var out []int
	for _, ddd := range ddds {
		if dddUFTable[ddd] == uf {
			out = append(out, ddd)
		}
	}
	return out
}

// genCEPForUF builds a valid 8-digit CEP within one of uf's postal ranges.
func genCEPForUF(uf UF) string {
	var ranges [][2]int
	for _, r := range cepPrefixRanges {
		if r.uf == uf {
			ranges = append(ranges, [2]int{r.from, r.to})
		}
	}
	r := ranges[rand.IntN(len(ranges))]
	prefix := r[0] + rand.IntN(r[1]-r[0]+1)
	return fmt.Sprintf("%03d%05d", prefix, rand.IntN(100000))
}

// formatPerson rewrites each document field into its canonical masked form,
// leaving fields untouched when the value cannot be formatted.
func formatPerson(p *Person) {
	fmtOr := func(d Document, v string) string {
		if v == "" {
			return v
		}
		if s, err := d.Format(v); err == nil {
			return s
		}
		return v
	}
	p.CPF = fmtOr(NewCPF(), p.CPF)
	p.CNH = fmtOr(NewCNH(), p.CNH)
	p.PIS = fmtOr(NewPIS(), p.PIS)
	p.Renavam = fmtOr(NewRenavam(), p.Renavam)
	p.VoterID = fmtOr(NewVoterID(), p.VoterID)
	p.CNS = fmtOr(NewCNS(), p.CNS)
	p.CEP = fmtOr(NewCEP(), p.CEP)
	p.Phone = fmtOr(NewPhone(), p.Phone)
	// RG is already masked by Generate; Company CNPJ formatted if present.
	if p.Company != nil {
		p.Company.CNPJ = fmtOr(NewCNPJ(), p.Company.CNPJ)
	}
	if p.Vehicle != nil {
		p.Vehicle.Renavam = fmtOr(NewRenavam(), p.Vehicle.Renavam)
	}
}
