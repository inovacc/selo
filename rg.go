package selo

import (
	"fmt"
	"math/rand/v2"
)

// RGBaseLength is the count of base (non-check) digits in an RG number.
const RGBaseLength = 8

// RGTotalLength is the count of significant characters in an RG number:
// 8 base digits plus 1 check character.
const RGTotalLength = 9

// rgWeights are the positional mod-11 weights applied to the 8 base digits
// of an SP/RJ RG, least-significant base digit first.
var rgWeights = [RGBaseLength]int{2, 3, 4, 5, 6, 7, 8, 9}

// RG is the Registro Geral (state identity card) document type. Only the SP
// and RJ algorithms are well-defined; other federative units return
// ErrUFNotImplemented.
type RG struct{}

// NewRG returns a stateless RG document.
func NewRG() *RG { return &RG{} }

// Kind reports the document kind (KindRG).
func (r *RG) Kind() Kind { return KindRG }

func init() { Register(&RG{}) }

// parse strips RG formatting and returns the 8 base digits, the represented
// check value, and ok=true when the input has exactly 8 base digits plus one
// valid check character. The check character is 'X'/'x' (=> 10), a digit, or
// '0' (=> 11). base is filled index 0..7 in input order.
func (r *RG) parse(value string) (base [RGBaseLength]int, check int, ok bool) {
	cleaned := r.clean(value)
	if len(cleaned) != RGTotalLength {
		return base, 0, false
	}

	last := cleaned[RGBaseLength] // the check character
	switch {
	case last == 'X' || last == 'x':
		check = 10
	case last == '0':
		check = 11
	case last >= '1' && last <= '9':
		check = int(last - '0')
	default:
		return base, 0, false
	}

	for i := range RGBaseLength {
		c := cleaned[i]
		if c < '0' || c > '9' {
			return base, 0, false
		}

		base[i] = int(c - '0')
	}

	return base, check, true
}

// clean strips dots and dashes (and any other non-alphanumeric punctuation)
// from an RG, preserving digits and a trailing X/x check character.
func (r *RG) clean(value string) string {
	out := make([]byte, 0, RGTotalLength)

	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= '0' && c <= '9') || c == 'X' || c == 'x' {
			out = append(out, c)
		}
	}

	return string(out)
}

// checkDigit computes the mod-11 check value for the 8 base digits using the
// SP/RJ positional weights 2..9 and the DV = 11 - (sum mod 11) convention. The
// result is in 1..11, where 10 is encoded as the check char 'X' and 11 as '0'
// (matching parse and Format); digits 1..9 encode as themselves.
func (r *RG) checkDigit(base [RGBaseLength]int) int {
	sum := 0
	for i := range RGBaseLength {
		sum += base[i] * rgWeights[i]
	}

	return 11 - (sum % 11)
}

// rgImplemented is the set of federative units whose RG algorithm is shipped.
var rgImplemented = map[UF]bool{UFSP: true, UFRJ: true}

// ImplementedUFs returns the federative units for which RG validation is
// supported (SP and RJ).
func (r *RG) ImplementedUFs() []UF { return []UF{UFSP, UFRJ} }

// ValidateUF reports whether value is a valid RG for the given federative
// unit. SP and RJ share the mod-11/weights-2..9 algorithm. Any other UF
// yields (false, ErrUFNotImplemented). A malformed value for a supported UF
// yields (false, ErrInvalidFormat).
func (r *RG) ValidateUF(value string, uf UF) (bool, error) {
	if !rgImplemented[uf] {
		return false, fmt.Errorf("%w: %s", ErrUFNotImplemented, uf)
	}

	base, check, ok := r.parse(value)
	if !ok {
		return false, ErrInvalidFormat
	}

	return r.checkDigit(base) == check, nil
}

// Validate reports whether value is a valid RG under any implemented
// federative unit (SP or RJ). It satisfies the Document interface.
func (r *RG) Validate(value string) bool {
	for _, uf := range r.ImplementedUFs() {
		ok, err := r.ValidateUF(value, uf)
		if err == nil && ok {
			return true
		}
	}

	return false
}

// Format renders an RG as XX.XXX.XXX-C. The check character is normalized:
// 'x' becomes 'X', and '0' is preserved. It returns ErrInvalidFormat when the
// value does not have 8 base digits plus a valid check character.
func (r *RG) Format(value string) (string, error) {
	base, check, ok := r.parse(value)
	if !ok {
		return "", ErrInvalidFormat
	}

	var checkChar byte

	switch check {
	case 10:
		checkChar = 'X'
	case 11:
		checkChar = '0'
	default:
		checkChar = byte('0' + check)
	}

	buf := make([]byte, 0, 12)
	for i := range RGBaseLength {
		buf = append(buf, byte('0'+base[i]))
		if i == 1 || i == 4 {
			buf = append(buf, '.')
		}
	}

	buf = append(buf, '-', checkChar)

	return string(buf), nil
}

// Generate returns a syntactically valid, SP-style RG in masked form
// XX.XXX.XXX-C. The 8 base digits are random; the check character is computed
// via the SP/RJ mod-11 algorithm ('X' when the DV is 10). It satisfies the
// Document interface.
func (r *RG) Generate() string {
	var base [RGBaseLength]int
	for i := range RGBaseLength {
		base[i] = rand.IntN(10)
	}

	dv := r.checkDigit(base)

	var checkChar byte

	switch dv {
	case 10:
		checkChar = 'X'
	case 11:
		checkChar = '0'
	default:
		checkChar = byte('0' + dv)
	}

	buf := make([]byte, 0, 12)
	for i := range RGBaseLength {
		buf = append(buf, byte('0'+base[i]))
		if i == 1 || i == 4 {
			buf = append(buf, '.')
		}
	}

	buf = append(buf, '-', checkChar)

	return string(buf)
}
