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
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	UF      UF       `json:"uf"`
	CPF     string   `json:"cpf"`
	RG      string   `json:"rg,omitempty"` // only when UF is SP (the implemented RG algorithm); else ""
	IE      string   `json:"ie,omitempty"` // Inscrição Estadual; only when the UF has a verified IE algorithm (SP, MG, RS, PR)
	CNH     string   `json:"cnh"`
	PIS     string   `json:"pis"`
	Renavam string   `json:"renavam"`
	VoterID string   `json:"voter_id"`
	CNS     string   `json:"cns"`
	CEP     string   `json:"cep"`
	Phone   string   `json:"phone"`
	PIXKeys []string `json:"pix_keys"`
	Vehicle *Vehicle `json:"vehicle,omitempty"` // populated only with WithVehicle()
	Company *Company `json:"company,omitempty"` // populated only with WithCompany()
}

// Vehicle is a synthetic vehicle linked to a Person.
type Vehicle struct {
	Plate   string `json:"plate"`
	Renavam string `json:"renavam"`
}

// Company is a synthetic company (CNPJ) linked to a Person.
type Company struct {
	Name string `json:"name"`
	CNPJ string `json:"cnpj"`
}

type personOpts struct {
	uf          UF
	ufSet       bool
	withVehicle bool
	withCompany bool
	formatted   bool
	r           *rand.Rand // nil = use global random
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

// NewSeededRand returns a deterministic *rand.Rand seeded from seed: the same
// seed always yields the same sequence. Pass it to WithRand (or to a document
// type's GenerateRand) for reproducible output. Sharing one source across
// several GeneratePerson calls keeps the whole batch reproducible while still
// yielding distinct people — the stream advances between draws.
func NewSeededRand(seed int64) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed), uint64(seed>>32)))
}

// WithSeed pins the random source to a deterministic seed. Same seed + same
// options always produces the same Person (useful for a single test fixture).
// Note: each GeneratePerson call re-runs this option and re-seeds, so reusing
// one WithSeed across a batch yields identical people — for a reproducible
// batch of distinct people, build one source with NewSeededRand and pass it
// via WithRand.
func WithSeed(seed int64) PersonOption {
	return func(o *personOpts) {
		o.r = NewSeededRand(seed)
	}
}

// WithRand supplies a caller-owned *rand.Rand. The caller is responsible for
// seeding; GeneratePerson will consume from it sequentially.
func WithRand(r *rand.Rand) PersonOption {
	return func(o *personOpts) { o.r = r }
}

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

	r := o.r
	if r == nil {
		r = newRand()
	}

	uf := o.uf
	if !o.ufSet {
		impl := personUFs()
		uf = impl[r.IntN(len(impl))]
	}

	first := personFirstNames[r.IntN(len(personFirstNames))]
	last := personSurnames[r.IntN(len(personSurnames))]
	name := first + " " + last
	email := fmt.Sprintf("%s.%s%d@example.com.br",
		strings.ToLower(asciiFold.Replace(first)),
		strings.ToLower(asciiFold.Replace(last)),
		r.IntN(1000))

	cpf := genCPFForUFRand(uf, r)
	phone := genPhoneForUFRand(uf, r)
	cep := genCEPForUFRand(uf, r)
	voter := genVoterIDForUFRand(uf, r)

	cnh := NewCNH().GenerateRand(r)
	pis := NewPIS().GenerateRand(r)
	renavam := NewRenavam().GenerateRand(r)
	cns := NewCNS().GenerateRand(r)

	rg := ""
	if _, ok := rgImplemented[uf]; ok {
		rg = NewRG().GenerateRand(r)
	}

	// IE is UF-scoped; populate it only when the person's UF has a verified IE
	// algorithm (SP, MG, RS, PR), generating for that specific UF so it stays
	// UF-consistent even as more UFs are added.
	ie := ""
	if algo, ok := ieTable[uf]; ok && algo.generateRand != nil {
		ie = algo.generateRand(r)
	}

	pix := []string{
		cpf,                       // CPF key
		"+55" + onlyDigits(phone), // phone key (E.164)
		email,                     // email key
		NewPIX().GenerateRand(r),  // a random EVP (UUIDv4) key
	}

	p := Person{
		Name: name, Email: email, UF: uf,
		CPF: cpf, RG: rg, IE: ie, CNH: cnh, PIS: pis, Renavam: renavam,
		VoterID: voter, CNS: cns, CEP: cep, Phone: phone, PIXKeys: pix,
	}

	if o.withVehicle {
		p.Vehicle = &Vehicle{
			Plate:   (&Plate{Mercosul: true}).GenerateRand(r),
			Renavam: NewRenavam().GenerateRand(r),
		}
	}

	if o.withCompany {
		p.Company = &Company{
			Name: last + " " + personCompanySuffixes[r.IntN(len(personCompanySuffixes))],
			CNPJ: NewCNPJ().GenerateRand(r),
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

// genCPFForUFRand returns a valid CPF whose 9th digit matches uf's fiscal region,
// using the supplied random source.
func genCPFForUFRand(uf UF, r *rand.Rand) string {
	region := cpfRegionByUF[uf]

	c := NewCPF()
	for {
		v := c.GenerateRand(r)
		if len(v) == CpfLength && int(v[8]-'0') == region {
			return v
		}
	}
}

// genVoterIDForUFRand returns a valid Título Eleitoral whose embedded UF code matches uf,
// using the supplied random source.
func genVoterIDForUFRand(uf UF, r *rand.Rand) string {
	code := fmt.Sprintf("%02d", voterCodeByUF[uf])

	v := NewVoterID()
	for {
		got := v.GenerateRand(r)
		if len(got) == 12 && got[8:10] == code {
			return got
		}
	}
}

// genPhoneForUFRand builds a valid 9-digit mobile for one of uf's DDD area codes,
// using the supplied random source.
func genPhoneForUFRand(uf UF, r *rand.Rand) string {
	dddList := dddsForUF(uf)
	ddd := dddList[r.IntN(len(dddList))]

	var sb strings.Builder
	fmt.Fprintf(&sb, "%02d9", ddd) // DDD + mobile leading 9

	for range 8 {
		sb.WriteByte(byte('0' + r.IntN(10)))
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

// genCEPForUFRand builds a valid 8-digit CEP within one of uf's postal ranges,
// using the supplied random source.
func genCEPForUFRand(uf UF, r *rand.Rand) string {
	var ranges [][2]int

	for _, rng := range cepPrefixRanges {
		if rng.uf == uf {
			ranges = append(ranges, [2]int{rng.from, rng.to})
		}
	}

	rng := ranges[r.IntN(len(ranges))]
	prefix := rng[0] + r.IntN(rng[1]-rng[0]+1)

	return fmt.Sprintf("%03d%05d", prefix, r.IntN(100000))
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
