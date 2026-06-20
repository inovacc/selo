package codegen

import (
	"fmt"
	"strconv"
	"strings"
)

// emit_ts_kinds2.go holds the shared TS render helpers plus the remaining
// per-kind renderers (RG/IE UF-scoped, plate/pix regex, cep/phone table lookup,
// voter dual-DV).

// writeHeader writes the generated-file banner and the standard imports from the
// shared mod11 reducer (always) and, when dataImports is non-empty, the named
// data-table symbols from src/data.js.
func writeHeader(b *strings.Builder, _modPath, dataImports string) {
	b.WriteString(headerComment())
	b.WriteString("\n")
	b.WriteString("import {\n")
	b.WriteString("  type CheckDigit,\n")
	b.WriteString("  charValue,\n")
	b.WriteString("  weightedSum,\n")
	b.WriteString("  computeDigit,\n")
	b.WriteString("  encodeDigit,\n")
	b.WriteString("  onlyDigits,\n")
	b.WriteString("  allEqual,\n")
	b.WriteString("} from \"./mod11.js\";\n")

	if dataImports != "" {
		fmt.Fprintf(b, "import { %s } from \"./data.js\";\n", dataImports)
	}

	b.WriteString("\n")
}

// tsMaskExpr converts a '#'/'X'-placeholder mask (e.g. "###.#####.##-#") into a
// TS template-literal expression slicing the cleaned digit variable `v`, e.g.
// `${v.slice(0,3)}.${v.slice(3,8)}.${v.slice(8,10)}-${v.slice(10,11)}`.
func tsMaskExpr(mask, v string) string {
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

// renderRG emits the UF-scoped RG module: 8 base digits + 1 check char
// (10->'X', 11->'0'); SP and RJ share the algorithm.
func (e tsEmitter) renderRG(plan KindPlan) string {
	var b strings.Builder
	writeHeader(&b, "src/mod11.js", "")

	dv := checkDigitLiteral(plan.Checks[0])
	ufs := tsStringArray(plan, []string{"SP", "RJ"})
	fmt.Fprintf(&b, `const DV: CheckDigit = %s;

/** RG_UFS lists the implemented federative units (shared SP/RJ algorithm). */
export const RG_UFS: string[] = %s;

interface RGParsed { base: number[]; check: number }

/** rgParse strips formatting and returns the 8 base digits + check value. */
function rgParse(value: string): RGParsed | null {
  let cleaned = "";
  for (const ch of value) {
    if ((ch >= "0" && ch <= "9") || ch === "X" || ch === "x") cleaned += ch;
  }
  if (cleaned.length !== 9) return null;
  const last = cleaned[8];
  let check: number;
  if (last === "X" || last === "x") check = 10;
  else if (last === "0") check = 11;
  else if (last >= "1" && last <= "9") check = Number(last);
  else return null;
  const base: number[] = [];
  for (let i = 0; i < 8; i++) {
    const c = cleaned[i];
    if (c < "0" || c > "9") return null;
    base.push(Number(c));
  }
  return { base, check };
}

/** validateRGForUF validates value as an RG for the given UF (SP/RJ only). */
export function validateRGForUF(value: string, uf: string): boolean {
  if (!RG_UFS.includes(uf)) return false;
  const p = rgParse(value);
  if (p === null) return false;
  return computeDigit(weightedSum(p.base, DV.weights), DV) === p.check;
}

/** validateRG validates value under any implemented UF (first match wins). */
export function validateRG(value: string): boolean {
  return RG_UFS.some((uf) => validateRGForUF(value, uf));
}

/** formatRG renders an RG as XX.XXX.XXX-C (check char normalized). */
export function formatRG(value: string): string {
  const p = rgParse(value);
  if (p === null) %s
  const checkChar = encodeDigit(p.check, DV);
  const d = p.base.join("");
  return `+"`${d.slice(0, 2)}.${d.slice(2, 5)}.${d.slice(5, 8)}-${checkChar}`"+`;
}

/** generateRG returns a random valid SP-style RG in masked form (XX.XXX.XXX-C). */
export function generateRG(): string {
  const base = Array.from({ length: 8 }, () => Math.floor(Math.random() * 10));
  const dv = computeDigit(weightedSum(base, DV.weights), DV);
  const checkChar = encodeDigit(dv, DV);
  const d = base.join("");
  return `+"`${d.slice(0, 2)}.${d.slice(2, 5)}.${d.slice(5, 8)}-${checkChar}`"+`;
}
`, dv, ufs, formatErrorThrow("ErrInvalidFormat"))

	return b.String()
}

// renderIE emits the UF-scoped IE module (SP only): two rightmost-digit DVs at
// non-adjacent positions 9 and 12.
func (e tsEmitter) renderIE(plan KindPlan) string {
	var b strings.Builder
	writeHeader(&b, "src/mod11.js", "")

	dv1 := checkDigitLiteral(plan.Checks[0])
	dv2 := checkDigitLiteral(plan.Checks[1])
	ufs := tsStringArray(plan, []string{"SP"})
	fmt.Fprintf(&b, `const DV1: CheckDigit = %s;
const DV2: CheckDigit = %s;

/** IE_UFS lists the implemented federative units (SP only). */
export const IE_UFS: string[] = %s;

/** ieSPValidate validates a 12-digit São Paulo IE. */
function ieSPValidate(d: string): boolean {
  if (d.length !== 12) return false;
  const digits = d.split("").map(Number);
  if (computeDigit(weightedSum(digits.slice(0, 8), DV1.weights), DV1) !== digits[8]) {
    return false;
  }
  return computeDigit(weightedSum(digits.slice(0, 11), DV2.weights), DV2) === digits[11];
}

/** validateIEForUF validates value as an IE for the given UF (SP only). */
export function validateIEForUF(value: string, uf: string): boolean {
  if (uf !== "SP") return false;
  const d = onlyDigits(value);
  if (d.length !== 12) return false;
  return ieSPValidate(d);
}

/** validateIE validates value under any implemented UF (first match wins). */
export function validateIE(value: string): boolean {
  return IE_UFS.some((uf) => validateIEForUF(value, uf));
}

/** formatIE renders SP IE as AAA.AAA.AAA.AAA, or throws when invalid. */
export function formatIE(value: string): string {
  const d = onlyDigits(value);
  if (d.length === 12 && ieSPValidate(d)) {
    return `+"`${d.slice(0, 3)}.${d.slice(3, 6)}.${d.slice(6, 9)}.${d.slice(9, 12)}`"+`;
  }
  %s
}
`, dv1, dv2, ufs, formatErrorThrow("ErrInvalidFormat"))

	b.WriteString(`const IE_W1 = [1, 3, 4, 5, 6, 7, 8, 10];
const IE_W2 = [3, 2, 10, 9, 8, 7, 6, 5, 4, 3, 2];

function ieRightmostDV(digits: number[], weights: number[]): number {
  let sum = 0;
  for (let i = 0; i < weights.length; i++) sum += digits[i] * weights[i];
  return (sum % 11) % 10;
}

/** generateIE returns a random valid SP IE in masked form (AAA.AAA.AAA.AAA). */
export function generateIE(): string {
  const d = new Array(12).fill(0) as number[];
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

// renderPlate emits the regex-only plate module (national + Mercosul).
func (e tsEmitter) renderPlate(_ KindPlan) string {
	var b strings.Builder
	b.WriteString(headerComment())
	b.WriteString("\n")
	b.WriteString("const NATIONAL = /^[A-Z]{3}-?[0-9]{4}$/;\n")
	b.WriteString("const MERCOSUL = /^[A-Z]{3}[0-9][A-Z][0-9]{2}$/;\n\n")
	b.WriteString(`/** validatePlate reports whether value is a national or Mercosul plate. */
export function validatePlate(value: string): boolean {
  const v = value.trim().toUpperCase();
  return NATIONAL.test(v) || MERCOSUL.test(v);
}

/** formatPlate canonicalizes the plate (national gains a dash), or throws. */
export function formatPlate(value: string): string {
  const v = value.trim().toUpperCase();
  if (MERCOSUL.test(v)) return v;
  if (NATIONAL.test(v)) {
    const s = v.replace(/-/g, "");
    return ` + "`${s.slice(0, 3)}-${s.slice(3, 7)}`" + `;
  }
  ` + formatErrorThrow("ErrInvalidFormat") + `
}

const PLATE_LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

/** generatePlate returns a random valid plate (national or Mercosul). */
export function generatePlate(): string {
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

// renderPIX emits the composite PIX module: dispatch EVP -> email -> phone ->
// CPF -> CNPJ, reusing the CPF/CNPJ validators.
func (e tsEmitter) renderPIX(_ KindPlan) string {
	var b strings.Builder
	b.WriteString(headerComment())
	b.WriteString("\n")
	b.WriteString("import { onlyDigits } from \"./mod11.js\";\n")
	b.WriteString("import { validateCPF } from \"./cpf.js\";\n")
	b.WriteString("import { validateCNPJ } from \"./cnpj.js\";\n\n")
	b.WriteString("const EVP = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$/;\n")
	b.WriteString("const PHONE = /^\\+55\\d{10,11}$/;\n")
	b.WriteString("const EMAIL = /^[A-Za-z0-9._%+\\-]+@[A-Za-z0-9](?:[A-Za-z0-9\\-]*[A-Za-z0-9])?(?:\\.[A-Za-z0-9](?:[A-Za-z0-9\\-]*[A-Za-z0-9])?)+$/;\n\n")
	b.WriteString(`/** detectPIXKind reports the PIX key kind, or null when value is not a key. */
export function detectPIXKind(value: string): string | null {
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
export function validatePIX(value: string): boolean {
  return detectPIXKind(value) !== null;
}

/** formatPIX returns the trimmed key verbatim, or throws when invalid. */
export function formatPIX(value: string): string {
  const v = value.trim();
  if (detectPIXKind(v) === null) ` + formatErrorThrow("ErrInvalidLength") + `
  return v;
}

/** generatePIX returns a random valid EVP (UUIDv4) PIX key. */
export function generatePIX(): string {
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

// renderCEP emits the table-lookup CEP module: prefix-range validation, mask
// format, and UF origin from the embedded CEP_RANGES table.
func (e tsEmitter) renderCEP(_ KindPlan) string {
	var b strings.Builder
	writeHeader(&b, "src/mod11.js", "CEP_RANGES")
	b.WriteString(`/** cepRangeFor returns the UF whose prefix range contains prefix, or null. */
function cepRangeFor(prefix: number): string | null {
  for (const r of CEP_RANGES) {
    if (prefix >= r.from && prefix <= r.to) return r.uf;
  }
  return null;
}

/** validateCEP reports whether value is a CEP whose prefix maps to a UF. */
export function validateCEP(value: string): boolean {
  const d = onlyDigits(value);
  if (d.length !== 8) return false;
  const prefix = Number(d.slice(0, 3));
  return cepRangeFor(prefix) !== null;
}

/** formatCEP masks a CEP as #####-###, or throws on bad length. */
export function formatCEP(value: string): string {
  const d = onlyDigits(value);
  if (d.length !== 8) ` + formatErrorThrow("ErrInvalidLength") + `
  return ` + "`${d.slice(0, 5)}-${d.slice(5, 8)}`" + `;
}

/** originCEP returns the UF whose prefix range contains value, or throws. */
export function originCEP(value: string): string {
  const d = onlyDigits(value);
  if (d.length !== 8) ` + formatErrorThrow("ErrInvalidLength") + `
  const uf = cepRangeFor(Number(d.slice(0, 3)));
  if (uf === null) ` + formatErrorThrow("ErrInvalidFormat") + `
  return uf;
}

/** generateCEP returns a random valid 8-digit CEP (unformatted). */
export function generateCEP(): string {
  const r = CEP_RANGES[Math.floor(Math.random() * CEP_RANGES.length)];
  const prefix = r.from + Math.floor(Math.random() * (r.to - r.from + 1));
  const suffix = Math.floor(Math.random() * 100000);
  return String(prefix).padStart(3, "0") + String(suffix).padStart(5, "0");
}
`)

	return b.String()
}

// renderPhone emits the table-lookup phone module: optional +55/0055 prefix,
// DDD->UF validation, mobile/landline mask, and DDD origin.
func (e tsEmitter) renderPhone(_ KindPlan) string {
	var b strings.Builder
	writeHeader(&b, "src/mod11.js", "DDD_TO_UF")
	b.WriteString(`/** nationalNumber strips a +55/0055 country prefix, returning the rest. */
function nationalNumber(d: string): string | null {
  if (d.startsWith("0055")) d = d.slice(4);
  else if (d.startsWith("55") && d.length > 11) d = d.slice(2);
  if (d === "") return null;
  return d;
}

/** validatePhone reports whether value is a valid phone whose DDD maps to a UF. */
export function validatePhone(value: string): boolean {
  const n = nationalNumber(onlyDigits(value));
  if (n === null) return false;
  if (n.length !== 10 && n.length !== 11) return false;
  const ddd = n.slice(0, 2);
  if (!(ddd in DDD_TO_UF)) return false;
  if (n.length === 11 && n[2] !== "9") return false;
  return true;
}

/** formatPhone masks as (DD) NNNNN-NNNN or (DD) NNNN-NNNN, or throws. */
export function formatPhone(value: string): string {
  const n = nationalNumber(onlyDigits(value));
  if (n === null || (n.length !== 10 && n.length !== 11)) ` + formatErrorThrow("ErrInvalidLength") + `
  const ddd = n.slice(0, 2);
  if (!(ddd in DDD_TO_UF)) ` + formatErrorThrow("ErrInvalidFormat") + `
  const sub = n.slice(2);
  if (sub.length === 9) return ` + "`(${ddd}) ${sub.slice(0, 5)}-${sub.slice(5, 9)}`" + `;
  return ` + "`(${ddd}) ${sub.slice(0, 4)}-${sub.slice(4, 8)}`" + `;
}

/** originPhone returns the UF for the phone's DDD, or throws. */
export function originPhone(value: string): string {
  const n = nationalNumber(onlyDigits(value));
  if (n === null || (n.length !== 10 && n.length !== 11)) ` + formatErrorThrow("ErrInvalidLength") + `
  const ddd = n.slice(0, 2);
  const uf = DDD_TO_UF[ddd];
  if (uf === undefined) ` + formatErrorThrow("ErrInvalidFormat") + `
  return uf;
}

/** generatePhone returns a random valid Brazilian phone number (national digits only). */
export function generatePhone(): string {
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

// renderVoterID emits the dual-DV voter module (bespoke per the Note): DV1 over
// the 8 sequence digits; DV2 over [ufDigit0, ufDigit1, dv1]; UF code 01..28.
func (e tsEmitter) renderVoterID(plan KindPlan) string {
	var b strings.Builder
	writeHeader(&b, "src/mod11.js", "VOTER_UF_NAMES")

	dv1 := checkDigitLiteral(plan.Checks[0])
	dv2 := checkDigitLiteral(plan.Checks[1])
	fmt.Fprintf(&b, `const DV1: CheckDigit = %s;
const DV2: CheckDigit = %s;

/** voterDV1 computes the first check digit over the 8 sequence digits. */
function voterDV1(d: string): number {
  const seq = d.slice(0, 8).split("").map(Number);
  return computeDigit(weightedSum(seq, DV1.weights), DV1);
}

/** voterDV2 computes the second check digit over [uf0, uf1, dv1]. */
function voterDV2(d: string, dv1: number): number {
  const vals = [Number(d[8]), Number(d[9]), dv1];
  return computeDigit(weightedSum(vals, DV2.weights), DV2);
}

/** validateVoterId reports whether value is a well-formed Título Eleitoral. */
export function validateVoterId(value: string): boolean {
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
export function formatVoterId(value: string): string {
  const d = onlyDigits(value);
  if (d.length !== 12) %s
  return `+"`${d.slice(0, 4)} ${d.slice(4, 8)} ${d.slice(8, 12)}`"+`;
}

/** originVoterId returns the region encoded in the UF code, or throws. */
export function originVoterId(value: string): string {
  const d = onlyDigits(value);
  if (d.length !== 12) %s
  const ufCode = Number(d[8]) * 10 + Number(d[9]);
  const name = VOTER_UF_NAMES[ufCode];
  if (name === undefined) %s
  return name;
}
`, dv1, dv2, formatErrorThrow("ErrInvalidLength"),
		formatErrorThrow("ErrInvalidLength"), formatErrorThrow("ErrInvalidFormat"))

	b.WriteString(`
/** generateVoterId returns a random valid 12-digit Título Eleitoral. */
export function generateVoterId(): string {
  while (true) {
    const d = new Array(12).fill(0) as number[];
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

// tsStringArray renders the UFs list from the plan's vector (if available) or a
// fallback, as a TS string[] literal.
func tsStringArray(_ KindPlan, fallback []string) string {
	quoted := make([]string, len(fallback))
	for i, s := range fallback {
		quoted[i] = strconv.Quote(s)
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}
