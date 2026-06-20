package codegen

import (
	"fmt"
	"strings"
)

// emit_rust_kinds2.go holds the remaining per-kind Rust module renderers (RG/IE
// UF-scoped, plate/pix/phone hand-rolled matchers, cep/phone table lookup, voter
// dual-DV). Every algorithm is translated verbatim from the PHP/Python
// references; the regex-based kinds (plate, pix, phone) use explicit char-class
// matchers so the generated crate has zero runtime dependencies.

// renderRG emits the UF-scoped RG module: 8 base digits + 1 check char
// (10->'X', 11->'0'); SP and RJ share the algorithm.
func (e rustEmitter) renderRG(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::mod11::{self, CheckDigit, Rule, SeloError};\n\n")

	dv := rustCheckDigitLiteral(plan.Checks[0])
	ufs := rustStringList([]string{"SP", "RJ"})
	fmt.Fprintf(&b, `const DV: CheckDigit = %s;

pub const UFS: &[&str] = %s;

/// Strip formatting and return (base_digits, check) or None.
fn parse(value: &str) -> Option<(Vec<i32>, i32)> {
    let mut cleaned: Vec<u8> = Vec::new();
    for &ch in value.as_bytes() {
        if ch.is_ascii_digit() || ch == b'X' || ch == b'x' {
            cleaned.push(ch);
        }
    }
    if cleaned.len() != 9 {
        return None;
    }
    let last = cleaned[8];
    let check = if last == b'X' || last == b'x' {
        10
    } else if last == b'0' {
        11
    } else if (b'1'..=b'9').contains(&last) {
        (last - b'0') as i32
    } else {
        return None;
    };
    let mut base: Vec<i32> = Vec::with_capacity(8);
    for &c in &cleaned[0..8] {
        if !c.is_ascii_digit() {
            return None;
        }
        base.push((c - b'0') as i32);
    }
    Some((base, check))
}

/// Validate value as an RG for the given UF (SP/RJ only).
pub fn validate_rg_for_uf(value: &str, uf: &str) -> bool {
    if !UFS.contains(&uf) {
        return false;
    }
    match parse(value) {
        Some((base, check)) => {
            mod11::compute_digit(mod11::weighted_sum(&base, DV.weights, false), &DV) == check
        }
        None => false,
    }
}

/// Validate value under any implemented UF (first match wins).
pub fn validate_rg(value: &str) -> bool {
    UFS.iter().any(|uf| validate_rg_for_uf(value, uf))
}

/// Render an RG as XX.XXX.XXX-C (check char normalized).
pub fn format_rg(value: &str) -> Result<String, SeloError> {
    let (base, check) = match parse(value) {
        Some(p) => p,
        None => %s
    };
    let check_char = mod11::encode_digit(check, &DV);
    let d: String = base.iter().map(|n| n.to_string()).collect();
    Ok(format!("{}.{}.{}-{}", &d[0..2], &d[2..5], &d[5..8], check_char))
}

/// Return a valid SP-style RG in masked form (XX.XXX.XXX-C).
pub fn generate_rg() -> String {
    let mut base: Vec<i32> = Vec::with_capacity(8);
    for _ in 0..8 {
        base.push(mod11::rand_int(9) as i32);
    }
    let dv = mod11::compute_digit(mod11::weighted_sum(&base, DV.weights, false), &DV);
    let check_char = mod11::encode_digit(dv, &DV);
    let d: String = base.iter().map(|n| n.to_string()).collect();
    format!("{}.{}.{}-{}", &d[0..2], &d[2..5], &d[5..8], check_char)
}
`, dv, ufs, rustThrowArm("ErrInvalidFormat"))

	return b.String()
}

