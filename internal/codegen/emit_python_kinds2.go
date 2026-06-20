package codegen

import (
	"fmt"
	"strings"

	"github.com/inovacc/selo"
)

// emit_python_kinds2.go holds the remaining per-kind Python module renderers
// (RG/IE UF-scoped, plate/pix regex, cep/phone table lookup, voter dual-DV).
// Every algorithm is translated verbatim from the TS reference.

// renderRG emits the UF-scoped RG module: 8 base digits + 1 check char
// (10->'X', 11->'0'); SP and RJ share the algorithm.
func (e pythonEmitter) renderRG(plan KindPlan) string {
	var b strings.Builder
	writePythonHeader(&b, []string{"weighted_sum", "compute_digit", "encode_digit"}, "")

	dv := pythonCheckDigitLiteral(plan.Checks[0])
	ufs := pythonStringList([]string{"SP", "RJ"})
	fmt.Fprintf(&b, `from typing import List, Optional, Tuple

_DV = %s

RG_UFS: List[str] = %s


def _rg_parse(value: str) -> Optional[Tuple[List[int], int]]:
    """Strip formatting and return (base_digits, check) or None."""
    cleaned = ""
    for ch in value:
        if ("0" <= ch <= "9") or ch == "X" or ch == "x":
            cleaned += ch
    if len(cleaned) != 9:
        return None
    last = cleaned[8]
    if last == "X" or last == "x":
        check = 10
    elif last == "0":
        check = 11
    elif "1" <= last <= "9":
        check = int(last)
    else:
        return None
    base: List[int] = []
    for i in range(8):
        c = cleaned[i]
        if c < "0" or c > "9":
            return None
        base.append(int(c))
    return base, check


def validate_rg_for_uf(value: str, uf: str) -> bool:
    """Validate value as an RG for the given UF (SP/RJ only)."""
    if uf not in RG_UFS:
        return False
    p = _rg_parse(value)
    if p is None:
        return False
    return compute_digit(weighted_sum(p[0], _DV["weights"]), _DV) == p[1]


def validate_rg(value: str) -> bool:
    """Validate value under any implemented UF (first match wins)."""
    return any(validate_rg_for_uf(value, uf) for uf in RG_UFS)


def format_rg(value: str) -> str:
    """Render an RG as XX.XXX.XXX-C (check char normalized)."""
    p = _rg_parse(value)
    if p is None:
        %s
    check_char = encode_digit(p[1], _DV)
    d = "".join(str(x) for x in p[0])
    return f"{d[0:2]}.{d[2:5]}.{d[5:8]}-{check_char}"


def generate_rg() -> str:
    """Return a valid SP-style RG in masked form (XX.XXX.XXX-C)."""
    base = [random.randint(0, 9) for _ in range(8)]
    dv = compute_digit(weighted_sum(base, _DV["weights"]), _DV)
    check_char = encode_digit(dv, _DV)
    d = "".join(str(x) for x in base)
    return f"{d[0:2]}.{d[2:5]}.{d[5:8]}-{check_char}"
`, dv, ufs, pythonRaise("ErrInvalidFormat"))

	return b.String()
}

