package codegen

import (
	"fmt"
	"strings"
)

// emit_rust_kinds.go holds the per-kind Rust module renderers for the check-digit
// kinds (groups A, B, C). Each renders a deterministic module from the
// declarative KindPlan. The numeric kinds reuse the shared mod11 reducer; the
// irregular kinds (CNH coupled DVs, CNS sum-zero, CNPJ char-map) carry bespoke
// fragments, exactly as the PHP/Python references.

// rustThrowExpr maps a sentinel to a bare Err(...) expression terminated by a
// comma, for use as a `match` arm body (expression position), where rustThrow's
// `return …;` statement form would read awkwardly. The expression-position
// analogue of rustThrow.
func rustThrowExpr(sentinel string) string {
	switch sentinel {
	case "ErrInvalidLength":
		return "Err(SeloError::InvalidLength),"
	default:
		return "Err(SeloError::InvalidFormat),"
	}
}

// rustThrowArm maps a sentinel to a diverging `return Err(…),` match arm, for
// use when the match is bound to a value (`let x = match … {}`) and the other
// arms yield a non-Result value, so the error arm must diverge rather than
// produce a Result. Comma-terminated (arm position) unlike rustThrow's `;`.
func rustThrowArm(sentinel string) string {
	switch sentinel {
	case "ErrInvalidLength":
		return "return Err(SeloError::InvalidLength),"
	default:
		return "return Err(SeloError::InvalidFormat),"
	}
}

// renderCPF emits the CPF module: two input-coupled mod-11 DVs, all-equal
// rejection, mask format, and ninth-digit origin.
func (e rustEmitter) renderCPF(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::data::CPF_REGIONS;\n")
	b.WriteString("use crate::mod11::{self, CheckDigit, Rule, SeloError};\n\n")

	dv1 := rustCheckDigitLiteral(plan.Checks[0])
	dv2 := rustCheckDigitLiteral(plan.Checks[1])

	fmt.Fprintf(&b, `const DV1: CheckDigit = %s;
const DV2: CheckDigit = %s;

fn to_ints(d: &str) -> Vec<i32> {
    d.bytes().map(|c| (c - b'0') as i32).collect()
}

/// Report whether value is a valid CPF (formatted or not).
pub fn validate_cpf(value: &str) -> bool {
    let d = mod11::only_digits(value);
    if d.len() != 11 {
        return false;
    }
    if mod11::all_equal(&d) {
        return false;
    }
    let digits = to_ints(&d);
    let dv1 = mod11::compute_digit(mod11::weighted_sum(&digits[0..9], DV1.weights, false), &DV1);
    let dv2 = mod11::compute_digit(mod11::weighted_sum(&digits[0..10], DV2.weights, false), &DV2);
    dv1 == digits[9] && dv2 == digits[10]
}

/// Render value as XXX.XXX.XXX-XX, or return an error on bad length.
pub fn format_cpf(value: &str) -> Result<String, SeloError> {
    let d = mod11::only_digits(value);
    if d.len() != 11 {
        %s
    }
    Ok(%s)
}

/// Return the issuing region from the 9th digit, or an error.
pub fn origin_cpf(value: &str) -> Result<String, SeloError> {
    let d = mod11::only_digits(value);
    if d.len() < 9 {
        %s
    }
    let key = (d.as_bytes()[8] - b'0') as i32;
    match CPF_REGIONS.iter().find(|(k, _)| *k == key).map(|(_, v)| *v) {
        Some(region) => Ok(region.to_string()),
        None => Err(SeloError::InvalidLength),
    }
}

/// Return a random, valid CPF (unformatted, 11 digits).
pub fn generate_cpf() -> String {
    loop {
        let mut number: Vec<i32> = Vec::with_capacity(11);
        for _ in 0..9 {
            number.push(mod11::rand_int(9) as i32);
        }
        number.push(mod11::compute_digit(mod11::weighted_sum(&number, DV1.weights, false), &DV1));
        number.push(mod11::compute_digit(mod11::weighted_sum(&number, DV2.weights, false), &DV2));
        let out: String = number.iter().map(|n| n.to_string()).collect();
        if !mod11::all_equal(&out) {
            return out;
        }
    }
}
`, dv1, dv2, rustThrow("ErrInvalidLength"), rustMaskExpr(plan.Mask, "d"),
		rustThrow("ErrInvalidLength"))

	return b.String()
}

