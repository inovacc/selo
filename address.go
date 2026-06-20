package selo

import (
	"fmt"
	"math/rand/v2"
)

// Address is a synthetic, UF-consistent Brazilian street address attached to a
// Person. The City is a real municipality in the person's UF; the CEP equals
// Person.CEP (same generated value) and falls inside that UF's postal range.
// Street/Number/Neighborhood are synthesized — realistic in shape but not a real
// registered logradouro. Synthetic data only; never real PII.
type Address struct {
	Street       string `json:"street"`       // e.g. "Rua das Flores" / "Avenida Brasil"
	Number       string `json:"number"`       // e.g. "1234" or "s/n" (sem número)
	Neighborhood string `json:"neighborhood"` // e.g. "Centro"
	City         string `json:"city"`         // real municipality within UF
	UF           UF     `json:"uf"`           // same as Person.UF
	CEP          string `json:"cep"`          // equals Person.CEP (raw or formatted to match)
}

// genAddressForUFRand builds a UF-consistent synthetic Address: a real city in
// uf, a synthesized logradouro/neighborhood, and the already-generated cep.
// All randomness flows through r so WithSeed/WithRand stay deterministic. The
// draw order (city, street-type, surname, number, [s/n coin], neighborhood) is
// fixed and documented so future edits don't silently reshuffle the stream.
func genAddressForUFRand(uf UF, cep string, r *rand.Rand) *Address {
	cities := citiesByUF[uf]
	city := cities[r.IntN(len(cities))]

	street := pickWeighted(logradouroTypes, r) + " " + personSurnames[r.IntN(len(personSurnames))]

	number := fmt.Sprintf("%d", r.IntN(2000)+1)
	if r.IntN(20) == 0 {
		number = "s/n"
	}

	neighborhood := pickWeighted(neighborhoodTokens, r)

	return &Address{
		Street:       street,
		Number:       number,
		Neighborhood: neighborhood,
		City:         city,
		UF:           uf,
		CEP:          cep,
	}
}

// pickWeighted returns one token value, biased by weight, drawn from r. It
// consumes exactly one draw (r.IntN) regardless of the slice contents.
func pickWeighted(toks []weightedToken, r *rand.Rand) string {
	total := 0
	for _, t := range toks {
		total += t.weight
	}

	n := r.IntN(total)
	for _, t := range toks {
		if n < t.weight {
			return t.value
		}

		n -= t.weight
	}

	return toks[len(toks)-1].value // unreachable
}