// renderIE emits the UF-scoped IE module (SP only): two rightmost-digit DVs at
// non-adjacent positions 9 and 12.
func (e rustEmitter) renderIE(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::mod11::{self, CheckDigit, Rule, SeloError};\n\n")

	dv1 := rustCheckDigitLiteral(plan.Checks[0])
	dv2 := rustCheckDigitLiteral(plan.Checks[1])
	ufs := rustStringList([]string{"SP"})
	fmt.Fprintf(&b, `const DV1: CheckDigit = %s;
const DV2: CheckDigit = %s;

pub const UFS: &[&str] = %s;

const W1: &[i32] = &[1, 3, 4, 5, 6, 7, 8, 10];
const W2: &[i32] = &[3, 2, 10, 9, 8, 7, 6, 5, 4, 3, 2];

/// Validate a 12-digit São Paulo IE.
fn sp_validate(d: &str) -> bool {
    if d.len() != 12 {
        return false;
    }
    let digits: Vec<i32> = d.bytes().map(|c| (c - b'0') as i32).collect();
    if mod11::compute_digit(mod11::weighted_sum(&digits[0..8], DV1.weights, false), &DV1) != digits[8] {
        return false;
    }
    mod11::compute_digit(mod11::weighted_sum(&digits[0..11], DV2.weights, false), &DV2) == digits[11]
}

/// Validate value as an IE for the given UF (SP only).
pub fn validate_ie_for_uf(value: &str, uf: &str) -> bool {
    if uf != "SP" {
        return false;
    }
    let d = mod11::only_digits(value);
    if d.len() != 12 {
        return false;
    }
    sp_validate(&d)
}

/// Validate value under any implemented UF (first match wins).
pub fn validate_ie(value: &str) -> bool {
    UFS.iter().any(|uf| validate_ie_for_uf(value, uf))
}

/// Render SP IE as AAA.AAA.AAA.AAA, or return an error when invalid.
pub fn format_ie(value: &str) -> Result<String, SeloError> {
    let d = mod11::only_digits(value);
    if d.len() == 12 && sp_validate(&d) {
        return Ok(format!("{}.{}.{}.{}", &d[0..3], &d[3..6], &d[6..9], &d[9..12]));
    }
    %s
}

/// Compute a rightmost-digit DV over the leading digits with weights.
fn rightmost_dv(digits: &[i32], weights: &[i32]) -> i32 {
    let mut total = 0;
    for i in 0..weights.len() {
        total += digits[i] * weights[i];
    }
    (total %% 11) %% 10
}

/// Return a valid SP IE in masked form (AAA.AAA.AAA.AAA).
pub fn generate_ie() -> String {
    let mut d: Vec<i32> = vec![0; 12];
    for i in 0..8 {
        d[i] = mod11::rand_int(9) as i32;
    }
    d[8] = rightmost_dv(&d[0..8], W1);
    d[9] = mod11::rand_int(9) as i32;
    d[10] = mod11::rand_int(9) as i32;
    d[11] = rightmost_dv(&d[0..11], W2);
    let s: String = d.iter().map(|n| n.to_string()).collect();
    format!("{}.{}.{}.{}", &s[0..3], &s[3..6], &s[6..9], &s[9..12])
}
`, dv1, dv2, ufs, rustThrow("ErrInvalidFormat"))

	return b.String()
}

