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
	// mask renders a cleaned digit string in the UF's canonical masked form, or
	// returns it unchanged when no mask is defined.
	mask func(d string) string
}

// ieTable holds the implemented per-UF algorithms. Only verified UFs (an
// authoritative algorithm plus >=2 sourced real samples) are listed here.
var ieTable = map[UF]ieAlgo{
	UFSP: {
		lengths:  []int{ieSPLength},
		validate: ieSPValidate,
		generate: ieSPGenerate,
		mask:     ieSPMask,
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

// Generate returns a freshly generated valid IE in masked form, for a randomly
// chosen implemented UF that supports constructive generation. It returns ""
// only if no implemented UF supports generation (not the case while SP ships).
func (e *IE) Generate() string {
	var gen []UF

	for _, uf := range e.ImplementedUFs() {
		if ieTable[uf].generate != nil {
			gen = append(gen, uf)
		}
	}

	if len(gen) == 0 {
		return ""
	}

	return ieTable[gen[rand.IntN(len(gen))]].generate()
}

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
