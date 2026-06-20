package codegen

import (
	"fmt"
	"strconv"
	"strings"
)

// emit_js_kinds2.go holds the remaining per-kind JavaScript module renderers.

// jsMaskExpr converts a '#'/'X'-placeholder mask into a JS template-literal
// expression slicing the cleaned digit variable `v`.
func jsMaskExpr(mask, v string) string {
	var b strings.Builder
	b.WriteString("`")

	pos := 0

	i := 0
	for i < len(mask) {
		c := mask[i]
		if c == '#' || c == 'X' {
			start := pos

			for i < len(mask) && (mask[i] == '#' || mask[i] == 'X') {
				i++
				pos++
			}

			fmt.Fprintf(&b, "${%s.slice(%d, %d)}", v, start, pos)

			continue
		}
		// literal separator
		b.WriteByte(c)

		i++
	}

	b.WriteString("`")

	return b.String()
}

// jsStringArray renders a string slice as a JS array literal.
func jsStringArray(fallback []string) string {
	quoted := make([]string, len(fallback))
	for i, s := range fallback {
		quoted[i] = strconv.Quote(s)
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}

// renderRG emits the UF-scoped RG module.
func (e jsEmitter) renderRG(plan KindPlan) string {
	var b strings.Builder
	jsWriteHeader(&b, "")

	dv := jsCheckDigitLiteral(plan.Checks[0])
	ufs := jsStringArray([]string{"SP", "RJ"})
	fmt.Fprintf(&b, `const DV = %s;

/** RG_UFS lists the implemented federative units (shared SP/RJ algorithm). */
export const RG_UFS = %s;

/** rgParse strips formatting and returns the 8 base digits + check value. */
function rgParse(value) {
  let cleaned = "";
  for (const ch of value) {
    if ((ch >= "0" && ch <= "9") || ch === "X" || ch === "x") cleaned += ch;
  }
  if (cleaned.length !== 9) return null;
  const last = cleaned[8];
  let check;
  if (last === "X" || last === "x") check = 10;
  else if (last === "0") check = 11;
  else if (last >= "1" && last <= "9") check = Number(last);
  else return null;
  const base = [];
  for (let i = 0; i < 8; i++) {
    const c = cleaned[i];
    if (c < "0" || c > "9") return null;
    base.push(Number(c));
  }
  return { base, check };
}

/** validateRGForUF validates value as an RG for the given UF (SP/RJ only). */
export function validateRGForUF(value, uf) {
  if (!RG_UFS.includes(uf)) return false;
  const p = rgParse(value);
  if (p === null) return false;
  return computeDigit(weightedSum(p.base, DV.weights), DV) === p.check;
}

/** validateRG validates value under any implemented UF (first match wins). */
export function validateRG(value) {
  return RG_UFS.some((uf) => validateRGForUF(value, uf));
}

/** formatRG renders an RG as XX.XXX.XXX-C (check char normalized). */
export function formatRG(value) {
  const p = rgParse(value);
  if (p === null) %s
  const checkChar = encodeDigit(p.check, DV);
  const d = p.base.join("");
  return `+"`${d.slice(0, 2)}.${d.slice(2, 5)}.${d.slice(5, 8)}-${checkChar}`"+`;
}

/** generateRG returns a random valid SP-style RG in masked form (XX.XXX.XXX-C). */
export function generateRG() {
  const base = Array.from({ length: 8 }, () => Math.floor(Math.random() * 10));
  const dv = computeDigit(weightedSum(base, DV.weights), DV);
  const checkChar = encodeDigit(dv, DV);
  const d = base.join("");
  return `+"`${d.slice(0, 2)}.${d.slice(2, 5)}.${d.slice(5, 8)}-${checkChar}`"+`;
}
`, dv, ufs, jsFormatErrorThrow("ErrInvalidFormat"))

	return b.String()
}

// renderIE emits the UF-scoped IE module (SP only).
func (e jsEmitter) renderIE(plan KindPlan) string {
	var b strings.Builder
	jsWriteHeader(&b, "")

	dv1 := jsCheckDigitLiteral(plan.Checks[0])
	dv2 := jsCheckDigitLiteral(plan.Checks[1])
	ufs := jsStringArray([]string{"SP"})
	fmt.Fprintf(&b, `const DV1 = %s;
const DV2 = %s;

/** IE_UFS lists the implemented federative units (SP only). */
export const IE_UFS = %s;

/** ieSPValidate validates a 12-digit São Paulo IE. */
function ieSPValidate(d) {
  if (d.length !== 12) return false;
  const digits = d.split("").map(Number);
  if (computeDigit(weightedSum(digits.slice(0, 8), DV1.weights), DV1) !== digits[8]) {
    return false;
  }
  return computeDigit(weightedSum(digits.slice(0, 11), DV2.weights), DV2) === digits[11];
}

/** validateIEForUF validates value as an IE for the given UF (SP only). */
export function validateIEForUF(value, uf) {
  if (uf !== "SP") return false;
  const d = onlyDigits(value);
  if (d.length !== 12) return false;
  return ieSPValidate(d);
}

/** validateIE validates value under any implemented UF (first match wins). */
export function validateIE(value) {
  return IE_UFS.some((uf) => validateIEForUF(value, uf));
}

/** formatIE renders SP IE as AAA.AAA.AAA.AAA, or throws when invalid. */
export function formatIE(value) {
  const d = onlyDigits(value);
  if (d.length === 12 && ieSPValidate(d)) {
    return `+"`${d.slice(0, 3)}.${d.slice(3, 6)}.${d.slice(6, 9)}.${d.slice(9, 12)}`"+`;
  }
  %s
}
`, dv1, dv2, ufs, jsFormatErrorThrow("ErrInvalidFormat"))

	b.WriteString(`const IE_W1 = [1, 3, 4, 5, 6, 7, 8, 10];
const IE_W2 = [3, 2, 10, 9, 8, 7, 6, 5, 4, 3, 2];

function ieRightmostDV(digits, weights) {
  let sum = 0;
  for (let i = 0; i < weights.length; i++) sum += digits[i] * weights[i];
  return (sum % 11) % 10;
}

/** generateIE returns a random valid SP IE in masked form (AAA.AAA.AAA.AAA). */
export function generateIE() {
  const d = new Array(12).fill(0);
  for (let i = 0; i < 8; i++) d[i] = Math.floor(Math.random() * 10);
  d[8] = ieRightmostDV(d.slice(0, 8), IE_W1);
  d[9] = Math.floor(Math.random() * 10);
  d[10] = Math.floor(Math.random() * 10);
  d[11] = ieRightmostDV(d.slice(0, 11), IE_W2);
  const s = d.join("");
  return ` + "`${s.slice(0, 3)}.${s.slice(3, 6)}.${s.slice(6, 9)}.${s.slice(9, 12)}`" + `;
}
`)

	return b.String()
}

// renderPlate emits the regex-only plate module.
func (e jsEmitter) renderPlate(_ KindPlan) string {
	var b strings.Builder
	b.WriteString(jsHeaderComment())
	b.WriteString("\n")
	b.WriteString("const NATIONAL = /^[A-Z]{3}-?[0-9]{4}$/;\n")
	b.WriteString("const MERCOSUL = /^[A-Z]{3}[0-9][A-Z][0-9]{2}$/;\n\n")
	b.WriteString(`/** validatePlate reports whether value is a national or Mercosul plate. */
export function validatePlate(value) {
  const v = value.trim().toUpperCase();
  return NATIONAL.test(v) || MERCOSUL.test(v);
}

/** formatPlate canonicalizes the plate (national gains a dash), or throws. */
export function formatPlate(value) {
  const v = value.trim().toUpperCase();
  if (MERCOSUL.test(v)) return v;
  if (NATIONAL.test(v)) {
    const s = v.replace(/-/g, "");
    return ` + "`${s.slice(0, 3)}-${s.slice(3, 7)}`" + `;
  }
  ` + jsFormatErrorThrow("ErrInvalidFormat") + `
}

const PLATE_LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

/** generatePlate returns a random valid plate (national or Mercosul). */
export function generatePlate() {
  const rl = () => PLATE_LETTERS[Math.floor(Math.random() * 26)];
  const rd = () => String(Math.floor(Math.random() * 10));
  const letters = rl() + rl() + rl();
  if (Math.random() < 0.5) {
    return letters + "-" + rd() + rd() + rd() + rd();
  }
  return letters + rd() + rl() + rd() + rd();
}
`)

	return b.String()
}

// renderPIX emits the composite PIX module.
func (e jsEmitter) renderPIX(_ KindPlan) string {
	var b strings.Builder
	b.WriteString(jsHeaderComment())
	b.WriteString("\n")
	b.WriteString("import { onlyDigits } from \"./mod11.js\";\n")
	b.WriteString("import { validateCPF } from \"./cpf.js\";\n")
	b.WriteString("import { validateCNPJ } from \"./cnpj.js\";\n\n")
	b.WriteString("const EVP = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$/;\n")
	b.WriteString("const PHONE = /^\\+55\\d{10,11}$/;\n")
	b.WriteString("const EMAIL = /^[A-Za-z0-9._%+\\-]+@[A-Za-z0-9](?:[A-Za-z0-9\\-]*[A-Za-z0-9])?(?:\\.[A-Za-z0-9](?:[A-Za-z0-9\\-]*[A-Za-z0-9])?)+$/;\n\n")
	b.WriteString(`/** detectPIXKind reports the PIX key kind, or null when value is not a key. */
export function detectPIXKind(value) {
  const v = value.trim();
  if (EVP.test(v)) return "evp";
  if (v.includes("@")) return EMAIL.test(v) ? "email" : null;
  if (v.startsWith("+")) return PHONE.test(v) ? "phone" : null;
  const digits = onlyDigits(v).length;
  if (digits === 11 && validateCPF(v)) return "cpf";
  if (digits === 14 && validateCNPJ(v)) return "cnpj";
  return null;
}

/** validatePIX reports whether value is a well-formed PIX key of any kind. */
export function validatePIX(value) {
  return detectPIXKind(value) !== null;
}

/** formatPIX returns the trimmed key verbatim, or throws when invalid. */
export function formatPIX(value) {
  const v = value.trim();
  if (detectPIXKind(v) === null) ` + jsFormatErrorThrow("ErrInvalidLength") + `
  return v;
}

/** generatePIX returns a random valid EVP (UUIDv4) PIX key. */
export function generatePIX() {
  const b = new Uint8Array(16);
  for (let i = 0; i < 16; i++) b[i] = Math.floor(Math.random() * 256);
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  const h = Array.from(b, (x) => x.toString(16).padStart(2, "0"));
  return h.slice(0, 4).join("") + "-" + h.slice(4, 6).join("") + "-" + h.slice(6, 8).join("") + "-" + h.slice(8, 10).join("") + "-" + h.slice(10).join("");
}
`)

	return b.String()
}

// renderCEP emits the table-lookup CEP module.
func (e jsEmitter) renderCEP(_ KindPlan) string {
	var b strings.Builder
	jsWriteHeader(&b, "CEP_RANGES")
	b.WriteString(`/** cepRangeFor returns the UF whose prefix range contains prefix, or null. */
function cepRangeFor(prefix) {
  for (const r of CEP_RANGES) {
    if (prefix >= r.from && prefix <= r.to) return r.uf;
  }
  return null;
}

/** validateCEP reports whether value is a CEP whose prefix maps to a UF. */
export function validateCEP(value) {
  const d = onlyDigits(value);
  if (d.length !== 8) return false;
  const prefix = Number(d.slice(0, 3));
  return cepRangeFor(prefix) !== null;
}

/** formatCEP masks a CEP as #####-###, or throws on bad length. */
export function formatCEP(value) {
  const d = onlyDigits(value);
  if (d.length !== 8) ` + jsFormatErrorThrow("ErrInvalidLength") + `
  return ` + "`${d.slice(0, 5)}-${d.slice(5, 8)}`" + `;
}

/** originCEP returns the UF whose prefix range contains value, or throws. */
export function originCEP(value) {
  const d = onlyDigits(value);
  if (d.length !== 8) ` + jsFormatErrorThrow("ErrInvalidLength") + `
  const uf = cepRangeFor(Number(d.slice(0, 3)));
  if (uf === null) ` + jsFormatErrorThrow("ErrInvalidFormat") + `
  return uf;
}

/** generateCEP returns a random valid 8-digit CEP (unformatted). */
export function generateCEP() {
  const r = CEP_RANGES[Math.floor(Math.random() * CEP_RANGES.length)];
  const prefix = r.from + Math.floor(Math.random() * (r.to - r.from + 1));
  const suffix = Math.floor(Math.random() * 100000);
  return String(prefix).padStart(3, "0") + String(suffix).padStart(5, "0");
}
`)

	return b.String()
}

// renderPhone emits the table-lookup phone module.
func (e jsEmitter) renderPhone(_ KindPlan) string {
	var b strings.Builder
	jsWriteHeader(&b, "DDD_TO_UF")
	b.WriteString(`/** nationalNumber strips a +55/0055 country prefix, returning the rest. */
function nationalNumber(d) {
  if (d.startsWith("0055")) d = d.slice(4);
  else if (d.startsWith("55") && d.length > 11) d = d.slice(2);
  if (d === "") return null;
  return d;
}

/** validatePhone reports whether value is a valid phone whose DDD maps to a UF. */
export function validatePhone(value) {
  const n = nationalNumber(onlyDigits(value));
  if (n === null) return false;
  if (n.length !== 10 && n.length !== 11) return false;
  const ddd = n.slice(0, 2);
  if (!(ddd in DDD_TO_UF)) return false;
  if (n.length === 11 && n[2] !== "9") return false;
  return true;
}

/** formatPhone masks as (DD) NNNNN-NNNN or (DD) NNNN-NNNN, or throws. */
export function formatPhone(value) {
  const n = nationalNumber(onlyDigits(value));
  if (n === null || (n.length !== 10 && n.length !== 11)) ` + jsFormatErrorThrow("ErrInvalidLength") + `
  const ddd = n.slice(0, 2);
  if (!(ddd in DDD_TO_UF)) ` + jsFormatErrorThrow("ErrInvalidFormat") + `
  const sub = n.slice(2);
  if (sub.length === 9) return ` + "`(${ddd}) ${sub.slice(0, 5)}-${sub.slice(5, 9)}`" + `;
  return ` + "`(${ddd}) ${sub.slice(0, 4)}-${sub.slice(4, 8)}`" + `;
}

/** originPhone returns the UF for the phone's DDD, or throws. */
export function originPhone(value) {
  const n = nationalNumber(onlyDigits(value));
  if (n === null || (n.length !== 10 && n.length !== 11)) ` + jsFormatErrorThrow("ErrInvalidLength") + `
  const ddd = n.slice(0, 2);
  const uf = DDD_TO_UF[ddd];
  if (uf === undefined) ` + jsFormatErrorThrow("ErrInvalidFormat") + `
  return uf;
}

/** generatePhone returns a random valid Brazilian phone number (national digits only). */
export function generatePhone() {
  const ddds = Object.keys(DDD_TO_UF);
  const ddd = ddds[Math.floor(Math.random() * ddds.length)];
  if (Math.random() < 0.5) {
    const sub = "9" + Array.from({ length: 8 }, () => String(Math.floor(Math.random() * 10))).join("");
    return ddd + sub;
  }
  const first = String(2 + Math.floor(Math.random() * 4));
  const sub = first + Array.from({ length: 7 }, () => String(Math.floor(Math.random() * 10))).join("");
  return ddd + sub;
}
`)

	return b.String()
}

// renderVoterID emits the dual-DV voter module.
func (e jsEmitter) renderVoterID(plan KindPlan) string {
	var b strings.Builder
	jsWriteHeader(&b, "VOTER_UF_NAMES")

	dv1 := jsCheckDigitLiteral(plan.Checks[0])
	dv2 := jsCheckDigitLiteral(plan.Checks[1])
	fmt.Fprintf(&b, `const DV1 = %s;
const DV2 = %s;

/** voterDV1 computes the first check digit over the 8 sequence digits. */
function voterDV1(d) {
  const seq = d.slice(0, 8).split("").map(Number);
  return computeDigit(weightedSum(seq, DV1.weights), DV1);
}

/** voterDV2 computes the second check digit over [uf0, uf1, dv1]. */
function voterDV2(d, dv1) {
  const vals = [Number(d[8]), Number(d[9]), dv1];
  return computeDigit(weightedSum(vals, DV2.weights), DV2);
}

/** validateVoterId reports whether value is a well-formed Título Eleitoral. */
export function validateVoterId(value) {
  const d = onlyDigits(value);
  if (d.length !== 12) return false;
  if (allEqual(d)) return false;
  const ufCode = Number(d[8]) * 10 + Number(d[9]);
  if (ufCode < 1 || ufCode > 28) return false;
  const dv1 = voterDV1(d);
  const dv2 = voterDV2(d, dv1);
  return dv1 === Number(d[10]) && dv2 === Number(d[11]);
}

/** formatVoterId groups the voter ID as "SSSS SSSS UUDD", or throws. */
export function formatVoterId(value) {
  const d = onlyDigits(value);
  if (d.length !== 12) %s
  return `+"`${d.slice(0, 4)} ${d.slice(4, 8)} ${d.slice(8, 12)}`"+`;
}

/** originVoterId returns the region encoded in the UF code, or throws. */
export function originVoterId(value) {
  const d = onlyDigits(value);
  if (d.length !== 12) %s
  const ufCode = Number(d[8]) * 10 + Number(d[9]);
  const name = VOTER_UF_NAMES[ufCode];
  if (name === undefined) %s
  return name;
}
`, dv1, dv2, jsFormatErrorThrow("ErrInvalidLength"),
		jsFormatErrorThrow("ErrInvalidLength"), jsFormatErrorThrow("ErrInvalidFormat"))

	b.WriteString(`
/** generateVoterId returns a random valid 12-digit Título Eleitoral. */
export function generateVoterId() {
  while (true) {
    const d = new Array(12).fill(0);
    for (let i = 0; i < 8; i++) d[i] = Math.floor(Math.random() * 10);
    const uf = 1 + Math.floor(Math.random() * 28);
    d[8] = Math.floor(uf / 10);
    d[9] = uf % 10;
    const s = d.slice(0, 10).join("");
    const dv1 = voterDV1(s);
    d[10] = dv1;
    d[11] = voterDV2(s, dv1);
    const out = d.join("");
    if (!allEqual(out)) return out;
  }
}
`)

	return b.String()
}