// renderIE emits the UF-scoped IE module (SP only): two rightmost-digit DVs at
// non-adjacent positions 9 and 12.
func (e pythonEmitter) renderIE(plan KindPlan) string {
	var b strings.Builder
	writePythonHeader(&b, []string{"weighted_sum", "compute_digit", "only_digits"}, "")

	dv1 := pythonCheckDigitLiteral(plan.Checks[0])
	dv2 := pythonCheckDigitLiteral(plan.Checks[1])
	ufs := pythonStringList([]string{"SP"})
	fmt.Fprintf(&b, `from typing import List

_DV1 = %s
_DV2 = %s

IE_UFS: List[str] = %s

_IE_W1 = [1, 3, 4, 5, 6, 7, 8, 10]
_IE_W2 = [3, 2, 10, 9, 8, 7, 6, 5, 4, 3, 2]


def _ie_sp_validate(d: str) -> bool:
    """Validate a 12-digit São Paulo IE."""
    if len(d) != 12:
        return False
    digits = [int(c) for c in d]
    if compute_digit(weighted_sum(digits[0:8], _DV1["weights"]), _DV1) != digits[8]:
        return False
    return compute_digit(weighted_sum(digits[0:11], _DV2["weights"]), _DV2) == digits[11]


def validate_ie_for_uf(value: str, uf: str) -> bool:
    """Validate value as an IE for the given UF (SP only)."""
    if uf != "SP":
        return False
    d = only_digits(value)
    if len(d) != 12:
        return False
    return _ie_sp_validate(d)


def validate_ie(value: str) -> bool:
    """Validate value under any implemented UF (first match wins)."""
    return any(validate_ie_for_uf(value, uf) for uf in IE_UFS)


def format_ie(value: str) -> str:
    """Render SP IE as AAA.AAA.AAA.AAA, or raise when invalid."""
    d = only_digits(value)
    if len(d) == 12 and _ie_sp_validate(d):
        return f"{d[0:3]}.{d[3:6]}.{d[6:9]}.{d[9:12]}"
    %s


def _ie_rightmost_dv(digits: List[int], weights: List[int]) -> int:
    total = 0
    for i in range(len(weights)):
        total += digits[i] * weights[i]
    return (total %% 11) %% 10


def generate_ie() -> str:
    """Return a valid SP IE in masked form (AAA.AAA.AAA.AAA)."""
    d = [0] * 12
    for i in range(8):
        d[i] = random.randint(0, 9)
    d[8] = _ie_rightmost_dv(d[0:8], _IE_W1)
    d[9] = random.randint(0, 9)
    d[10] = random.randint(0, 9)
    d[11] = _ie_rightmost_dv(d[0:11], _IE_W2)
    s = "".join(str(x) for x in d)
    return f"{s[0:3]}.{s[3:6]}.{s[6:9]}.{s[9:12]}"
`, dv1, dv2, ufs, pythonRaise("ErrInvalidFormat"))

	return b.String()
}

// renderPlate emits the regex-only plate module (national + Mercosul).
func (e pythonEmitter) renderPlate() string {
	var b strings.Builder
	b.WriteString(pythonHeaderComment())
	b.WriteString("\n")
	b.WriteString("from __future__ import annotations\n\n")
	b.WriteString("import random\nimport re\n\n")
	fmt.Fprintf(&b, `_NATIONAL = re.compile(r"^[A-Z]{3}-?[0-9]{4}$")
_MERCOSUL = re.compile(r"^[A-Z]{3}[0-9][A-Z][0-9]{2}$")

_PLATE_LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"


def validate_plate(value: str) -> bool:
    """Report whether value is a national or Mercosul plate."""
    v = value.strip().upper()
    return bool(_NATIONAL.match(v)) or bool(_MERCOSUL.match(v))


def format_plate(value: str) -> str:
    """Canonicalize the plate (national gains a dash), or raise."""
    v = value.strip().upper()
    if _MERCOSUL.match(v):
        return v
    if _NATIONAL.match(v):
        s = v.replace("-", "")
        return f"{s[0:3]}-{s[3:7]}"
    %s


def generate_plate() -> str:
    """Return a random valid plate (national or Mercosul)."""
    def rl() -> str:
        return random.choice(_PLATE_LETTERS)

    def rd() -> str:
        return str(random.randint(0, 9))

    letters = rl() + rl() + rl()
    if random.random() < 0.5:
        return letters + "-" + rd() + rd() + rd() + rd()
    return letters + rd() + rl() + rd() + rd()
`, pythonRaise("ErrInvalidFormat"))

	return b.String()
}

// renderPIX emits the composite PIX module: dispatch EVP -> email -> phone ->
// CPF -> CNPJ, reusing the CPF/CNPJ validators.
func (e pythonEmitter) renderPIX() string {
	var b strings.Builder
	b.WriteString(pythonHeaderComment())
	b.WriteString("\n")
	b.WriteString("from __future__ import annotations\n\n")
	b.WriteString("import random\nimport re\nimport uuid\n\n")
	b.WriteString("from typing import Optional\n\n")
	b.WriteString("from .mod11 import only_digits\n")
	b.WriteString("from .cpf import validate_cpf\n")
	b.WriteString("from .cnpj import validate_cnpj\n\n")
	fmt.Fprintf(&b, `_EVP = re.compile(r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$")
_PHONE = re.compile(r"^\+55\d{10,11}$")
_EMAIL = re.compile(r"^[A-Za-z0-9._%%+\-]+@[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?)+$")


def detect_pix_kind(value: str) -> Optional[str]:
    """Report the PIX key kind, or None when value is not a key."""
    v = value.strip()
    if _EVP.match(v):
        return "evp"
    if "@" in v:
        return "email" if _EMAIL.match(v) else None
    if v.startswith("+"):
        return "phone" if _PHONE.match(v) else None
    digits = len(only_digits(v))
    if digits == 11 and validate_cpf(v):
        return "cpf"
    if digits == 14 and validate_cnpj(v):
        return "cnpj"
    return None


def validate_pix(value: str) -> bool:
    """Report whether value is a well-formed PIX key of any kind."""
    return detect_pix_kind(value) is not None


def format_pix(value: str) -> str:
    """Return the trimmed key verbatim, or raise when invalid."""
    v = value.strip()
    if detect_pix_kind(v) is None:
        %s
    return v


def generate_pix() -> str:
    """Return a random, valid EVP (UUIDv4) PIX key."""
    return str(uuid.UUID(int=random.getrandbits(128), version=4))
`, pythonRaise("ErrInvalidLength"))

	return b.String()
}