// renderPlate emits the plate module with hand-rolled (no-regex) matchers for
// national and Mercosul patterns.
func (e rustEmitter) renderPlate() string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::mod11::{self, SeloError};\n\n")

	fmt.Fprintf(&b, `const LETTERS: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ";

/// Match the national pattern ^[A-Z]{3}-?[0-9]{4}$ (optional dash).
fn is_national(v: &str) -> bool {
    let b = v.as_bytes();
    // Without dash: AAA9999 (7 chars).
    if b.len() == 7 {
        return b[0..3].iter().all(|c| c.is_ascii_uppercase())
            && b[3..7].iter().all(|c| c.is_ascii_digit());
    }
    // With dash: AAA-9999 (8 chars).
    if b.len() == 8 {
        return b[0..3].iter().all(|c| c.is_ascii_uppercase())
            && b[3] == b'-'
            && b[4..8].iter().all(|c| c.is_ascii_digit());
    }
    false
}

/// Match the Mercosul pattern ^[A-Z]{3}[0-9][A-Z][0-9]{2}$ (7 chars).
fn is_mercosul(v: &str) -> bool {
    let b = v.as_bytes();
    if b.len() != 7 {
        return false;
    }
    b[0..3].iter().all(|c| c.is_ascii_uppercase())
        && b[3].is_ascii_digit()
        && b[4].is_ascii_uppercase()
        && b[5].is_ascii_digit()
        && b[6].is_ascii_digit()
}

/// Report whether value is a national or Mercosul plate.
pub fn validate_plate(value: &str) -> bool {
    let v = value.trim().to_uppercase();
    is_national(&v) || is_mercosul(&v)
}

/// Canonicalize the plate (national gains a dash), or return an error.
pub fn format_plate(value: &str) -> Result<String, SeloError> {
    let v = value.trim().to_uppercase();
    if is_mercosul(&v) {
        return Ok(v);
    }
    if is_national(&v) {
        let s: String = v.chars().filter(|&c| c != '-').collect();
        return Ok(format!("{}-{}", &s[0..3], &s[3..7]));
    }
    %s
}

/// Return a random valid plate (national or Mercosul).
pub fn generate_plate() -> String {
    let rl = || LETTERS[mod11::rand_int(25) as usize] as char;
    let rd = || char::from(b'0' + mod11::rand_int(9) as u8);
    let rds = |n: usize| -> String { (0..n).map(|_| rd()).collect() };
    let letters: String = (0..3).map(|_| rl()).collect();
    if mod11::rand_int(1) == 0 {
        return format!("{}-{}", letters, rds(4));
    }
    format!("{}{}{}{}", letters, rd(), rl(), rds(2))
}
`, rustThrow("ErrInvalidFormat"))

	return b.String()
}

