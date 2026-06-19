package selo

// This file exposes read-only, serializable copies of the internal data tables
// used by the document algorithms (CEP prefix ranges, DDD→UF map, CPF region
// map). They exist so out-of-package consumers — notably internal/codegen, which
// renders these tables as constants in other languages — can read the exact data
// the validators use without depending on unexported state.
//
// These accessors are ADDITIVE public API. They do not change any validation or
// generation algorithm; each returns a fresh copy so callers cannot mutate the
// underlying tables.

import "maps"

// CEPRange is one inclusive CEP prefix range and the UF it maps to. The bounds
// are three-digit prefixes (cep / 100000); some UFs have multiple disjoint
// blocks, all of which are returned by CEPRanges.
type CEPRange struct {
	UF   UF  `json:"uf"`
	From int `json:"from"`
	To   int `json:"to"`
}

// CEPRanges returns a copy of the CEP prefix→UF allocation table, in the same
// order the CEP validator scans it (first match wins). It includes every
// disjoint block (e.g. AM and DF appear more than once).
func CEPRanges() []CEPRange {
	out := make([]CEPRange, 0, len(cepPrefixRanges))
	for _, r := range cepPrefixRanges {
		out = append(out, CEPRange{UF: r.uf, From: r.from, To: r.to})
	}

	return out
}

// DDDtoUF returns a copy of the DDD area-code→UF map used by the phone
// validator and origin resolver.
func DDDtoUF() map[int]UF {
	out := make(map[int]UF, len(dddUFTable))
	maps.Copy(out, dddUFTable)

	return out
}

// CPFRegions returns a copy of the CPF ninth-digit→region map. The key is the
// ninth digit (0–9) of a CPF; the value is the human-readable region string the
// CPF origin resolver reports.
func CPFRegions() map[int]string {
	return map[int]string{
		0: IsDigit0,
		1: IsDigit1,
		2: IsDigit2,
		3: IsDigit3,
		4: IsDigit4,
		5: IsDigit5,
		6: IsDigit6,
		7: IsDigit7,
		8: IsDigit8,
		9: IsDigit9,
	}
}

// VoterUFNames returns a copy of the TSE voter-ID UF code (01–28)→region name
// map used by the voter-ID origin resolver.
func VoterUFNames() map[int]string {
	out := make(map[int]string, len(voterUFNames))
	maps.Copy(out, voterUFNames)

	return out
}
