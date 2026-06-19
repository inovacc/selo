package selo

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
)

// PIX key-kind identifiers returned by DetectPIXKind.
const (
	PIXKindCPF   = "cpf"
	PIXKindCNPJ  = "cnpj"
	PIXKindEmail = "email"
	PIXKindPhone = "phone"
	PIXKindEVP   = "evp"
)

// pixEmailRe is an RFC 5322-lite matcher: a sane local part, a single '@', and a
// dotted domain. It deliberately rejects consecutive '@', missing domain dot, and
// leading/trailing dots in the domain.
var pixEmailRe = regexp.MustCompile(
	`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?)+$`,
)

// pixPhoneRe matches the strict E.164 Brazilian form required by PIX: "+55", a
// 2-digit DDD, then an 8- or 9-digit subscriber number (10 or 11 trailing digits).
var pixPhoneRe = regexp.MustCompile(`^\+55\d{10,11}$`)

// pixEVPRe matches a canonical UUIDv4 (version nibble 4, variant nibble 8/9/a/b).
var pixEVPRe = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`,
)

// DetectPIXKind reports which of the five BCB PIX key kinds value is, and whether it
// is a well-formed key at all. The five kinds are checked in a deterministic order:
// EVP, email, phone, then CPF/CNPJ by digit length. It returns ("", false) when value
// is not a well-formed key of any kind.
func DetectPIXKind(value string) (string, bool) {
	v := strings.TrimSpace(value)

	// EVP (UUIDv4) — most specific shape, checked first.
	if pixEVPRe.MatchString(v) {
		return PIXKindEVP, true
	}

	// Email — contains '@' and matches the RFC5322-lite shape.
	if strings.Contains(v, "@") {
		if pixEmailRe.MatchString(v) {
			return PIXKindEmail, true
		}
		return "", false
	}

	// Phone — strict E.164 "+55..." form.
	if strings.HasPrefix(v, "+") {
		if pixPhoneRe.MatchString(v) {
			return PIXKindPhone, true
		}
		return "", false
	}

	// CPF / CNPJ — discriminate by digit count, then run the real check-digit validator.
	switch len(onlyDigits(v)) {
	case CpfLength:
		if NewCPF().Validate(v) {
			return PIXKindCPF, true
		}
	case CnpjLength:
		if NewCNPJ().Validate(v) {
			return PIXKindCNPJ, true
		}
	}

	return "", false
}

// PIX validates Brazilian PIX keys across all five BCB key kinds: CPF, CNPJ, email,
// phone (E.164 "+55..."), and EVP (UUIDv4). It implements the Document interface and
// self-registers in init().
type PIX struct{}

// NewPIX creates a new PIX key validator instance.
func NewPIX() *PIX { return &PIX{} }

// Kind returns KindPIX.
func (p *PIX) Kind() Kind { return KindPIX }

// Validate reports whether value is a well-formed PIX key of any of the five kinds.
func (p *PIX) Validate(value string) bool {
	_, ok := DetectPIXKind(value)
	return ok
}

// Format returns the cleaned (whitespace-trimmed) PIX key. PIX keys have no canonical
// mask, so the key is returned verbatim; CPF/CNPJ/email/phone formatting is intentionally
// preserved because the stored key string is what the bank matches. It returns
// ErrInvalidLength (wrapped) when value is not a valid PIX key.
func (p *PIX) Format(value string) (string, error) {
	v := strings.TrimSpace(value)
	if _, ok := DetectPIXKind(v); !ok {
		return "", fmt.Errorf("selo: %q is not a valid PIX key: %w", value, ErrInvalidLength)
	}
	return v, nil
}

// Generate returns a random EVP (Endereço Virtual de Pagamento) PIX key: a canonical
// lowercase UUIDv4, which is itself always a valid PIX key. Uses crypto/rand so generated
// keys are unpredictable (PIX EVP keys are bank-assigned random identifiers).
func (p *PIX) Generate() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand.Read never fails on supported platforms; fall back deterministically.
		return "00000000-0000-4000-8000-000000000000"
	}

	// Set version (4) and variant (10xx) bits per RFC 4122.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	const hexdigits = "0123456789abcdef"
	var out [36]byte
	pos := 0
	for i, by := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[pos] = '-'
			pos++
		}
		out[pos] = hexdigits[by>>4]
		out[pos+1] = hexdigits[by&0x0f]
		pos += 2
	}

	return string(out[:])
}

func init() { Register(&PIX{}) }

// compile-time guarantee that *PIX satisfies the Document interface.
var _ Document = (*PIX)(nil)
