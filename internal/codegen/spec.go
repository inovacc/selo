// Package codegen holds the declarative, data-only description of every selo
// document kind plus the machinery that turns the verified Go library into
// golden test vectors and (in later milestones) idiomatic code for other
// languages. It is internal: not part of the public selo API.
//
// spec.go is the single source of the per-kind plan. Every value here is
// TRANSCRIBED from the corresponding selo Go file (cpf.go, cnpj.go, …); none is
// invented. The plan is declarative so per-language emitters can render the same
// algorithm without re-deriving it. Kinds whose algorithm does not fit the
// declarative CheckDigit model (CNH's coupled offset DV, the voter-ID dual DV)
// carry a Note and are reproduced faithfully by the vector emitter and by
// bespoke per-language template fragments rather than by the generic reducer.
package codegen

import "github.com/inovacc/selo"

// DVRule identifies how a weighted mod-11 sum is folded into a check digit.
// The variants cover every reduction selo uses across group A/B/C kinds.
type DVRule int

const (
	// DVElevenMinus is dv = 11 - (sum % 11), with encodings for 10 and 11.
	// Used by RG (10→'X', 11→'0').
	DVElevenMinus DVRule = iota
	// DVModRemainder folds via the remainder of the weighted sum mod 11 and maps
	// the low/overflow remainders to 0. The exact mapping is kind-specific and is
	// pinned by RemainderTo0 (see CheckDigit); examples: CPF/RENAVAM use
	// rest=(sum*10)%11 with 10/11→0; CNPJ uses 11-(sum%11) with 0/1→0; PIS uses
	// 11-(sum%11) with mod<=1→0.
	DVModRemainder
	// DVRightmostDigit is dv = (sum % 11) % 10 (a remainder of 10 yields 0).
	// Used by IE (São Paulo).
	DVRightmostDigit
	// DVSumZero is a verify-only rule: the kind is valid when the full weighted
	// sum is ≡ 0 (mod 11). Used by CNS. Weights cover every position (no separate
	// check-digit position).
	DVSumZero
)

// String returns a stable identifier for the rule (used in tests and emitters).
func (r DVRule) String() string {
	switch r {
	case DVElevenMinus:
		return "eleven_minus"
	case DVModRemainder:
		return "mod_remainder"
	case DVRightmostDigit:
		return "rightmost_digit"
	case DVSumZero:
		return "sum_zero"
	default:
		return "unknown"
	}
}

// CheckDigit declares one check digit (or, for DVSumZero, one verify pass).
//
// Weights are listed in INPUT (left-to-right) order, aligned with the digit
// positions the rule consumes — exactly as the source selo code applies them.
// For CNPJ the weights cycle right-to-left (2..9 repeating); that special
// cycling is flagged by CyclingRightToLeft so an emitter knows to apply the
// listed weights from the right and repeat them, rather than positionally.
type CheckDigit struct {
	// Weights are the positional mod-11 weights, in input order. For
	// CyclingRightToLeft kinds (CNPJ) the single cycle (2..9) is listed and
	// applied from the rightmost base character, repeating.
	Weights []int `json:"weights"`
	// Rule is how the weighted sum becomes the check digit / verification result.
	Rule DVRule `json:"rule"`
	// RemainderTo0 lists the remainder values (of the rule's fold) that collapse
	// to a 0 check digit. CPF/RENAVAM: {10,11}; CNPJ: {0,1}; PIS: {0,1}. Empty
	// for rules that need no such mapping (DVRightmostDigit, DVElevenMinus,
	// DVSumZero).
	RemainderTo0 []int `json:"remainderTo0,omitempty"`
	// MultiplyBy10 reports whether the fold is (sum*10)%11 (CPF, RENAVAM) rather
	// than sum%11 (PIS, CNPJ). Only meaningful for DVModRemainder.
	MultiplyBy10 bool `json:"multiplyBy10,omitempty"`
	// EncodeXAt is the computed value that encodes as the character 'X' (RG: 10).
	// 0 means no X encoding.
	EncodeXAt int `json:"encodeXAt,omitempty"`
	// EncodeZeroAt is the computed value that encodes as the character '0'
	// (RG: 11). 0 means no special zero encoding.
	EncodeZeroAt int `json:"encodeZeroAt,omitempty"`
}

// OriginKind classifies how a kind resolves geographic origin (if at all).
type OriginKind int

