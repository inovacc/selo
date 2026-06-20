package codegen

import (
	"fmt"
	"strings"
)

// emit_js_kinds.go holds the per-kind JavaScript module renderers.

// renderCPF emits the CPF module.
func (e jsEmitter) renderCPF(plan KindPlan) string {
	var b strings.Builder
	jsWriteHeader(&b, "CPF_REGIONS")

	dv1 := jsCheckDigitLiteral(plan.Checks[0])
	dv2 := jsCheckDigitLiteral(plan.Checks[1])

	fmt.Fprintf(&b, `const DV1 = %s;
const DV2 = %s;

/** validateCPF reports whether value is a valid CPF (formatted or not). */
export function validateCPF(value) {
  const d = onlyDigits(value);
  if (d.length !== 11) return false;
  if (allEqual(d)) return false;
  const digits = d.split("").map(Number);
  const dv1 = computeDigit(weightedSum(digits.slice(0, 9), DV1.weights), DV1);
  const dv2 = computeDigit(weightedSum(digits.slice(0, 10), DV2.weights), DV2);
  return dv1 === digits[9] && dv2 === digits[10];
}

/** formatCPF renders value as XXX.XXX.XXX-XX, or throws on bad length. */
export function formatCPF(value) {
  const d = onlyDigits(value);
  if (d.length !== 11) %s
  return `+"`${d.slice(0, 3)}.${d.slice(3, 6)}.${d.slice(6, 9)}-${d.slice(9, 11)}`"+`;
}

/** originCPF returns the issuing region from the 9th digit, or throws. */
export function originCPF(value) {
  const d = onlyDigits(value);
  if (d.length < 9) %s
  const region = CPF_REGIONS[Number(d[8])];
  if (region === undefined) %s
  return region;
}
`, dv1, dv2, jsFormatErrorThrow("ErrInvalidLength"),
		jsFormatErrorThrow("ErrInvalidLength"), jsFormatErrorThrow("ErrInvalidLength"))

	b.WriteString(`
/** generateCPF returns a random valid CPF in formatted form (XXX.XXX.XXX-XX). */
export function generateCPF() {
  const d = [];
  for (let i = 0; i < 9; i++) d.push(Math.floor(Math.random() * 10));
  d.push(computeDigit(weightedSum(d.slice(0, 9), DV1.weights), DV1));
  d.push(computeDigit(weightedSum(d.slice(0, 10), DV2.weights), DV2));
  return formatCPF(d.join(""));
}
`)

	return b.String()
}

// renderSimpleNumeric emits a single-DV numeric kind (PIS).
func (e jsEmitter) renderSimpleNumeric(plan KindPlan, name string, length int) string {
	var b strings.Builder
	jsWriteHeader(&b, "")

	dv := jsCheckDigitLiteral(plan.Checks[0])
	base := length - 1
	mask := jsMaskExpr(plan.Mask, "d")

	fmt.Fprintf(&b, `const DV = %s;

/** validate%[2]s reports whether value is a valid %[2]s. */
export function validate%[2]s(value) {
  const d = onlyDigits(value);
  if (d.length !== %[3]d) return false;
  if (allEqual(d)) return false;
  const digits = d.split("").map(Number);
  const dv = computeDigit(weightedSum(digits.slice(0, %[4]d), DV.weights), DV);
  return dv === digits[%[4]d];
}

/** format%[2]s renders the canonical mask, or throws on bad length. */
export function format%[2]s(value) {
  const d = onlyDigits(value);
  if (d.length !== %[3]d) %[5]s
  return %[6]s;
}
`, dv, name, length, base, jsFormatErrorThrow("ErrInvalidLength"), mask)

	fmt.Fprintf(&b, `
/** generate%[1]s returns a random valid %[1]s in formatted form. */
export function generate%[1]s() {
  let out;
  do {
    const d = Array.from({ length: %[2]d }, () => Math.floor(Math.random() * 10));
    const dv = computeDigit(weightedSum(d, DV.weights), DV);
    out = d.join("") + dv;
  } while (allEqual(out));
  return format%[1]s(out);
}
`, name, base)

	return b.String()
}

// renderRenavam emits RENAVAM.
func (e jsEmitter) renderRenavam(plan KindPlan) string {
	var b strings.Builder
	jsWriteHeader(&b, "")

	dv := jsCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `const DV = %s;

/** validateRenavam reports whether value is a valid 11-digit RENAVAM. */
export function validateRenavam(value) {
  const d = onlyDigits(value);
  if (d.length !== 11) return false;
  if (allEqual(d)) return false;
  const digits = d.split("").map(Number);
  const dv = computeDigit(weightedSum(digits.slice(0, 10), DV.weights), DV);
  return dv === digits[10];
}

/** formatRenavam left-pads shorter inputs to 11 digits (no separator mask). */
export function formatRenavam(value) {
  let d = onlyDigits(value);
  if (d.length < 11) d = "0".repeat(11 - d.length) + d;
  return d;
}

/** generateRenavam returns a random valid 11-digit RENAVAM. */
export function generateRenavam() {
  let out;
  do {
    const d = Array.from({ length: 10 }, () => Math.floor(Math.random() * 10));
    const dv = computeDigit(weightedSum(d, DV.weights), DV);
    out = d.join("") + dv;
  } while (allEqual(out));
  return out;
}
`, dv)

	return b.String()
}

