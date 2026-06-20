package codegen

import (
	"fmt"
	"strings"
)

// emit_python_kinds.go holds the per-kind Python module renderers for the
// check-digit kinds (groups A, B, C). Each renders a deterministic module from
// the declarative KindPlan. The numeric kinds reuse the shared mod11 reducer;
// the irregular kinds (CNH coupled DVs, CNS sum-zero, CNPJ char-map) carry
// bespoke fragments, exactly as the TS reference.

// renderCPF emits the CPF module: two input-coupled mod-11 DVs, all-equal
// rejection, mask format, and ninth-digit origin.
func (e pythonEmitter) renderCPF(plan KindPlan) string {
	var b strings.Builder
	writePythonHeader(&b, []string{"weighted_sum", "compute_digit", "only_digits", "all_equal"}, "CPF_REGIONS")

	dv1 := pythonCheckDigitLiteral(plan.Checks[0])
	dv2 := pythonCheckDigitLiteral(plan.Checks[1])

	fmt.Fprintf(&b, `_DV1 = %s
_DV2 = %s


def validate_cpf(value: str) -> bool:
    """Report whether value is a valid CPF (formatted or not)."""
    d = only_digits(value)
    if len(d) != 11:
        return False
    if all_equal(d):
        return False
    digits = [int(c) for c in d]
    dv1 = compute_digit(weighted_sum(digits[0:9], _DV1["weights"]), _DV1)
    dv2 = compute_digit(weighted_sum(digits[0:10], _DV2["weights"]), _DV2)
    return dv1 == digits[9] and dv2 == digits[10]


def format_cpf(value: str) -> str:
    """Render value as XXX.XXX.XXX-XX, or raise on bad length."""
    d = only_digits(value)
    if len(d) != 11:
        %s
    return f"{d[0:3]}.{d[3:6]}.{d[6:9]}-{d[9:11]}"


def origin_cpf(value: str) -> str:
    """Return the issuing region from the 9th digit, or raise."""
    d = only_digits(value)
    if len(d) < 9:
        %s
    region = CPF_REGIONS.get(int(d[8]))
    if region is None:
        %s
    return region


def generate_cpf() -> str:
    """Return a random, valid CPF (unformatted, 11 digits)."""
    while True:
        number = [random.randint(0, 9) for _ in range(9)]
        number.append(compute_digit(weighted_sum(number, _DV1["weights"]), _DV1))
        number.append(compute_digit(weighted_sum(number, _DV2["weights"]), _DV2))
        out = "".join(str(x) for x in number)
        if not all_equal(out):
            return out
`, dv1, dv2, pythonRaise("ErrInvalidLength"),
		pythonRaise("ErrInvalidLength"), pythonRaise("ErrInvalidLength"))

	return b.String()
}

// renderSimpleNumeric emits a single-DV numeric kind (PIS): mod-11 DV over the
// first length-1 digits, all-equal rejection, and a mask format.
func (e pythonEmitter) renderSimpleNumeric(plan KindPlan, name string, length int) string {
	var b strings.Builder
	writePythonHeader(&b, []string{"weighted_sum", "compute_digit", "only_digits", "all_equal"}, "")

	dv := pythonCheckDigitLiteral(plan.Checks[0])
	base := length - 1
	mask := pythonMaskExpr(plan.Mask, "d")

	fmt.Fprintf(&b, `_DV = %[1]s


def validate_%[2]s(value: str) -> bool:
    """Report whether value is a valid %[2]s."""
    d = only_digits(value)
    if len(d) != %[3]d:
        return False
    if all_equal(d):
        return False
    digits = [int(c) for c in d]
    dv = compute_digit(weighted_sum(digits[0:%[4]d], _DV["weights"]), _DV)
    return dv == digits[%[4]d]


def format_%[2]s(value: str) -> str:
    """Render the canonical mask, or raise on bad length."""
    d = only_digits(value)
    if len(d) != %[3]d:
        %[5]s
    return %[6]s


def generate_%[2]s() -> str:
    """Return a random, valid %[2]s (unformatted)."""
    while True:
        b = [random.randint(0, 9) for _ in range(%[4]d)]
        b.append(compute_digit(weighted_sum(b, _DV["weights"]), _DV))
        out = "".join(str(x) for x in b)
        if not all_equal(out):
            return out
`, dv, name, length, base, pythonRaise("ErrInvalidLength"), mask)

	return b.String()
}

