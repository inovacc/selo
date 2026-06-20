package selo

import (
	"fmt"
	"math/rand/v2"
)

// CepLength is the number of digits in a CEP (postal code).
const CepLength = 8

// cepPrefixRanges maps each UF to the inclusive [from,to] range of the
// first-three-digit CEP prefix (cep / 100000). Source: Correios allocation.
// Some UFs have multiple disjoint blocks, all listed so Generate can pick any
// and Origin can resolve every block: AM (690–692 and 694–698) and DF
// (700–727 and 730–736, with GO interleaved at 728–729 and 737–767).
var cepPrefixRanges = []struct {
	uf       UF
	from, to int
}{
	{UFSP, 10, 199}, {UFRJ, 200, 289}, {UFES, 290, 299},
	{UFMG, 300, 399}, {UFBA, 400, 489}, {UFSE, 490, 499},
	{UFPE, 500, 569}, {UFAL, 570, 579}, {UFPB, 580, 589},
	{UFRN, 590, 599}, {UFCE, 600, 639}, {UFPI, 640, 649},
	{UFMA, 650, 659}, {UFPA, 660, 688}, {UFAP, 689, 689},
	{UFAM, 690, 692}, {UFRR, 693, 693}, {UFAM, 694, 698},
	{UFAC, 699, 699}, {UFDF, 700, 727}, {UFGO, 728, 729},
	{UFDF, 730, 736}, {UFGO, 737, 767}, {UFRO, 768, 769},
	{UFTO, 770, 779}, {UFMT, 780, 788}, {UFMS, 790, 799},
	{UFPR, 800, 879}, {UFSC, 880, 899}, {UFRS, 900, 999},
}

func init() {
	// Populate the cepRanges stub declared in uf.go (M0-3) with the primary
	// block per UF (first block wins for UFs with multiple blocks, e.g. AM).
	for _, r := range cepPrefixRanges {
		if _, exists := cepRanges[r.uf]; !exists {
			cepRanges[r.uf] = [2]int{r.from, r.to}
		}
	}

	Register(&CEP{})
}

// CEP validates, generates, and formats Brazilian postal codes (8 digits),
// and resolves the issuing federative unit from the numeric prefix.
type CEP struct{}

// NewCEP creates a new CEP instance.
func NewCEP() *CEP { return &CEP{} }

// Kind returns KindCEP.
func (c *CEP) Kind() Kind { return KindCEP }

// cepRangeFor returns the UF whose prefix range contains prefix (cep/100000),
// and ok=false when no range matches.
func cepRangeFor(prefix int) (UF, bool) {
	for _, r := range cepPrefixRanges {
		if prefix >= r.from && prefix <= r.to {
			return r.uf, true
		}
	}

	return "", false
}

// Validate reports whether value is a well-formed CEP whose prefix maps to a UF.
func (c *CEP) Validate(value string) bool {
	d := onlyDigits(value)
	if len(d) != CepLength {
		return false
	}

	prefix := int(d[0]-'0')*100 + int(d[1]-'0')*10 + int(d[2]-'0')
	_, ok := cepRangeFor(prefix)

	return ok
}

// Format masks a CEP as #####-###. It returns ErrInvalidLength when the
// cleaned value does not have exactly CepLength digits.
func (c *CEP) Format(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != CepLength {
		return "", fmt.Errorf("selo: cep needs %d digits, got %d: %w", CepLength, len(d), ErrInvalidLength)
	}

	return d[0:5] + "-" + d[5:8], nil
}

// Origin returns the federative unit (e.g. "SP") whose CEP prefix range
// contains value. It returns ErrInvalidLength on bad length and
// ErrInvalidFormat when the prefix maps to no UF. CEP satisfies OriginResolver.
func (c *CEP) Origin(value string) (string, error) {
	d := onlyDigits(value)
	if len(d) != CepLength {
		return "", fmt.Errorf("selo: cep needs %d digits, got %d: %w", CepLength, len(d), ErrInvalidLength)
	}

	prefix := int(d[0]-'0')*100 + int(d[1]-'0')*10 + int(d[2]-'0')

	uf, ok := cepRangeFor(prefix)
	if !ok {
		return "", fmt.Errorf("selo: cep prefix %03d has no UF: %w", prefix, ErrInvalidFormat)
	}

	return uf.String(), nil
}

// GenerateRand returns a valid 8-digit CEP using the supplied random source.
func (c *CEP) GenerateRand(r *rand.Rand) string {
	rng := cepPrefixRanges[r.IntN(len(cepPrefixRanges))]
	prefix := rng.from + r.IntN(rng.to-rng.from+1)
	suffix := r.IntN(100000)

	return fmt.Sprintf("%03d%05d", prefix, suffix)
}

// Generate returns a random, valid 8-digit CEP (unformatted) by picking a
// real UF prefix range and filling the remaining 5 digits at random.
func (c *CEP) Generate() string { return c.GenerateRand(newRand()) }
