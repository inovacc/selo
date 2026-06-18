package brdoc

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// dddUFTable lists each DDD area code with its federative unit (ANATEL).
var dddUFTable = map[int]UF{
	11: UFSP, 12: UFSP, 13: UFSP, 14: UFSP, 15: UFSP, 16: UFSP, 17: UFSP, 18: UFSP, 19: UFSP,
	21: UFRJ, 22: UFRJ, 24: UFRJ,
	27: UFES, 28: UFES,
	31: UFMG, 32: UFMG, 33: UFMG, 34: UFMG, 35: UFMG, 37: UFMG, 38: UFMG,
	41: UFPR, 42: UFPR, 43: UFPR, 44: UFPR, 45: UFPR, 46: UFPR,
	47: UFSC, 48: UFSC, 49: UFSC,
	51: UFRS, 53: UFRS, 54: UFRS, 55: UFRS,
	61: UFDF,
	62: UFGO, 64: UFGO,
	63: UFTO,
	65: UFMT, 66: UFMT,
	67: UFMS,
	68: UFAC,
	69: UFRO,
	71: UFBA, 73: UFBA, 74: UFBA, 75: UFBA, 77: UFBA,
	79: UFSE,
	81: UFPE, 87: UFPE,
	82: UFAL,
	83: UFPB,
	84: UFRN,
	85: UFCE, 88: UFCE,
	86: UFPI, 89: UFPI,
	91: UFPA, 93: UFPA, 94: UFPA,
	92: UFAM, 97: UFAM,
	95: UFRR,
	96: UFAP,
	98: UFMA, 99: UFMA,
}

// ddds is a stable, sorted slice of valid DDD codes (for Generate).
var ddds []int

func init() {
	// Populate the dddToUF stub declared in uf.go (M0-3).
	for ddd, uf := range dddUFTable {
		dddToUF[ddd] = uf
		ddds = append(ddds, ddd)
	}
	// Keep ddds deterministic for reproducible test debugging.
	for i := 1; i < len(ddds); i++ {
		for j := i; j > 0 && ddds[j-1] > ddds[j]; j-- {
			ddds[j-1], ddds[j] = ddds[j], ddds[j-1]
		}
	}
	Register(&Phone{})
}

// Phone validates, generates, and formats Brazilian telephone numbers and
// resolves the federative unit from the DDD area code.
type Phone struct{}

// NewPhone creates a new Phone instance.
func NewPhone() *Phone { return &Phone{} }

// Kind returns KindPhone.
func (p *Phone) Kind() Kind { return KindPhone }

// nationalNumber strips an optional +55 / 0055 / 55 country prefix and returns
// the remaining national digits. ok=false when nothing is left.
func nationalNumber(d string) (string, bool) {
	switch {
	case strings.HasPrefix(d, "0055"):
		d = d[4:]
	case strings.HasPrefix(d, "55") && len(d) > 11:
		// Only treat a leading "55" as the country code when the remainder is
		// a plausible national number (10 or 11 digits).
		d = d[2:]
	}
	if d == "" {
		return "", false
	}
	return d, true
}

// Validate reports whether value is a well-formed Brazilian phone number whose
// DDD maps to a known UF. Accepts +55/0055 prefix and any punctuation.
func (p *Phone) Validate(value string) bool {
	d, ok := nationalNumber(onlyDigits(value))
	if !ok {
		return false
	}
	// National number is DDD(2) + subscriber(8 landline | 9 mobile).
	if len(d) != 10 && len(d) != 11 {
		return false
	}
	ddd := int(d[0]-'0')*10 + int(d[1]-'0')
	if _, known := dddUFTable[ddd]; !known {
		return false
	}
	// 9-digit mobile must begin with 9.
	if len(d) == 11 && d[2] != '9' {
		return false
	}
	return true
}

// Format masks a phone number as "(DD) NNNNN-NNNN" (mobile) or
// "(DD) NNNN-NNNN" (landline). Returns ErrInvalidLength on bad length and
// ErrInvalidFormat when the DDD is unknown.
func (p *Phone) Format(value string) (string, error) {
	d, ok := nationalNumber(onlyDigits(value))
	if !ok || (len(d) != 10 && len(d) != 11) {
		return "", fmt.Errorf("brdoc: phone needs 10 or 11 national digits: %w", ErrInvalidLength)
	}
	ddd := int(d[0]-'0')*10 + int(d[1]-'0')
	if _, known := dddUFTable[ddd]; !known {
		return "", fmt.Errorf("brdoc: phone DDD %02d unknown: %w", ddd, ErrInvalidFormat)
	}
	sub := d[2:]
	if len(sub) == 9 {
		return "(" + d[0:2] + ") " + sub[0:5] + "-" + sub[5:9], nil
	}
	return "(" + d[0:2] + ") " + sub[0:4] + "-" + sub[4:8], nil
}

// Origin returns the federative unit for the phone's DDD. Returns
// ErrInvalidLength on bad length and ErrInvalidFormat for an unknown DDD.
// Phone satisfies OriginResolver.
func (p *Phone) Origin(value string) (string, error) {
	d, ok := nationalNumber(onlyDigits(value))
	if !ok || (len(d) != 10 && len(d) != 11) {
		return "", fmt.Errorf("brdoc: phone needs 10 or 11 national digits: %w", ErrInvalidLength)
	}
	ddd := int(d[0]-'0')*10 + int(d[1]-'0')
	uf, known := dddUFTable[ddd]
	if !known {
		return "", fmt.Errorf("brdoc: phone DDD %02d unknown: %w", ddd, ErrInvalidFormat)
	}
	return uf.String(), nil
}

// Generate returns a random valid Brazilian phone number (unformatted national
// digits). It picks a real DDD and randomly emits a 9-digit mobile (leading 9)
// or an 8-digit landline (leading 2-5).
func (p *Phone) Generate() string {
	ddd := ddds[rand.IntN(len(ddds))]
	var sb strings.Builder
	fmt.Fprintf(&sb, "%02d", ddd)
	if rand.IntN(2) == 0 {
		// 9-digit mobile: leading 9 + 8 random digits.
		sb.WriteByte('9')
		for i := 0; i < 8; i++ {
			sb.WriteByte(byte('0' + rand.IntN(10)))
		}
	} else {
		// 8-digit landline: leading 2-5 + 7 random digits.
		sb.WriteByte(byte('2' + rand.IntN(4)))
		for i := 0; i < 7; i++ {
			sb.WriteByte(byte('0' + rand.IntN(10)))
		}
	}
	return sb.String()
}