// renderRenavam emits RENAVAM: single (sum*10)%11 DV, all-equal rejection, and a
// left-pad-to-11 format (no separator mask).
func (e pythonEmitter) renderRenavam(plan KindPlan) string {
	var b strings.Builder
	writePythonHeader(&b, []string{"weighted_sum", "compute_digit", "only_digits", "all_equal"}, "")

	dv := pythonCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `_DV = %s


def validate_renavam(value: str) -> bool:
    """Report whether value is a valid 11-digit RENAVAM."""
    d = only_digits(value)
    if len(d) != 11:
        return False
    if all_equal(d):
        return False
    digits = [int(c) for c in d]
    dv = compute_digit(weighted_sum(digits[0:10], _DV["weights"]), _DV)
    return dv == digits[10]


def format_renavam(value: str) -> str:
    """Left-pad shorter inputs to 11 digits (no separator mask)."""
    d = only_digits(value)
    if len(d) < 11:
        d = ("0" * (11 - len(d))) + d
    return d


def generate_renavam() -> str:
    """Return a random, valid RENAVAM (unformatted, 11 digits)."""
    while True:
        b = [random.randint(0, 9) for _ in range(10)]
        b.append(compute_digit(weighted_sum(b, _DV["weights"]), _DV))
        out = "".join(str(x) for x in b)
        if not all_equal(out):
            return out
`, dv)

	return b.String()
}

// renderCNH emits the coupled-DV CNH module (bespoke fragment per the spec Note):
// DV1 descending 9..1 (raw remainder >=10 -> DV1=0, carry offset 2); DV2
// ascending 1..9 with the offset subtracted before the mod-11 fold.
func (e pythonEmitter) renderCNH() string {
	var b strings.Builder
	writePythonHeader(&b, []string{"only_digits", "all_equal"}, "")

	fmt.Fprintf(&b, `from typing import Tuple


def _cnh_check_digits(base: str) -> Tuple[int, int]:
    """Compute both coupled CNH check digits over the 9-digit base."""
    dsc = 0
    total = 0
    for i in range(9):
        total += int(base[i]) * (9 - i)
    r = total %% 11
    if r >= 10:
        dv1 = 0
        dsc = 2
    else:
        dv1 = r
    total = 0
    for i in range(9):
        total += int(base[i]) * (1 + i)
    r = (total %% 11) - dsc
    if r < 0:
        r += 11
    dv2 = 0 if r >= 10 else r
    return dv1, dv2


def validate_cnh(value: str) -> bool:
    """Report whether value is a valid 11-digit CNH."""
    d = only_digits(value)
    if len(d) != 11:
        return False
    if all_equal(d):
        return False
    dv1, dv2 = _cnh_check_digits(d[0:9])
    return dv1 == int(d[9]) and dv2 == int(d[10])


def format_cnh(value: str) -> str:
    """Return the cleaned 11-digit CNH (no separator mask)."""
    d = only_digits(value)
    if len(d) != 11:
        %s
    return d


def generate_cnh() -> str:
    """Return a random, valid 11-digit CNH (unformatted)."""
    while True:
        base = "".join(str(random.randint(0, 9)) for _ in range(9))
        dv1, dv2 = _cnh_check_digits(base)
        out = base + str(dv1) + str(dv2)
        if not all_equal(out):
            return out
`, pythonRaise("ErrInvalidLength"))

	return b.String()
}