// renderCNH emits the coupled-DV CNH module.
func (e jsEmitter) renderCNH(_ KindPlan) string {
	var b strings.Builder
	jsWriteHeader(&b, "")

	b.WriteString(`/** cnhCheckDigits computes both coupled CNH check digits over the 9-digit base. */
function cnhCheckDigits(base) {
  let dsc = 0;
  let sum = 0;
  for (let i = 0; i < 9; i++) sum += Number(base[i]) * (9 - i);
  let r = sum % 11;
  let dv1;
  if (r >= 10) {
    dv1 = 0;
    dsc = 2;
  } else {
    dv1 = r;
  }
  sum = 0;
  for (let i = 0; i < 9; i++) sum += Number(base[i]) * (1 + i);
  r = (sum % 11) - dsc;
  if (r < 0) r += 11;
  const dv2 = r >= 10 ? 0 : r;
  return [dv1, dv2];
}

/** validateCNH reports whether value is a valid 11-digit CNH. */
export function validateCNH(value) {
  const d = onlyDigits(value);
  if (d.length !== 11) return false;
  if (allEqual(d)) return false;
  const [dv1, dv2] = cnhCheckDigits(d.slice(0, 9));
  return dv1 === Number(d[9]) && dv2 === Number(d[10]);
}

/** formatCNH returns the cleaned 11-digit CNH (no separator mask). */
export function formatCNH(value) {
  const d = onlyDigits(value);
  if (d.length !== 11) ` + jsFormatErrorThrow("ErrInvalidLength") + `
  return d;
}

/** generateCNH returns a random valid 11-digit CNH. */
export function generateCNH() {
  let out;
  do {
    const base = Array.from({ length: 9 }, () => Math.floor(Math.random() * 10)).join("");
    const [dv1, dv2] = cnhCheckDigits(base);
    out = base + dv1 + dv2;
  } while (allEqual(out));
  return out;
}
`)

	return b.String()
}

// renderCNS emits the verify-only sum-zero module.
func (e jsEmitter) renderCNS(plan KindPlan) string {
	var b strings.Builder
	jsWriteHeader(&b, "")

	dv := jsCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `const DV = %s;

/** validateCNS reports whether value is a well-formed CNS (sum %% 11 === 0). */
export function validateCNS(value) {
  const d = onlyDigits(value);
  if (d.length !== 15) return false;
  if (allEqual(d)) return false;
  const lead = d[0];
  if (!(lead === "1" || lead === "2" || lead === "7" || lead === "8" || lead === "9")) {
    return false;
  }
  const digits = d.split("").map(Number);
  return computeDigit(weightedSum(digits, DV.weights), DV) === 0;
}

/** formatCNS returns the cleaned 15-digit CNS (no separator mask). */
export function formatCNS(value) {
  const d = onlyDigits(value);
  if (d.length !== 15) %s
  return d;
}
`, dv, jsFormatErrorThrow("ErrInvalidLength"))

	b.WriteString(`const CNS_PREFIXES = ["1", "2", "7", "8", "9"];

/** generateCNS returns a random valid 15-digit CNS. */
export function generateCNS() {
  while (true) {
    const d = [];
    d.push(Number(CNS_PREFIXES[Math.floor(Math.random() * CNS_PREFIXES.length)]));
    for (let i = 1; i < 14; i++) d.push(Math.floor(Math.random() * 10));
    let partial = 0;
    for (let i = 0; i < 14; i++) partial += d[i] * (15 - i);
    const last = (11 - (partial % 11)) % 11;
    if (last === 10) continue;
    d.push(last);
    const out = d.join("");
    if (!allEqual(out)) return out;
  }
}
`)

	return b.String()
}

// renderCNPJ emits the alphanumeric CNPJ module.
func (e jsEmitter) renderCNPJ(plan KindPlan) string {
	var b strings.Builder
	jsWriteHeader(&b, "")

	dv := jsCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `const DV = %s;

/** cnpjClean uppercases and keeps only [0-9A-Z], capped at 14 chars. */
function cnpjClean(value) {
  let out = "";
  for (const ch of value) {
    const up = ch.toUpperCase();
    if ((up >= "0" && up <= "9") || (up >= "A" && up <= "Z")) {
      out += up;
      if (out.length === 14) break;
    }
  }
  return out;
}

/** cnpjDV computes one check digit over the base string (RL-cycling weights). */
function cnpjDV(base) {
  const vals = base.split("").map(charValue);
  return computeDigit(weightedSum(vals, DV.weights, true), DV);
}

/** validateCNPJ reports whether value is a valid alphanumeric CNPJ. */
export function validateCNPJ(value) {
  const c = cnpjClean(value);
  if (c.length !== 14) return false;
  if (allEqual(c)) return false;
  if (c[12] < "0" || c[12] > "9") return false;
  if (c[13] < "0" || c[13] > "9") return false;
  const base = c.slice(0, 12);
  const dv1 = cnpjDV(base);
  const dv2 = cnpjDV(base + String(dv1));
  return dv1 === Number(c[12]) && dv2 === Number(c[13]);
}

/** formatCNPJ renders value as XX.XXX.XXX/XXXX-XX, or throws on bad length. */
export function formatCNPJ(value) {
  const c = cnpjClean(value);
  if (c.length !== 14) %s
  return `+"`${c.slice(0, 2)}.${c.slice(2, 5)}.${c.slice(5, 8)}/${c.slice(8, 12)}-${c.slice(12, 14)}`"+`;
}
`, dv, jsFormatErrorThrow("ErrInvalidLength"))

	b.WriteString(`const CNPJ_ALPHANUM = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ";

/** generateCNPJ returns a random valid alphanumeric CNPJ. */
export function generateCNPJ() {
  const base = Array.from({ length: 12 }, () =>
    CNPJ_ALPHANUM[Math.floor(Math.random() * CNPJ_ALPHANUM.length)]
  ).join("");
  const dv1 = cnpjDV(base);
  const dv2 = cnpjDV(base + String(dv1));
  return base + dv1 + dv2;
}
`)

	return b.String()
}