const (
	// OriginNone means the kind has no origin resolution.
	OriginNone OriginKind = iota
	// OriginCPFRegion derives a region from the CPF ninth digit.
	OriginCPFRegion
	// OriginCEPRange derives a UF from the CEP prefix range table.
	OriginCEPRange
	// OriginDDD derives a UF from the phone DDD table.
	OriginDDD
	// OriginVoterUF derives a region from the voter-ID embedded UF code.
	OriginVoterUF
)

// KindPlan is the full declarative plan for one document kind. It maps the kind
// to its taxonomy group (design spec §3), the lengths/mask/checks the validator
// and formatter use, and the irregular-kind notes that keep the generated code
// honest.
type KindPlan struct {
	// Kind is the selo registry identifier.
	Kind selo.Kind `json:"kind"`
	// Group is the taxonomy group letter (A–F) from the design spec §3.
	Group string `json:"group"`
	// Lengths are the accepted cleaned lengths (digits, or characters for CNPJ).
	// Empty when the kind is purely pattern-based (group D) — use Pattern.
	Lengths []int `json:"lengths,omitempty"`
	// Checks are the check-digit passes, in computation order. Empty for kinds
	// with no check digit (groups D, E, F).
	Checks []CheckDigit `json:"checks,omitempty"`
	// CharMap reports whether the kind uses the alphanumeric char→value map
	// (CNPJ: '0'-'9'→0-9, 'A'-'Z'→17-42).
	CharMap bool `json:"charMap,omitempty"`
	// AllEqualReject reports whether an all-identical-character input is rejected.
	AllEqualReject bool `json:"allEqualReject,omitempty"`
	// Mask is the canonical format mask using '#'/'X' placeholders for variable
	// positions and literal separators. Empty when the kind has no separator mask
	// (CNH, RENAVAM, CNS are identity; group D/E use Pattern/composition).
	Mask string `json:"mask,omitempty"`
	// Origin classifies origin resolution.
	Origin OriginKind `json:"origin,omitempty"`
	// UFScoped reports whether validation takes a UF parameter (RG, IE).
	UFScoped bool `json:"ufScoped,omitempty"`
	// Pattern is a regular expression (or a "|"-joined set) describing accepted
	// forms for pattern/composite kinds (plate, pix). Empty otherwise.
	Pattern string `json:"pattern,omitempty"`
	// Note records anything the declarative model cannot capture, so emitters and
	// reviewers know a kind needs a bespoke fragment rather than the generic
	// reducer.
	Note string `json:"note,omitempty"`
}