// renderSimpleNumeric emits a single-DV numeric kind (PIS): mod-11 DV over the
// first length-1 digits, all-equal rejection, and a mask format.
func (e rustEmitter) renderSimpleNumeric(plan KindPlan, name string, length int) string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::mod11::{self, CheckDigit, Rule, SeloError};\n\n")

	dv := rustCheckDigitLiteral(plan.Checks[0])
	base := length - 1
	mask := rustMaskExpr(plan.Mask, "d")
	className := strings.ToUpper(name[:1]) + name[1:]

	fmt.Fprintf(&b, `const DV: CheckDigit = %[1]s;

/// Report whether value is a valid %[2]s.
pub fn validate_%[3]s(value: &str) -> bool {
    let d = mod11::only_digits(value);
    if d.len() != %[4]d {
        return false;
    }
    if mod11::all_equal(&d) {
        return false;
    }
    let digits: Vec<i32> = d.bytes().map(|c| (c - b'0') as i32).collect();
    let dv = mod11::compute_digit(mod11::weighted_sum(&digits[0..%[5]d], DV.weights, false), &DV);
    dv == digits[%[5]d]
}

/// Render the canonical mask, or return an error on bad length.
pub fn format_%[3]s(value: &str) -> Result<String, SeloError> {
    let d = mod11::only_digits(value);
    if d.len() != %[4]d {
        %[6]s
    }
    Ok(%[7]s)
}

/// Return a random, valid %[2]s (unformatted).
pub fn generate_%[3]s() -> String {
    loop {
        let mut b: Vec<i32> = Vec::with_capacity(%[4]d);
        for _ in 0..%[5]d {
            b.push(mod11::rand_int(9) as i32);
        }
        b.push(mod11::compute_digit(mod11::weighted_sum(&b, DV.weights, false), &DV));
        let out: String = b.iter().map(|n| n.to_string()).collect();
        if !mod11::all_equal(&out) {
            return out;
        }
    }
}
`, dv, className, name, length, base, rustThrow("ErrInvalidLength"), mask)

	return b.String()
}

// renderRenavam emits RENAVAM: single (sum*10)%11 DV, all-equal rejection, and a
// left-pad-to-11 format (no separator mask).
func (e rustEmitter) renderRenavam(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::mod11::{self, CheckDigit, Rule, SeloError};\n\n")

	dv := rustCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `const DV: CheckDigit = %s;

/// Report whether value is a valid 11-digit RENAVAM.
pub fn validate_renavam(value: &str) -> bool {
    let d = mod11::only_digits(value);
    if d.len() != 11 {
        return false;
    }
    if mod11::all_equal(&d) {
        return false;
    }
    let digits: Vec<i32> = d.bytes().map(|c| (c - b'0') as i32).collect();
    let dv = mod11::compute_digit(mod11::weighted_sum(&digits[0..10], DV.weights, false), &DV);
    dv == digits[10]
}

/// Left-pad shorter inputs to 11 digits (no separator mask).
pub fn format_renavam(value: &str) -> Result<String, SeloError> {
    let mut d = mod11::only_digits(value);
    if d.len() < 11 {
        d = format!("{}{}", "0".repeat(11 - d.len()), d);
    }
    Ok(d)
}

/// Return a random, valid RENAVAM (unformatted, 11 digits).
pub fn generate_renavam() -> String {
    loop {
        let mut b: Vec<i32> = Vec::with_capacity(11);
        for _ in 0..10 {
            b.push(mod11::rand_int(9) as i32);
        }
        b.push(mod11::compute_digit(mod11::weighted_sum(&b, DV.weights, false), &DV));
        let out: String = b.iter().map(|n| n.to_string()).collect();
        if !mod11::all_equal(&out) {
            return out;
        }
    }
}
`, dv)

	return b.String()
}

// renderCNH emits the coupled-DV CNH module (bespoke fragment per the spec Note):
// DV1 descending 9..1 (raw remainder >=10 -> DV1=0, carry offset 2); DV2
// ascending 1..9 with the offset subtracted before the mod-11 fold.
func (e rustEmitter) renderCNH() string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::mod11::{self, SeloError};\n\n")

	fmt.Fprintf(&b, `/// Compute both coupled CNH check digits over the 9-digit base.
fn check_digits(base: &str) -> (i32, i32) {
    let bytes = base.as_bytes();
    let dsc;
    let dv1;
    let mut total = 0;
    for i in 0..9 {
        total += ((bytes[i] - b'0') as i32) * (9 - i as i32);
    }
    let mut r = total %% 11;
    if r >= 10 {
        dv1 = 0;
        dsc = 2;
    } else {
        dv1 = r;
        dsc = 0;
    }
    total = 0;
    for i in 0..9 {
        total += ((bytes[i] - b'0') as i32) * (1 + i as i32);
    }
    r = (total %% 11) - dsc;
    if r < 0 {
        r += 11;
    }
    let dv2 = if r >= 10 { 0 } else { r };
    (dv1, dv2)
}

/// Report whether value is a valid 11-digit CNH.
pub fn validate_cnh(value: &str) -> bool {
    let d = mod11::only_digits(value);
    if d.len() != 11 {
        return false;
    }
    if mod11::all_equal(&d) {
        return false;
    }
    let (dv1, dv2) = check_digits(&d[0..9]);
    let bytes = d.as_bytes();
    dv1 == (bytes[9] - b'0') as i32 && dv2 == (bytes[10] - b'0') as i32
}

/// Return the cleaned 11-digit CNH (no separator mask).
pub fn format_cnh(value: &str) -> Result<String, SeloError> {
    let d = mod11::only_digits(value);
    if d.len() != 11 {
        %s
    }
    Ok(d)
}

/// Return a random, valid 11-digit CNH (unformatted).
pub fn generate_cnh() -> String {
    loop {
        let mut base = String::with_capacity(9);
        for _ in 0..9 {
            base.push_str(&mod11::rand_int(9).to_string());
        }
        let (dv1, dv2) = check_digits(&base);
        let out = format!("{}{}{}", base, dv1, dv2);
        if !mod11::all_equal(&out) {
            return out;
        }
    }
}
`, rustThrow("ErrInvalidLength"))

	return b.String()
}