// renderPIX emits the composite PIX module: dispatch EVP -> email -> phone ->
// CPF -> CNPJ, reusing the cpf/cnpj validators; matchers are hand-rolled.
func (e rustEmitter) renderPIX() string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::cnpj::validate_cnpj;\n")
	b.WriteString("use crate::cpf::validate_cpf;\n")
	b.WriteString("use crate::mod11::{self, SeloError};\n\n")

	fmt.Fprintf(&b, `/// Match the EVP UUIDv4 shape:
/// ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$
fn is_evp(v: &str) -> bool {
    let b = v.as_bytes();
    if b.len() != 36 {
        return false;
    }
    if b[8] != b'-' || b[13] != b'-' || b[18] != b'-' || b[23] != b'-' {
        return false;
    }
    let hex = |c: u8| c.is_ascii_hexdigit();
    for (i, &c) in b.iter().enumerate() {
        match i {
            8 | 13 | 18 | 23 => continue,
            14 => {
                if c != b'4' {
                    return false;
                }
            }
            19 => {
                if !matches!(c, b'8' | b'9' | b'a' | b'b' | b'A' | b'B') {
                    return false;
                }
            }
            _ => {
                if !hex(c) {
                    return false;
                }
            }
        }
    }
    true
}

/// Match a +55 phone key: ^\+55\d{10,11}$.
fn is_phone_key(v: &str) -> bool {
    let b = v.as_bytes();
    if b.len() < 13 || b.len() > 14 {
        return false;
    }
    if b[0] != b'+' || b[1] != b'5' || b[2] != b'5' {
        return false;
    }
    b[3..].iter().all(|c| c.is_ascii_digit())
}

/// Match an email key (contains '@', simple label/domain checks; no regex).
fn is_email(v: &str) -> bool {
    let at = match v.find('@') {
        Some(i) => i,
        None => return false,
    };
    if v[at + 1..].contains('@') {
        return false;
    }
    let local = &v[0..at];
    let domain = &v[at + 1..];
    if local.is_empty() || domain.is_empty() {
        return false;
    }
    // Local part: [A-Za-z0-9._%%+-]+
    if !local
        .bytes()
        .all(|c| c.is_ascii_alphanumeric() || matches!(c, b'.' | b'_' | b'%%' | b'+' | b'-'))
    {
        return false;
    }
    // Domain: dot-separated labels, each starts/ends alphanumeric, may contain '-'.
    let labels: Vec<&str> = domain.split('.').collect();
    if labels.len() < 2 {
        return false;
    }
    for label in labels {
        if label.is_empty() {
            return false;
        }
        let lb = label.as_bytes();
        if !lb[0].is_ascii_alphanumeric() || !lb[lb.len() - 1].is_ascii_alphanumeric() {
            return false;
        }
        if !lb.iter().all(|c| c.is_ascii_alphanumeric() || *c == b'-') {
            return false;
        }
    }
    true
}

/// Report the PIX key kind, or None when value is not a key.
pub fn detect_pix_kind(value: &str) -> Option<&'static str> {
    let v = value.trim();
    if is_evp(v) {
        return Some("evp");
    }
    if v.contains('@') {
        return if is_email(v) { Some("email") } else { None };
    }
    if v.starts_with('+') {
        return if is_phone_key(v) { Some("phone") } else { None };
    }
    let digits = mod11::only_digits(v).len();
    if digits == 11 && validate_cpf(v) {
        return Some("cpf");
    }
    if digits == 14 && validate_cnpj(v) {
        return Some("cnpj");
    }
    None
}

/// Report whether value is a well-formed PIX key of any kind.
pub fn validate_pix(value: &str) -> bool {
    detect_pix_kind(value).is_some()
}

/// Return the trimmed key verbatim, or return an error when invalid.
pub fn format_pix(value: &str) -> Result<String, SeloError> {
    let v = value.trim();
    if detect_pix_kind(v).is_none() {
        %s
    }
    Ok(v.to_string())
}

/// Return a random, valid EVP (UUIDv4) PIX key.
pub fn generate_pix() -> String {
    let mut bytes = [0u8; 16];
    let a = mod11::rand_u64().to_le_bytes();
    let b = mod11::rand_u64().to_le_bytes();
    bytes[0..8].copy_from_slice(&a);
    bytes[8..16].copy_from_slice(&b);
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    let hex: String = bytes.iter().map(|b| format!("{:02x}", b)).collect();
    format!(
        "{}-{}-{}-{}-{}",
        &hex[0..8],
        &hex[8..12],
        &hex[12..16],
        &hex[16..20],
        &hex[20..32]
    )
}
`, rustThrow("ErrInvalidLength"))

	return b.String()
}

// renderCEP emits the table-lookup CEP module: prefix-range validation, mask
// format, and UF origin from the embedded data::CEP_RANGES table.
func (e rustEmitter) renderCEP() string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::data::CEP_RANGES;\n")
	b.WriteString("use crate::mod11::{self, SeloError};\n\n")

	fmt.Fprintf(&b, `/// Return the UF whose prefix range contains prefix, or None.
fn range_for(prefix: i32) -> Option<&'static str> {
    CEP_RANGES
        .iter()
        .find(|r| r.from <= prefix && prefix <= r.to)
        .map(|r| r.uf)
}

/// Report whether value is a CEP whose prefix maps to a UF.
pub fn validate_cep(value: &str) -> bool {
    let d = mod11::only_digits(value);
    if d.len() != 8 {
        return false;
    }
    let prefix: i32 = d[0..3].parse().unwrap_or(-1);
    range_for(prefix).is_some()
}

/// Mask a CEP as #####-###, or return an error on bad length.
pub fn format_cep(value: &str) -> Result<String, SeloError> {
    let d = mod11::only_digits(value);
    if d.len() != 8 {
        %s
    }
    Ok(format!("{}-{}", &d[0..5], &d[5..8]))
}

/// Return the UF whose prefix range contains value, or an error.
pub fn origin_cep(value: &str) -> Result<String, SeloError> {
    let d = mod11::only_digits(value);
    if d.len() != 8 {
        %s
    }
    let prefix: i32 = d[0..3].parse().unwrap_or(-1);
    match range_for(prefix) {
        Some(uf) => Ok(uf.to_string()),
        None => %s
    }
}

/// Return a random, valid 8-digit CEP (unformatted).
pub fn generate_cep() -> String {
    let r = &CEP_RANGES[mod11::rand_int(CEP_RANGES.len() as u64 - 1) as usize];
    let prefix = r.from + mod11::rand_int((r.to - r.from) as u64) as i32;
    let suffix = mod11::rand_int(99999);
    format!("{:03}{:05}", prefix, suffix)
}
`, rustThrow("ErrInvalidLength"), rustThrow("ErrInvalidLength"), rustThrowExpr("ErrInvalidFormat"))

	return b.String()
}

