package selo

import (
	"fmt"
	"math/rand/v2"
	"slices"
)

// IE is the Inscrição Estadual (state tax registration) document type. IE has
// no national standard — each federative unit defines its own length(s),
// weights, and check-digit rule(s) — so IE is UF-scoped like RG. This is a
// first-batch implementation: only the UFs in ieTable are supported; every
// other UF returns ErrUFNotImplemented. See docs/IE-NOTES.md for the per-UF
// research, sample provenance, and the remaining-UF roadmap.
type IE struct{}

// NewIE returns a stateless IE document.
func NewIE() *IE { return &IE{} }

// Kind reports the document kind (KindIE).
func (e *IE) Kind() Kind { return KindIE }

func init() { Register(&IE{}) }

// ieAlgo describes one federative unit's IE rules. The table is shaped so a UF
// can grow additional accepted lengths/formats without changing this struct.
type ieAlgo struct {
	// lengths are the accepted cleaned-digit lengths for this UF.
	lengths []int
	// validate reports whether d (a cleaned digit string of an accepted length)
	// is a valid IE for this UF.
	validate func(d string) bool
	// generate returns a freshly generated valid IE in masked form, or is nil
	// when constructive generation is not implemented for this UF.
	generate func() string
	// generateRand returns a freshly generated valid IE using the supplied random source,
	// or is nil when seeded generation is not implemented for this UF.
	generateRand func(r *rand.Rand) string
	// mask renders a cleaned digit string in the UF's canonical masked form, or
	// returns it unchanged when no mask is defined.
	mask func(d string) string
}

// ieTable holds the implemented per-UF algorithms. Only verified UFs (an
// authoritative algorithm plus >=2 sourced real samples) are listed here.
var ieTable = map[UF]ieAlgo{
	UFSP: {
		lengths:      []int{ieSPLength},
		validate:     ieSPValidate,
		generate:     ieSPGenerate,
		generateRand: ieSPGenerateRand,
		mask:         ieSPMask,
	},
	UFMG: {
		lengths:      []int{ieMGLength},
		validate:     ieMGValidate,
		generate:     ieMGGenerate,
		generateRand: ieMGGenerateRand,
		mask:         ieMGMask,
	},
	UFRS: {
		lengths:      []int{ieRSLength},
		validate:     ieRSValidate,
		generate:     ieRSGenerate,
		generateRand: ieRSGenerateRand,
		mask:         ieRSMask,
	},
	UFPR: {
		lengths:      []int{iePRLength},
		validate:     iePRValidate,
		generate:     iePRGenerate,
		generateRand: iePRGenerateRand,
		mask:         iePRMask,
	},
}

// ImplementedUFs returns, sorted, the federative units with IE support.
func (e *IE) ImplementedUFs() []UF {
	ufs := make([]UF, 0, len(ieTable))
	for uf := range ieTable {
		ufs = append(ufs, uf)
	}

	slices.Sort(ufs)

	return ufs
}

// ValidateUF reports whether value is a valid IE for the given federative unit.
// An unsupported UF yields (false, ErrUFNotImplemented); a value whose cleaned
// length is not accepted for a supported UF yields (false, ErrInvalidFormat); a
// correctly-shaped value with a bad check digit yields (false, nil).
func (e *IE) ValidateUF(value string, uf UF) (bool, error) {
	algo, ok := ieTable[uf]
	if !ok {
		return false, fmt.Errorf("%w: %s", ErrUFNotImplemented, uf)
	}

	d := onlyDigits(value)
	if !slices.Contains(algo.lengths, len(d)) {
		return false, ErrInvalidFormat
	}

	return algo.validate(d), nil
}

// Validate reports whether value is a valid IE under any implemented federative
// unit. It satisfies the Document interface (first match wins).
func (e *IE) Validate(value string) bool {
	for _, uf := range e.ImplementedUFs() {
		if ok, err := e.ValidateUF(value, uf); err == nil && ok {
			return true
		}
	}

	return false
}

// Format renders value in the canonical mask of the first implemented UF under
// which it validates. A value that validates under no implemented UF yields
// ErrInvalidFormat. (IE masks are UF-specific; formatting requires knowing the
// UF, which is inferred here from which algorithm accepts the value.)
func (e *IE) Format(value string) (string, error) {
	d := onlyDigits(value)

	for _, uf := range e.ImplementedUFs() {
		algo := ieTable[uf]
		if slices.Contains(algo.lengths, len(d)) && algo.validate(d) {
			return algo.mask(d), nil
		}
	}

	return "", ErrInvalidFormat
}

// GenerateRand returns a valid IE in masked form using the supplied random source,
// for a randomly chosen implemented UF that supports seeded generation.
func (e *IE) GenerateRand(r *rand.Rand) string {
	var gen []UF

	for _, uf := range e.ImplementedUFs() {
		if ieTable[uf].generateRand != nil {
			gen = append(gen, uf)
		}
	}

	if len(gen) == 0 {
		return ""
	}

	return ieTable[gen[r.IntN(len(gen))]].generateRand(r)
}