// renderCNS emits the verify-only sum-zero module with prefix constraint.
func (e rustEmitter) renderCNS(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::mod11::{self, CheckDigit, Rule, SeloError};\n\n")

	dv := rustCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `const DV: CheckDigit = %s;

const PREFIXES: &[u8] = b"12789";

/// Report whether value is a well-formed CNS (sum %% 11 == 0).
pub fn validate_cns(value: &str) -> bool {
    let d = mod11::only_digits(value);
    if d.len() != 15 {
        return false;
    }
    if mod11::all_equal(&d) {
        return false;
    }
    let lead = d.as_bytes()[0];
    if !PREFIXES.contains(&lead) {
        return false;
    }
    let digits: Vec<i32> = d.bytes().map(|c| (c - b'0') as i32).collect();
    mod11::compute_digit(mod11::weighted_sum(&digits, DV.weights, false), &DV) == 0
}

/// Return the cleaned 15-digit CNS (no separator mask).
pub fn format_cns(value: &str) -> Result<String, SeloError> {
    let d = mod11::only_digits(value);
    if d.len() != 15 {
        %s
    }
    Ok(d)
}

/// Return a random, valid CNS (15 digits, sum %% 11 == 0).
pub fn generate_cns() -> String {
    loop {
        let mut d: Vec<i32> = Vec::with_capacity(15);
        d.push((PREFIXES[mod11::rand_int(PREFIXES.len() as u64 - 1) as usize] - b'0') as i32);
        for _ in 1..14 {
            d.push(mod11::rand_int(9) as i32);
        }
        let mut partial = 0;
        for i in 0..14 {
            partial += d[i] * (15 - i as i32);
        }
        let last = (11 - (partial %% 11)) %% 11;
        if last == 10 {
            continue;
        }
        d.push(last);
        let out: String = d.iter().map(|n| n.to_string()).collect();
        if !mod11::all_equal(&out) {
            return out;
        }
    }
}
`, dv, rustThrow("ErrInvalidLength"))

	return b.String()
}

// renderCNPJ emits the alphanumeric CNPJ module (bespoke char-map + RL-cycling
// weights per the spec Note): two DVs, last two chars numeric, all-equal reject.
func (e rustEmitter) renderCNPJ(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::mod11::{self, CheckDigit, Rule, SeloError};\n\n")

	dv := rustCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `const DV: CheckDigit = %s;

const ALPHANUM: &[u8] = b"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ";

/// Uppercase and keep only [0-9A-Z], capped at 14 chars.
fn clean(value: &str) -> String {
    let mut out = String::with_capacity(14);
    for ch in value.bytes() {
        let up = ch.to_ascii_uppercase();
        if up.is_ascii_digit() || up.is_ascii_uppercase() {
            out.push(up as char);
            if out.len() == 14 {
                break;
            }
        }
    }
    out
}

/// Compute one check digit over the base string (RL-cycling weights).
fn dv(base: &str) -> i32 {
    let vals: Vec<i32> = base.bytes().map(mod11::char_value).collect();
    mod11::compute_digit(mod11::weighted_sum(&vals, DV.weights, true), &DV)
}

/// Report whether value is a valid alphanumeric CNPJ.
pub fn validate_cnpj(value: &str) -> bool {
    let c = clean(value);
    if c.len() != 14 {
        return false;
    }
    if mod11::all_equal(&c) {
        return false;
    }
    let bytes = c.as_bytes();
    if !bytes[12].is_ascii_digit() {
        return false;
    }
    if !bytes[13].is_ascii_digit() {
        return false;
    }
    let base = &c[0..12];
    let dv1 = dv(base);
    let dv2 = dv(&format!("{}{}", base, dv1));
    dv1 == (bytes[12] - b'0') as i32 && dv2 == (bytes[13] - b'0') as i32
}

/// Render value as XX.XXX.XXX/XXXX-XX, or return an error on bad length.
pub fn format_cnpj(value: &str) -> Result<String, SeloError> {
    let c = clean(value);
    if c.len() != 14 {
        %s
    }
    Ok(format!("{}.{}.{}/{}-{}", &c[0..2], &c[2..5], &c[5..8], &c[8..12], &c[12..14]))
}

/// Return a random, valid alphanumeric CNPJ (14 chars, unformatted).
pub fn generate_cnpj() -> String {
    let mut base = String::with_capacity(12);
    for _ in 0..12 {
        base.push(ALPHANUM[mod11::rand_int(ALPHANUM.len() as u64 - 1) as usize] as char);
    }
    let dv1 = dv(&base);
    let dv2 = dv(&format!("{}{}", base, dv1));
    format!("{}{}{}", base, dv1, dv2)
}
`, dv, rustThrow("ErrInvalidLength"))

	return b.String()
}