// renderPhone emits the table-lookup phone module: optional +55/0055 prefix,
// DDD->UF validation, mobile/landline mask, and DDD origin.
func (e rustEmitter) renderPhone() string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::data::DDD_TO_UF;\n")
	b.WriteString("use crate::mod11::{self, SeloError};\n\n")

	fmt.Fprintf(&b, `/// Look up a DDD in the DDD->UF table.
fn ddd_uf(ddd: &str) -> Option<&'static str> {
    DDD_TO_UF.iter().find(|(k, _)| *k == ddd).map(|(_, v)| *v)
}

/// Strip a +55/0055 country prefix, returning the rest or None.
fn national_number(d: &str) -> Option<String> {
    let mut s = d.to_string();
    if s.starts_with("0055") {
        s = s[4..].to_string();
    } else if s.starts_with("55") && s.len() > 11 {
        s = s[2..].to_string();
    }
    if s.is_empty() {
        return None;
    }
    Some(s)
}

/// Report whether value is a valid phone whose DDD maps to a UF.
pub fn validate_phone(value: &str) -> bool {
    let n = match national_number(&mod11::only_digits(value)) {
        Some(n) => n,
        None => return false,
    };
    if n.len() != 10 && n.len() != 11 {
        return false;
    }
    let ddd = &n[0..2];
    if ddd_uf(ddd).is_none() {
        return false;
    }
    if n.len() == 11 && n.as_bytes()[2] != b'9' {
        return false;
    }
    true
}

/// Mask as (DD) NNNNN-NNNN or (DD) NNNN-NNNN, or return an error.
pub fn format_phone(value: &str) -> Result<String, SeloError> {
    let n = match national_number(&mod11::only_digits(value)) {
        Some(n) if n.len() == 10 || n.len() == 11 => n,
        _ => %s
    };
    let ddd = &n[0..2];
    if ddd_uf(ddd).is_none() {
        %s
    }
    let sub = &n[2..];
    if sub.len() == 9 {
        return Ok(format!("({}) {}-{}", ddd, &sub[0..5], &sub[5..9]));
    }
    Ok(format!("({}) {}-{}", ddd, &sub[0..4], &sub[4..8]))
}

/// Return the UF for the phone's DDD, or an error.
pub fn origin_phone(value: &str) -> Result<String, SeloError> {
    let n = match national_number(&mod11::only_digits(value)) {
        Some(n) if n.len() == 10 || n.len() == 11 => n,
        _ => %s
    };
    let ddd = &n[0..2];
    match ddd_uf(ddd) {
        Some(uf) => Ok(uf.to_string()),
        None => %s
    }
}

/// Return a random valid Brazilian phone (unformatted national digits).
pub fn generate_phone() -> String {
    let (ddd, _) = DDD_TO_UF[mod11::rand_int(DDD_TO_UF.len() as u64 - 1) as usize];
    if mod11::rand_int(1) == 0 {
        let mut sub = String::from("9");
        for _ in 0..8 {
            sub.push_str(&mod11::rand_int(9).to_string());
        }
        return format!("{}{}", ddd, sub);
    }
    let mut sub = (2 + mod11::rand_int(3)).to_string();
    for _ in 0..7 {
        sub.push_str(&mod11::rand_int(9).to_string());
    }
    format!("{}{}", ddd, sub)
}
`, rustThrowArm("ErrInvalidLength"), rustThrow("ErrInvalidFormat"),
		rustThrowArm("ErrInvalidLength"), rustThrowExpr("ErrInvalidFormat"))

	return b.String()
}