// Generate returns a freshly generated valid IE in masked form, for a randomly
// chosen implemented UF that supports constructive generation. It returns ""
// only if no implemented UF supports generation (not the case while SP ships).
func (e *IE) Generate() string { return e.GenerateRand(newRand()) }

// ieRightmostDV computes a São Paulo-style check digit: the rightmost (units)
// digit of (weighted sum mod 11). A remainder of 10 therefore yields 0. weights
// must not be longer than d.
func ieRightmostDV(d string, weights []int) int {
	sum := 0
	for i, w := range weights {
		sum += int(d[i]-'0') * w
	}

	return (sum % 11) % 10
}

// --- São Paulo (SP) -----------------------------------------------------------
// 12 digits, format AAA.AAA.AAA.AAA. The 9th digit is the first check digit and
// the 12th is the second, each the rightmost digit of (weighted sum mod 11).
// Source: SEFAZ-SP / Sintegra rotina de consistência (see docs/IE-NOTES.md).

const ieSPLength = 12

// ieSPWeights1 weights digits 1..8 for the first check digit (position 9).
var ieSPWeights1 = []int{1, 3, 4, 5, 6, 7, 8, 10}

// ieSPWeights2 weights digits 1..11 for the second check digit (position 12).
var ieSPWeights2 = []int{3, 2, 10, 9, 8, 7, 6, 5, 4, 3, 2}

func ieSPValidate(d string) bool {
	if len(d) != ieSPLength {
		return false
	}

	if ieRightmostDV(d, ieSPWeights1) != int(d[8]-'0') {
		return false
	}

	return ieRightmostDV(d, ieSPWeights2) == int(d[11]-'0')
}

func ieSPMask(d string) string {
	if len(d) != ieSPLength {
		return d
	}

	return d[0:3] + "." + d[3:6] + "." + d[6:9] + "." + d[9:12]
}

func ieSPGenerate() string {
	var d [ieSPLength]byte
	for i := range 8 {
		d[i] = byte('0' + rand.IntN(10))
	}

	d[8] = byte('0' + ieRightmostDV(string(d[:8]), ieSPWeights1))
	d[9] = byte('0' + rand.IntN(10))
	d[10] = byte('0' + rand.IntN(10))
	d[11] = byte('0' + ieRightmostDV(string(d[:11]), ieSPWeights2))

	return ieSPMask(string(d[:]))
}

func ieSPGenerateRand(r *rand.Rand) string {
	var d [ieSPLength]byte
	for i := range 8 {
		d[i] = byte('0' + r.IntN(10))
	}

	d[8] = byte('0' + ieRightmostDV(string(d[:8]), ieSPWeights1))
	d[9] = byte('0' + r.IntN(10))
	d[10] = byte('0' + r.IntN(10))
	d[11] = byte('0' + ieRightmostDV(string(d[:11]), ieSPWeights2))

	return ieSPMask(string(d[:]))
}

// ieWeightedSum returns the sum of each digit of d multiplied by the weight at
// the same index. weights must not be longer than d.
func ieWeightedSum(d string, weights []int) int {
	sum := 0
	for i, w := range weights {
		sum += int(d[i]-'0') * w
	}

	return sum
}

// ieMod11DV computes a "11 minus remainder" check digit from a weighted sum:
// DV = 11 - (sum mod 11), with a result of 10 or 11 collapsing to 0. This is the
// most common IE check-digit rule (RS, both PR digits, and MG's second digit).
func ieMod11DV(sum int) int {
	dv := 11 - (sum % 11)
	if dv >= 10 {
		return 0
	}

	return dv
}

// --- Minas Gerais (MG) --------------------------------------------------------
// 13 digits, format AAA.AAA.AAA/AAAA (3 municipio + 6 inscricao + 2 ordem + 2
// DV). D1 uses the "digit-sum" method: a 0 is inserted after the 3-digit
// municipio code, the resulting 12 digits are multiplied left->right by
// alternating weights 1,2,..., and the DIGITS of each product are summed; D1 is
// the amount needed to reach the next multiple of ten. D2 is a mod-11 digit over
// the 11 base digits plus D1. Source: SINTEGRA-MG roteiro de crítica, corroborated
// by two independent reference implementations (see docs/IE-NOTES.md).

const ieMGLength = 13

var (
	// ieMGWeights1 are the alternating digit-sum weights for D1 (over 12 digits).
	ieMGWeights1 = []int{1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2}
	// ieMGWeights2 are the mod-11 weights for D2, left->right over 12 digits.
	ieMGWeights2 = []int{3, 2, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2}
)

