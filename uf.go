package selo

import "slices"

// The 27 Brazilian federative units (26 states + the Federal District).
const (
	UFAC UF = "AC" // Acre
	UFAL UF = "AL" // Alagoas
	UFAP UF = "AP" // Amapá
	UFAM UF = "AM" // Amazonas
	UFBA UF = "BA" // Bahia
	UFCE UF = "CE" // Ceará
	UFDF UF = "DF" // Distrito Federal
	UFES UF = "ES" // Espírito Santo
	UFGO UF = "GO" // Goiás
	UFMA UF = "MA" // Maranhão
	UFMT UF = "MT" // Mato Grosso
	UFMS UF = "MS" // Mato Grosso do Sul
	UFMG UF = "MG" // Minas Gerais
	UFPA UF = "PA" // Pará
	UFPB UF = "PB" // Paraíba
	UFPR UF = "PR" // Paraná
	UFPE UF = "PE" // Pernambuco
	UFPI UF = "PI" // Piauí
	UFRJ UF = "RJ" // Rio de Janeiro
	UFRN UF = "RN" // Rio Grande do Norte
	UFRS UF = "RS" // Rio Grande do Sul
	UFRO UF = "RO" // Rondônia
	UFRR UF = "RR" // Roraima
	UFSC UF = "SC" // Santa Catarina
	UFSP UF = "SP" // São Paulo
	UFSE UF = "SE" // Sergipe
	UFTO UF = "TO" // Tocantins
)

// allUFs is the canonical set, kept private so AllUFs can hand out copies.
var allUFs = []UF{
	UFAC, UFAL, UFAP, UFAM, UFBA, UFCE, UFDF, UFES, UFGO, UFMA,
	UFMT, UFMS, UFMG, UFPA, UFPB, UFPR, UFPE, UFPI, UFRJ, UFRN,
	UFRS, UFRO, UFRR, UFSC, UFSP, UFSE, UFTO,
}

var ufSet = func() map[UF]struct{} {
	m := make(map[UF]struct{}, len(allUFs))
	for _, u := range allUFs {
		m[u] = struct{}{}
	}

	return m
}()

// String returns the two-letter UF code.
func (u UF) String() string { return string(u) }

// Valid reports whether u is one of the 27 known federative units.
func (u UF) Valid() bool {
	_, ok := ufSet[u]
	return ok
}

// AllUFs returns a sorted, stable copy of the 27 federative units.
func AllUFs() []UF {
	out := make([]UF, len(allUFs))
	copy(out, allUFs)
	slices.Sort(out)

	return out
}

// cepRanges maps a UF to its inclusive [low, high] 5-digit CEP prefix range.
// Populated by the CEP type task (cep.go); intentionally empty here.
var cepRanges = map[UF][2]int{}

// dddToUF maps a telephone area code (DDD) to its UF.
// Populated by the phone type task (phone.go); intentionally empty here.
var dddToUF = map[int]UF{}