// renderCEP emits the table-lookup CEP module: prefix-range validation, mask
// format, and UF origin from the embedded CEP_RANGES table.
func (e pythonEmitter) renderCEP() string {
	var b strings.Builder
	writePythonHeader(&b, []string{"only_digits"}, "CEP_RANGES")
	fmt.Fprintf(&b, `from typing import Optional


def _cep_range_for(prefix: int) -> Optional[str]:
    """Return the UF whose prefix range contains prefix, or None."""
    for r in CEP_RANGES:
        if r["from_"] <= prefix <= r["to"]:
            return r["uf"]
    return None


def validate_cep(value: str) -> bool:
    """Report whether value is a CEP whose prefix maps to a UF."""
    d = only_digits(value)
    if len(d) != 8:
        return False
    prefix = int(d[0:3])
    return _cep_range_for(prefix) is not None


def format_cep(value: str) -> str:
    """Mask a CEP as #####-###, or raise on bad length."""
    d = only_digits(value)
    if len(d) != 8:
        %s
    return f"{d[0:5]}-{d[5:8]}"


def origin_cep(value: str) -> str:
    """Return the UF whose prefix range contains value, or raise."""
    d = only_digits(value)
    if len(d) != 8:
        %s
    uf = _cep_range_for(int(d[0:3]))
    if uf is None:
        %s
    return uf


def generate_cep() -> str:
    """Return a random, valid 8-digit CEP (unformatted)."""
    r = random.choice(CEP_RANGES)
    prefix = r["from_"] + random.randint(0, r["to"] - r["from_"])
    suffix = random.randint(0, 99999)
    return f"{prefix:03d}{suffix:05d}"
`, pythonRaise("ErrInvalidLength"), pythonRaise("ErrInvalidLength"), pythonRaise("ErrInvalidFormat"))

	return b.String()
}

// renderPhone emits the table-lookup phone module: optional +55/0055 prefix,
// DDD->UF validation, mobile/landline mask, and DDD origin.
func (e pythonEmitter) renderPhone() string {
	var b strings.Builder
	writePythonHeader(&b, []string{"only_digits"}, "DDD_TO_UF")
	fmt.Fprintf(&b, `from typing import Optional


def _national_number(d: str) -> Optional[str]:
    """Strip a +55/0055 country prefix, returning the rest or None."""
    if d.startswith("0055"):
        d = d[4:]
    elif d.startswith("55") and len(d) > 11:
        d = d[2:]
    if d == "":
        return None
    return d


def validate_phone(value: str) -> bool:
    """Report whether value is a valid phone whose DDD maps to a UF."""
    n = _national_number(only_digits(value))
    if n is None:
        return False
    if len(n) != 10 and len(n) != 11:
        return False
    ddd = n[0:2]
    if ddd not in DDD_TO_UF:
        return False
    if len(n) == 11 and n[2] != "9":
        return False
    return True


def format_phone(value: str) -> str:
    """Mask as (DD) NNNNN-NNNN or (DD) NNNN-NNNN, or raise."""
    n = _national_number(only_digits(value))
    if n is None or (len(n) != 10 and len(n) != 11):
        %s
    ddd = n[0:2]
    if ddd not in DDD_TO_UF:
        %s
    sub = n[2:]
    if len(sub) == 9:
        return f"({ddd}) {sub[0:5]}-{sub[5:9]}"
    return f"({ddd}) {sub[0:4]}-{sub[4:8]}"


def origin_phone(value: str) -> str:
    """Return the UF for the phone's DDD, or raise."""
    n = _national_number(only_digits(value))
    if n is None or (len(n) != 10 and len(n) != 11):
        %s
    ddd = n[0:2]
    uf = DDD_TO_UF.get(ddd)
    if uf is None:
        %s
    return uf


def generate_phone() -> str:
    """Return a random valid Brazilian phone (unformatted national digits)."""
    ddd = random.choice(list(DDD_TO_UF.keys()))
    if random.random() < 0.5:
        sub = "9" + "".join(str(random.randint(0, 9)) for _ in range(8))
        return ddd + sub
    first = str(2 + random.randint(0, 3))
    sub = first + "".join(str(random.randint(0, 9)) for _ in range(7))
    return ddd + sub
`, pythonRaise("ErrInvalidLength"), pythonRaise("ErrInvalidFormat"),
		pythonRaise("ErrInvalidLength"), pythonRaise("ErrInvalidFormat"))

	return b.String()
}

