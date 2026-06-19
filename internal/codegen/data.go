package codegen

import (
	"sort"

	"github.com/inovacc/selo"
)

// data.go re-exposes the selo data tables that per-language emitters render as
// constants: CEP prefix→UF ranges, DDD→UF, the CPF ninth-digit region map, and
// the voter-ID UF-code region map. Everything comes from the additive selo
// accessors (selo.CEPRanges, selo.DDDtoUF, selo.CPFRegions, selo.VoterUFNames)
// so there is a single source of truth; this layer only reshapes the data into
// stable, deterministic, serializable forms suitable for code generation.

// UFRange is one CEP prefix range mapped to a UF. From/To are inclusive
// three-digit prefixes (cep / 100000).
type UFRange struct {
	UF   string `json:"uf"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

// CEPRanges returns the CEP prefix→UF allocation table in scan order (the order
// the validator uses; first match wins). Disjoint blocks for a UF appear more
// than once, matching the live table.
func CEPRanges() []UFRange {
	src := selo.CEPRanges()
	out := make([]UFRange, 0, len(src))
	for _, r := range src {
		out = append(out, UFRange{UF: r.UF.String(), From: r.From, To: r.To})
	}
	return out
}

// DDDtoUF returns the DDD area-code→UF map (DDD as int, UF as two-letter code).
func DDDtoUF() map[string]selo.UF {
	src := selo.DDDtoUF()
	out := make(map[string]selo.UF, len(src))
	for ddd, uf := range src {
		out[ddString(ddd)] = uf
	}
	return out
}

// DDDList returns the valid DDD codes in ascending order, for emitters that
// prefer a sorted slice over a map.
func DDDList() []int {
	src := selo.DDDtoUF()
	out := make([]int, 0, len(src))
	for ddd := range src {
		out = append(out, ddd)
	}
	sort.Ints(out)
	return out
}

// CPFRegions returns the CPF ninth-digit→region map (digit 0..9 → region text).
func CPFRegions() map[int]string {
	return selo.CPFRegions()
}

// VoterUFNames returns the voter-ID UF code (1..28)→region name map.
func VoterUFNames() map[int]string {
	return selo.VoterUFNames()
}

// ddString renders a DDD as a zero-padded two-digit string (e.g. 11 → "11").
func ddString(ddd int) string {
	return string([]byte{byte('0' + ddd/10), byte('0' + ddd%10)})
}