// renderCNS emits the verify-only sum-zero module with prefix constraint.
func (e pythonEmitter) renderCNS(plan KindPlan) string {
	var b strings.Builder
	writePythonHeader(&b, []string{"weighted_sum", "compute_digit", "only_digits", "all_equal"}, "")

	dv := pythonCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `_DV = %s

_CNS_PREFIXES = ["1", "2", "7", "8", "9"]


def validate_cns(value: str) -> bool:
    """Report whether value is a well-formed CNS (sum %% 11 == 0)."""
    d = only_digits(value)
    if len(d) != 15:
        return False
    if all_equal(d):
        return False
    lead = d[0]
    if lead not in ("1", "2", "7", "8", "9"):
        return False
    digits = [int(c) for c in d]
    return compute_digit(weighted_sum(digits, _DV["weights"]), _DV) == 0


def format_cns(value: str) -> str:
    """Return the cleaned 15-digit CNS (no separator mask)."""
    d = only_digits(value)
    if len(d) != 15:
        %s
    return d


def generate_cns() -> str:
    """Return a random, valid CNS (15 digits, sum %% 11 == 0)."""
    while True:
        d = [random.choice(_CNS_PREFIXES)]
        for _ in range(1, 14):
            d.append(str(random.randint(0, 9)))
        partial = 0
        for i in range(14):
            partial += int(d[i]) * (15 - i)
        last = (11 - (partial %% 11)) %% 11
        if last == 10:
            continue
        d.append(str(last))
        out = "".join(d)
        if not all_equal(out):
            return out
`, dv, pythonRaise("ErrInvalidLength"))

	return b.String()
}

// renderCNPJ emits the alphanumeric CNPJ module (bespoke char-map + RL-cycling
// weights per the spec Note): two DVs, last two chars numeric, all-equal reject.
func (e pythonEmitter) renderCNPJ(plan KindPlan) string {
	var b strings.Builder
	writePythonHeader(&b, []string{"char_value", "weighted_sum", "compute_digit", "all_equal"}, "")

	dv := pythonCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `_DV = %s

_CNPJ_ALPHANUM = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"


def _cnpj_clean(value: str) -> str:
    """Uppercase and keep only [0-9A-Z], capped at 14 chars."""
    out = ""
    for ch in value:
        up = ch.upper()
        if ("0" <= up <= "9") or ("A" <= up <= "Z"):
            out += up
            if len(out) == 14:
                break
    return out


def _cnpj_dv(base: str) -> int:
    """Compute one check digit over the base string (RL-cycling weights)."""
    vals = [char_value(c) for c in base]
    return compute_digit(weighted_sum(vals, _DV["weights"], True), _DV)


def validate_cnpj(value: str) -> bool:
    """Report whether value is a valid alphanumeric CNPJ."""
    c = _cnpj_clean(value)
    if len(c) != 14:
        return False
    if all_equal(c):
        return False
    if c[12] < "0" or c[12] > "9":
        return False
    if c[13] < "0" or c[13] > "9":
        return False
    base = c[0:12]
    dv1 = _cnpj_dv(base)
    dv2 = _cnpj_dv(base + str(dv1))
    return dv1 == int(c[12]) and dv2 == int(c[13])


def format_cnpj(value: str) -> str:
    """Render value as XX.XXX.XXX/XXXX-XX, or raise on bad length."""
    c = _cnpj_clean(value)
    if len(c) != 14:
        %s
    return f"{c[0:2]}.{c[2:5]}.{c[5:8]}/{c[8:12]}-{c[12:14]}"


def generate_cnpj() -> str:
    """Return a random, valid alphanumeric CNPJ (14 chars, unformatted)."""
    base = "".join(random.choice(_CNPJ_ALPHANUM) for _ in range(12))
    dv1 = _cnpj_dv(base)
    dv2 = _cnpj_dv(base + str(dv1))
    return base + str(dv1) + str(dv2)
`, dv, pythonRaise("ErrInvalidLength"))

	return b.String()
}