// renderVoterID emits the dual-DV voter module (bespoke per the spec Note): DV1
// over the 8 sequence digits; DV2 over [ufDigit0, ufDigit1, dv1]; UF code 01..28.
func (e pythonEmitter) renderVoterID(plan KindPlan) string {
	var b strings.Builder
	writePythonHeader(&b, []string{"weighted_sum", "compute_digit", "only_digits", "all_equal"}, "VOTER_UF_NAMES")

	dv1 := pythonCheckDigitLiteral(plan.Checks[0])
	dv2 := pythonCheckDigitLiteral(plan.Checks[1])
	fmt.Fprintf(&b, `_DV1 = %s
_DV2 = %s


def _voter_dv1(d: str) -> int:
    """Compute the first check digit over the 8 sequence digits."""
    seq = [int(c) for c in d[0:8]]
    return compute_digit(weighted_sum(seq, _DV1["weights"]), _DV1)


def _voter_dv2(d: str, dv1: int) -> int:
    """Compute the second check digit over [uf0, uf1, dv1]."""
    vals = [int(d[8]), int(d[9]), dv1]
    return compute_digit(weighted_sum(vals, _DV2["weights"]), _DV2)


def validate_voter_id(value: str) -> bool:
    """Report whether value is a well-formed Título Eleitoral."""
    d = only_digits(value)
    if len(d) != 12:
        return False
    if all_equal(d):
        return False
    uf_code = int(d[8]) * 10 + int(d[9])
    if uf_code < 1 or uf_code > 28:
        return False
    dv1 = _voter_dv1(d)
    dv2 = _voter_dv2(d, dv1)
    return dv1 == int(d[10]) and dv2 == int(d[11])


def format_voter_id(value: str) -> str:
    """Group the voter ID as "SSSS SSSS UUDD", or raise."""
    d = only_digits(value)
    if len(d) != 12:
        %s
    return f"{d[0:4]} {d[4:8]} {d[8:12]}"


def origin_voter_id(value: str) -> str:
    """Return the region encoded in the UF code, or raise."""
    d = only_digits(value)
    if len(d) != 12:
        %s
    uf_code = int(d[8]) * 10 + int(d[9])
    name = VOTER_UF_NAMES.get(uf_code)
    if name is None:
        %s
    return name


def generate_voter_id() -> str:
    """Return a random, valid Título Eleitoral (12 digits, unformatted)."""
    while True:
        d = [0] * 12
        for i in range(8):
            d[i] = random.randint(0, 9)
        uf = 1 + random.randint(0, 27)
        d[8] = uf // 10
        d[9] = uf %% 10
        s = "".join(str(x) for x in d[0:10])
        dv1 = _voter_dv1(s)
        d[10] = dv1
        d[11] = _voter_dv2(s, dv1)
        out = "".join(str(x) for x in d)
        if not all_equal(out):
            return out
`, dv1, dv2, pythonRaise("ErrInvalidLength"),
		pythonRaise("ErrInvalidLength"), pythonRaise("ErrInvalidFormat"))

	return b.String()
}

// pythonHasOrigin reports whether kind has an origin resolver in the generated
// Python module (mirrors originFnName in the TS test renderer).
func pythonHasOrigin(kind selo.Kind) bool {
	switch kind { //nolint:exhaustive // only origin-capable kinds return true; all others fall through
	case selo.KindCPF, selo.KindCEP, selo.KindPhone, selo.KindVoterID:
		return true
	default:
		return false
	}
}