// renderVoterID emits the dual-DV voter module (bespoke per the spec Note): DV1
// over the 8 sequence digits; DV2 over [uf_digit0, uf_digit1, dv1]; UF 01..28.
func (e rustEmitter) renderVoterID(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(rustFileHeader())
	b.WriteString("use crate::data::VOTER_UF_NAMES;\n")
	b.WriteString("use crate::mod11::{self, CheckDigit, Rule, SeloError};\n\n")

	dv1 := rustCheckDigitLiteral(plan.Checks[0])
	dv2 := rustCheckDigitLiteral(plan.Checks[1])
	fmt.Fprintf(&b, `const DV1: CheckDigit = %s;
const DV2: CheckDigit = %s;

/// Compute the first check digit over the 8 sequence digits.
fn dv1(d: &str) -> i32 {
    let bytes = d.as_bytes();
    let seq: Vec<i32> = (0..8).map(|i| (bytes[i] - b'0') as i32).collect();
    mod11::compute_digit(mod11::weighted_sum(&seq, DV1.weights, false), &DV1)
}

/// Compute the second check digit over [uf0, uf1, dv1].
fn dv2(d: &str, dv1: i32) -> i32 {
    let bytes = d.as_bytes();
    let vals = [(bytes[8] - b'0') as i32, (bytes[9] - b'0') as i32, dv1];
    mod11::compute_digit(mod11::weighted_sum(&vals, DV2.weights, false), &DV2)
}

/// Report whether value is a well-formed Título Eleitoral.
pub fn validate_voter_id(value: &str) -> bool {
    let d = mod11::only_digits(value);
    if d.len() != 12 {
        return false;
    }
    if mod11::all_equal(&d) {
        return false;
    }
    let bytes = d.as_bytes();
    let uf_code = ((bytes[8] - b'0') as i32) * 10 + (bytes[9] - b'0') as i32;
    if !(1..=28).contains(&uf_code) {
        return false;
    }
    let v1 = dv1(&d);
    let v2 = dv2(&d, v1);
    v1 == (bytes[10] - b'0') as i32 && v2 == (bytes[11] - b'0') as i32
}

/// Group the voter ID as "SSSS SSSS UUDD", or return an error.
pub fn format_voter_id(value: &str) -> Result<String, SeloError> {
    let d = mod11::only_digits(value);
    if d.len() != 12 {
        %s
    }
    Ok(format!("{} {} {}", &d[0..4], &d[4..8], &d[8..12]))
}

/// Return the region encoded in the UF code, or an error.
pub fn origin_voter_id(value: &str) -> Result<String, SeloError> {
    let d = mod11::only_digits(value);
    if d.len() != 12 {
        %s
    }
    let bytes = d.as_bytes();
    let uf_code = ((bytes[8] - b'0') as i32) * 10 + (bytes[9] - b'0') as i32;
    match VOTER_UF_NAMES.iter().find(|(k, _)| *k == uf_code).map(|(_, v)| *v) {
        Some(name) => Ok(name.to_string()),
        None => %s
    }
}

/// Return a random, valid Título Eleitoral (12 digits, unformatted).
pub fn generate_voter_id() -> String {
    loop {
        let mut d: Vec<i32> = vec![0; 12];
        for i in 0..8 {
            d[i] = mod11::rand_int(9) as i32;
        }
        let uf = 1 + mod11::rand_int(27) as i32;
        d[8] = uf / 10;
        d[9] = uf %% 10;
        let s: String = d[0..10].iter().map(|n| n.to_string()).collect();
        let v1 = dv1(&s);
        d[10] = v1;
        d[11] = dv2(&s, v1);
        let out: String = d.iter().map(|n| n.to_string()).collect();
        if !mod11::all_equal(&out) {
            return out;
        }
    }
}
`, dv1, dv2, rustThrow("ErrInvalidLength"),
		rustThrow("ErrInvalidLength"), rustThrowExpr("ErrInvalidFormat"))

	return b.String()
}