// ieMGDigits computes MG's two check digits from the 11 base digits.
func ieMGDigits(base11 string) (d1, d2 int) {
	base12 := base11[0:3] + "0" + base11[3:11]

	total := 0

	for i, w := range ieMGWeights1 {
		p := int(base12[i]-'0') * w
		total += p/10 + p%10
	}

	d1 = (10 - total%10) % 10

	base := base11 + string(byte('0'+d1))
	d2 = ieMod11DV(ieWeightedSum(base, ieMGWeights2))

	return d1, d2
}

func ieMGValidate(d string) bool {
	if len(d) != ieMGLength {
		return false
	}

	d1, d2 := ieMGDigits(d[:11])

	return d1 == int(d[11]-'0') && d2 == int(d[12]-'0')
}

func ieMGMask(d string) string {
	if len(d) != ieMGLength {
		return d
	}

	return d[0:3] + "." + d[3:6] + "." + d[6:9] + "/" + d[9:13]
}

func ieMGGenerateRand(r *rand.Rand) string {
	var d [ieMGLength]byte
	for i := range 11 {
		d[i] = byte('0' + r.IntN(10))
	}

	d1, d2 := ieMGDigits(string(d[:11]))
	d[11] = byte('0' + d1)
	d[12] = byte('0' + d2)

	return ieMGMask(string(d[:]))
}

func ieMGGenerate() string { return ieMGGenerateRand(newRand()) }

// --- Rio Grande do Sul (RS) ---------------------------------------------------
// 10 digits, format AAA/AAAAAAA (3 municipio + 6 empresa + 1 DV). The DV is a
// mod-11 digit (weights 2,9,8,...,2 left->right over the first 9 digits).
// Source: SINTEGRA-RS roteiro de crítica, corroborated by two independent
// reference implementations (see docs/IE-NOTES.md).

const ieRSLength = 10

// ieRSWeights are the mod-11 weights for RS's single DV, over the first 9 digits.
var ieRSWeights = []int{2, 9, 8, 7, 6, 5, 4, 3, 2}

func ieRSValidate(d string) bool {
	if len(d) != ieRSLength {
		return false
	}

	return ieMod11DV(ieWeightedSum(d[:9], ieRSWeights)) == int(d[9]-'0')
}

func ieRSMask(d string) string {
	if len(d) != ieRSLength {
		return d
	}

	return d[0:3] + "/" + d[3:10]
}

func ieRSGenerateRand(r *rand.Rand) string {
	var d [ieRSLength]byte
	for i := range 9 {
		d[i] = byte('0' + r.IntN(10))
	}

	d[9] = byte('0' + ieMod11DV(ieWeightedSum(string(d[:9]), ieRSWeights)))

	return ieRSMask(string(d[:]))
}

func ieRSGenerate() string { return ieRSGenerateRand(newRand()) }

// --- Paraná (PR) --------------------------------------------------------------
// 10 digits, format AAA.AAAAA-AA (8 base + 2 DV). Both DVs are mod-11: DV1 over
// the 8 base digits (weights 3,2,7,6,5,4,3,2); DV2 over the 8 base digits
// (weights 4,3,2,7,6,5,4,3) plus twice DV1 added to the sum. Source: SEFA-PR
// digit-verifier reference routine + worked example (see docs/IE-NOTES.md).

const iePRLength = 10

var (
	// iePRWeights1 are the mod-11 weights for PR's first DV, over 8 base digits.
	iePRWeights1 = []int{3, 2, 7, 6, 5, 4, 3, 2}
	// iePRWeights2 are the mod-11 weights for PR's second DV, over 8 base digits.
	iePRWeights2 = []int{4, 3, 2, 7, 6, 5, 4, 3}
)

// iePRDigits computes PR's two check digits from the 8 base digits.
func iePRDigits(base8 string) (d1, d2 int) {
	d1 = ieMod11DV(ieWeightedSum(base8, iePRWeights1))
	d2 = ieMod11DV(ieWeightedSum(base8, iePRWeights2) + 2*d1)

	return d1, d2
}

func iePRValidate(d string) bool {
	if len(d) != iePRLength {
		return false
	}

	d1, d2 := iePRDigits(d[:8])

	return d1 == int(d[8]-'0') && d2 == int(d[9]-'0')
}

func iePRMask(d string) string {
	if len(d) != iePRLength {
		return d
	}

	return d[0:3] + "." + d[3:8] + "-" + d[8:10]
}

func iePRGenerateRand(r *rand.Rand) string {
	var d [iePRLength]byte
	for i := range 8 {
		d[i] = byte('0' + r.IntN(10))
	}

	d1, d2 := iePRDigits(string(d[:8]))
	d[8] = byte('0' + d1)
	d[9] = byte('0' + d2)

	return iePRMask(string(d[:]))
}

func iePRGenerate() string { return iePRGenerateRand(newRand()) }