// Plans is the registry of per-kind plans, keyed by selo.Kind. It covers all 13
// kinds returned by selo.Kinds(). Values are transcribed from the selo source.
var Plans = map[selo.Kind]KindPlan{
	// --- Group A: numeric mod-11 check-digit kinds ---------------------------
	selo.KindCPF: {
		Kind:           selo.KindCPF,
		Group:          "A",
		Lengths:        []int{11},
		AllEqualReject: true,
		Mask:           "###.###.###-##",
		Origin:         OriginCPFRegion,
		Checks: []CheckDigit{
			// DV1: weights 10..2 over digits 0..8; rest=(sum*10)%11, 10/11→0.
			{Weights: []int{10, 9, 8, 7, 6, 5, 4, 3, 2}, Rule: DVModRemainder, MultiplyBy10: true, RemainderTo0: []int{10, 11}},
			// DV2: weights 11..2 over digits 0..9; rest=(sum*10)%11, 10/11→0.
			{Weights: []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2}, Rule: DVModRemainder, MultiplyBy10: true, RemainderTo0: []int{10, 11}},
		},
	},
	selo.KindPIS: {
		Kind:           selo.KindPIS,
		Group:          "A",
		Lengths:        []int{11},
		AllEqualReject: true,
		Mask:           "###.#####.##-#",
		Checks: []CheckDigit{
			// Single DV: weights {3,2,9,8,7,6,5,4,3,2} over digits 0..9;
			// mod=sum%11; mod<=1→0 else 11-mod.
			{Weights: []int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}, Rule: DVModRemainder, RemainderTo0: []int{0, 1}},
		},
	},
	selo.KindRenavam: {
		Kind:           selo.KindRenavam,
		Group:          "A",
		Lengths:        []int{11},
		AllEqualReject: true,
		// RENAVAM has no separator mask; Format left-pads to 11 digits.
		Mask: "",
		Checks: []CheckDigit{
			// Single DV: weights {3,2,9,8,7,6,5,4,3,2} over digits 0..9;
			// dv=(sum*10)%11; 10→0.
			{Weights: []int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}, Rule: DVModRemainder, MultiplyBy10: true, RemainderTo0: []int{10}},
		},
		Note: "Format left-pads inputs shorter than 11 digits with zeros; no separator mask.",
	},
	selo.KindCNH: {
		Kind:           selo.KindCNH,
		Group:          "A",
		Lengths:        []int{11},
		AllEqualReject: true,
		Mask:           "",
		// CNH's two DVs are COUPLED: DV1 carries a -2 offset (dsc) into DV2 when
		// DV1's raw remainder >= 10. This does not fit the independent-CheckDigit
		// model, so the weights are recorded for reference but the validator must
		// reproduce the coupled algorithm (see Note); the vector emitter and a
		// bespoke per-language fragment handle it.
		Checks: []CheckDigit{
			{Weights: []int{9, 8, 7, 6, 5, 4, 3, 2, 1}, Rule: DVModRemainder, RemainderTo0: []int{10}},
			{Weights: []int{1, 2, 3, 4, 5, 6, 7, 8, 9}, Rule: DVModRemainder, RemainderTo0: []int{10}},
		},
		Note: "COUPLED DVs: DV1 uses descending weights 9..1 (raw remainder >=10 → DV1=0 and a -2 offset is carried); DV2 uses ascending weights 1..9 with that offset subtracted before the mod-11 fold (negative wraps +11; result >=10 → 0). Not expressible as two independent CheckDigit passes — emitters need a bespoke CNH fragment.",
	},
	selo.KindRG: {
		Kind:     selo.KindRG,
		Group:    "A",
		Lengths:  []int{9}, // 8 base digits + 1 check char
		Mask:     "XX.XXX.XXX-X",
		UFScoped: true,
		Checks: []CheckDigit{
			// Single DV: weights {2,3,4,5,6,7,8,9} over the 8 base digits;
			// dv=11-(sum%11); 10→'X', 11→'0'.
			{Weights: []int{2, 3, 4, 5, 6, 7, 8, 9}, Rule: DVElevenMinus, EncodeXAt: 10, EncodeZeroAt: 11},
		},
		Note: "Implemented UFs: SP, RJ (shared algorithm). Check char encodes 10→'X', 11→'0'.",
	},
	selo.KindIE: {
		Kind:     selo.KindIE,
		Group:    "A",
		Lengths:  []int{12}, // São Paulo
		Mask:     "###.###.###.###",
		UFScoped: true,
		Checks: []CheckDigit{
			// DV1 (position 9): weights {1,3,4,5,6,7,8,10} over digits 0..7;
			// rightmost digit of (sum%11).
			{Weights: []int{1, 3, 4, 5, 6, 7, 8, 10}, Rule: DVRightmostDigit},
			// DV2 (position 12): weights {3,2,10,9,8,7,6,5,4,3,2} over digits 0..10;
			// rightmost digit of (sum%11).
			{Weights: []int{3, 2, 10, 9, 8, 7, 6, 5, 4, 3, 2}, Rule: DVRightmostDigit},
		},
		Note: "Implemented UF: SP only (12 digits). Each DV is the rightmost digit of (weighted sum mod 11). Other UFs return ErrUFNotImplemented.",
	},
	// --- Group B: mod-11 verify (sum ≡ 0) -----------------------------------
	selo.KindCNS: {
		Kind:           selo.KindCNS,
		Group:          "B",
		Lengths:        []int{15},
		AllEqualReject: true,
		Mask:           "",
		Checks: []CheckDigit{
			// Verify-only: descending weights 15..1 over all 15 digits; valid when
			// sum%11 == 0.
			{Weights: []int{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}, Rule: DVSumZero},
		},
		Note: "Leading digit must be one of 1,2 (definitive) or 7,8,9 (provisional). No separator mask.",
	},
	// --- Group C: alphanumeric mod-11 ---------------------------------------
	selo.KindCNPJ: {
		Kind:           selo.KindCNPJ,
		Group:          "C",
		Lengths:        []int{14},
		CharMap:        true,
		AllEqualReject: true,
		Mask:           "XX.XXX.XXX/XXXX-XX",
		Checks: []CheckDigit{
			// DV1 over the 12 base chars; DV2 over base+DV1. Weights {2..9} cycle
			// right-to-left, repeating. remainder=sum%11; 0/1→0 else 11-remainder.
			{Weights: []int{2, 3, 4, 5, 6, 7, 8, 9}, Rule: DVModRemainder, RemainderTo0: []int{0, 1}},
			{Weights: []int{2, 3, 4, 5, 6, 7, 8, 9}, Rule: DVModRemainder, RemainderTo0: []int{0, 1}},
		},
		Note: "Alphanumeric: char→value map ('0'-'9'→0-9, 'A'-'Z'→17-42). Weights cycle right-to-left (applied from the rightmost base char, repeating 2..9). Last 2 characters must be numeric.",
	},
	// --- Group D: pattern / regex -------------------------------------------
	selo.KindPlate: {
		Kind:    selo.KindPlate,
		Group:   "D",
		Pattern: `^[A-Z]{3}-?[0-9]{4}$|^[A-Z]{3}[0-9][A-Z][0-9]{2}$`,
		Note:    "National pattern ^[A-Z]{3}-?[0-9]{4}$ (formats to ABC-1234); Mercosul pattern ^[A-Z]{3}[0-9][A-Z][0-9]{2}$ (formats to ABC1D23). Matching is case-insensitive (input is upper-cased and trimmed).",
	},
	// --- Group E: composite -------------------------------------------------
	selo.KindPIX: {
		Kind:  selo.KindPIX,
		Group: "E",
		// Composite: dispatch by shape. EVP/email/phone are regex; CPF/CNPJ reuse
		// groups A/C by digit length.
		Pattern: `evp=^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$;phone=^\+55\d{10,11}$;email=^[A-Za-z0-9._%+\-]+@[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?)+$`,
		Note:    "Composite key: detection order is EVP (UUIDv4) → email (contains '@') → phone (+55, 10-11 digits) → CPF (11 digits, reuse CPF validate) → CNPJ (14 chars, reuse CNPJ validate). Format returns the trimmed key verbatim.",
	},
	// --- Group F: table lookup (validation + origin) ------------------------
	selo.KindCEP: {
		Kind:    selo.KindCEP,
		Group:   "F",
		Lengths: []int{8},
		Mask:    "#####-###",
		Origin:  OriginCEPRange,
		Note:    "Validation: the 3-digit prefix (cep/100000) must fall in a CEP→UF range (see codegen.CEPRanges / selo.CEPRanges).",
	},
	selo.KindPhone: {
		Kind:    selo.KindPhone,
		Group:   "F",
		Lengths: []int{10, 11}, // landline / mobile national digits
		Mask:    "(##) #####-####",
		Origin:  OriginDDD,
		Note:    "Accepts an optional +55/0055 country prefix. DDD (first 2 digits) must be in the DDD→UF table. 11-digit (mobile) numbers must have a '9' after the DDD. Landline mask is (##) ####-####; mobile mask is (##) #####-####.",
	},
	selo.KindVoterID: {
		Kind:           selo.KindVoterID,
		Group:          "F",
		Lengths:        []int{12},
		AllEqualReject: true,
		Mask:           "#### #### ####",
		Origin:         OriginVoterUF,
		Checks: []CheckDigit{
			// DV1 over the 8 sequence digits: weights {2..9}; mod=sum%11;
			// 10/11→0.
			{Weights: []int{2, 3, 4, 5, 6, 7, 8, 9}, Rule: DVModRemainder, RemainderTo0: []int{10, 11}},
			// DV2 over [ufDigit0, ufDigit1, dv1]: weights {7,8,9}; mod=sum%11;
			// 10/11→0.
			{Weights: []int{7, 8, 9}, Rule: DVModRemainder, RemainderTo0: []int{10, 11}},
		},
		Note: "Layout SSSSSSSS UU D1 D2. The 2-digit UF code (positions 9-10) must be 01..28. DV2's inputs are the two UF digits plus DV1 (not the sequence digits) — emitters need a bespoke voter fragment for this dependency.",
	},
}

// PlanFor returns the plan for kind and ok=false when no plan is registered.
func PlanFor(k selo.Kind) (KindPlan, bool) {
	p, ok := Plans[k]
	return p, ok
}
