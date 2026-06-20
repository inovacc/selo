package codegen

import (
	"fmt"
	"strings"
)

// emit_java_kinds.go holds the per-kind Java class renderers for groups A/B/C.
// Each renders a deterministic class from the declarative KindPlan, faithfully
// translating the M2 TypeScript reference (emit_ts_kinds.go) into Java. The
// check-digit kinds reuse the shared Mod11 reducer; the irregular kinds (CNH
// coupled DVs, CNS sum-zero, CNPJ char-map) carry bespoke fragments, exactly as
// the spec Notes flag.

// javaClassOpen writes the generated-file banner, the package declaration,
// optional imports, and the class header for a kind class.
func javaClassOpen(b *strings.Builder, className, doc string, imports ...string) {
	b.WriteString(javaHeaderComment())
	b.WriteString("package com.inovacc.selo;\n\n")

	for _, imp := range imports {
		fmt.Fprintf(b, "import %s;\n", imp)
	}

	if len(imports) > 0 {
		b.WriteString("\n")
	}

	fmt.Fprintf(b, "/** %s */\n", doc)
	fmt.Fprintf(b, "public final class %s {\n", className)
	fmt.Fprintf(b, "    private %s() {\n", className)
	b.WriteString("    }\n\n")
}

// renderCPF emits the CPF class: two coupled-by-input mod-11 DVs, all-equal
// rejection, mask format, and ninth-digit origin.
func (e javaEmitter) renderCPF(plan KindPlan) string {
	var b strings.Builder
	javaClassOpen(&b, "CPF", "CPF validates, formats, and resolves the origin of a Brazilian CPF.",
		"java.util.Random")

	dv1 := checkDigitNew(plan.Checks[0])
	dv2 := checkDigitNew(plan.Checks[1])

	fmt.Fprintf(&b, `    private static final Mod11.CheckDigit DV1 = %s;
    private static final Mod11.CheckDigit DV2 = %s;

    /** validate reports whether value is a valid CPF (formatted or not). */
    public static boolean validate(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 11) {
            return false;
        }
        if (Mod11.allEqual(d)) {
            return false;
        }
        int[] digits = Mod11.digits(d);
        int dv1 = Mod11.computeDigit(Mod11.weightedSum(Mod11.slice(digits, 0, 9), DV1.weights), DV1);
        int dv2 = Mod11.computeDigit(Mod11.weightedSum(Mod11.slice(digits, 0, 10), DV2.weights), DV2);
        return dv1 == digits[9] && dv2 == digits[10];
    }

    /** format renders value as XXX.XXX.XXX-XX, or throws on bad length. */
    public static String format(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 11) {
            %s
        }
        return d.substring(0, 3) + "." + d.substring(3, 6) + "." + d.substring(6, 9) + "-" + d.substring(9, 11);
    }

    /** origin returns the issuing region from the 9th digit, or throws. */
    public static String origin(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() < 9) {
            %s
        }
        String region = Data.CPF_REGIONS.get(d.charAt(8) - '0');
        if (region == null) {
            %s
        }
        return region;
    }

    /** generate returns a random valid CPF in formatted form (XXX.XXX.XXX-XX). */
    public static String generate() {
        Random rng = new Random();
        int[] d = new int[11];
        for (int i = 0; i < 9; i++) {
            d[i] = rng.nextInt(10);
        }
        d[9] = Mod11.computeDigit(Mod11.weightedSum(Mod11.slice(d, 0, 9), DV1.weights), DV1);
        d[10] = Mod11.computeDigit(Mod11.weightedSum(Mod11.slice(d, 0, 10), DV2.weights), DV2);
        StringBuilder sb = new StringBuilder();
        for (int v : d) {
            sb.append(v);
        }
        return format(sb.toString());
    }
}
`, dv1, dv2, javaThrow("ErrInvalidLength"), javaThrow("ErrInvalidLength"), javaThrow("ErrInvalidLength"))

	return b.String()
}

// renderSimpleNumeric emits a single-DV numeric kind (PIS): mod-11 DV over the
// first length-1 digits, all-equal rejection, and a mask format.
func (e javaEmitter) renderSimpleNumeric(plan KindPlan, name string, length int) string {
	var b strings.Builder
	javaClassOpen(&b, name, name+" validates and formats this single-check-digit numeric document.",
		"java.util.Random")

	dv := checkDigitNew(plan.Checks[0])
	base := length - 1
	mask := javaMaskExpr(plan.Mask, "d")

	fmt.Fprintf(&b, `    private static final Mod11.CheckDigit DV = %s;

    /** validate reports whether value is a valid %[2]s. */
    public static boolean validate(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != %[3]d) {
            return false;
        }
        if (Mod11.allEqual(d)) {
            return false;
        }
        int[] digits = Mod11.digits(d);
        int dv = Mod11.computeDigit(Mod11.weightedSum(Mod11.slice(digits, 0, %[4]d), DV.weights), DV);
        return dv == digits[%[4]d];
    }

    /** format renders the canonical mask, or throws on bad length. */
    public static String format(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != %[3]d) {
            %[5]s
        }
        return %[6]s;
    }

    /** generate returns a random valid %[2]s in formatted form. */
    public static String generate() {
        Random rng = new Random();
        String out;
        do {
            int[] d = new int[%[4]d];
            for (int i = 0; i < %[4]d; i++) {
                d[i] = rng.nextInt(10);
            }
            int dv = Mod11.computeDigit(Mod11.weightedSum(d, DV.weights), DV);
            StringBuilder sb = new StringBuilder();
            for (int v : d) {
                sb.append(v);
            }
            sb.append(dv);
            out = sb.toString();
        } while (Mod11.allEqual(out));
        return format(out);
    }
}
`, dv, name, length, base, javaThrow("ErrInvalidLength"), mask)

	return b.String()
}

// renderRenavam emits RENAVAM: single (sum*10)%11 DV, all-equal rejection, and a
// left-pad-to-11 format (no separator mask).
func (e javaEmitter) renderRenavam(plan KindPlan) string {
	var b strings.Builder
	javaClassOpen(&b, "Renavam", "Renavam validates and formats a Brazilian RENAVAM.",
		"java.util.Random")

	dv := checkDigitNew(plan.Checks[0])
	fmt.Fprintf(&b, `    private static final Mod11.CheckDigit DV = %s;

    /** validate reports whether value is a valid 11-digit RENAVAM. */
    public static boolean validate(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 11) {
            return false;
        }
        if (Mod11.allEqual(d)) {
            return false;
        }
        int[] digits = Mod11.digits(d);
        int dv = Mod11.computeDigit(Mod11.weightedSum(Mod11.slice(digits, 0, 10), DV.weights), DV);
        return dv == digits[10];
    }

    /** format left-pads shorter inputs to 11 digits (no separator mask). */
    public static String format(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() < 11) {
            StringBuilder sb = new StringBuilder();
            for (int i = 0; i < 11 - d.length(); i++) {
                sb.append('0');
            }
            d = sb + d;
        }
        return d;
    }

    /** generate returns a random valid 11-digit RENAVAM. */
    public static String generate() {
        Random rng = new Random();
        String out;
        do {
            int[] d = new int[10];
            for (int i = 0; i < 10; i++) {
                d[i] = rng.nextInt(10);
            }
            int dv = Mod11.computeDigit(Mod11.weightedSum(d, DV.weights), DV);
            StringBuilder sb = new StringBuilder();
            for (int v : d) {
                sb.append(v);
            }
            sb.append(dv);
            out = sb.toString();
        } while (Mod11.allEqual(out));
        return out;
    }
}
`, dv)

	return b.String()
}

// renderCNH emits the coupled-DV CNH class (bespoke fragment per the Note):
// DV1 descending 9..1 (raw remainder >=10 -> DV1=0, carry offset 2); DV2
// ascending 1..9 with the offset subtracted before the mod-11 fold.
func (e javaEmitter) renderCNH(_ KindPlan) string {
	var b strings.Builder
	javaClassOpen(&b, "CNH", "CNH validates and formats a Brazilian CNH (coupled check digits).",
		"java.util.Random")

	fmt.Fprintf(&b, `    /** cnhCheckDigits computes both coupled CNH check digits over the 9-digit base. */
    private static int[] cnhCheckDigits(String base) {
        int dsc = 0;
        int sum = 0;
        for (int i = 0; i < 9; i++) {
            sum += (base.charAt(i) - '0') * (9 - i);
        }
        int r = sum %% 11;
        int dv1;
        if (r >= 10) {
            dv1 = 0;
            dsc = 2;
        } else {
            dv1 = r;
        }
        sum = 0;
        for (int i = 0; i < 9; i++) {
            sum += (base.charAt(i) - '0') * (1 + i);
        }
        r = (sum %% 11) - dsc;
        if (r < 0) {
            r += 11;
        }
        int dv2 = r >= 10 ? 0 : r;
        return new int[]{dv1, dv2};
    }

    /** validate reports whether value is a valid 11-digit CNH. */
    public static boolean validate(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 11) {
            return false;
        }
        if (Mod11.allEqual(d)) {
            return false;
        }
        int[] dv = cnhCheckDigits(d.substring(0, 9));
        return dv[0] == (d.charAt(9) - '0') && dv[1] == (d.charAt(10) - '0');
    }

    /** format returns the cleaned 11-digit CNH (no separator mask). */
    public static String format(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 11) {
            %s
        }
        return d;
    }

    /** generate returns a random valid 11-digit CNH. */
    public static String generate() {
        Random rng = new Random();
        String out;
        do {
            StringBuilder baseSb = new StringBuilder();
            for (int i = 0; i < 9; i++) {
                baseSb.append(rng.nextInt(10));
            }
            String base = baseSb.toString();
            int[] dv = cnhCheckDigits(base);
            out = base + dv[0] + dv[1];
        } while (Mod11.allEqual(out));
        return out;
    }
}
`, javaThrow("ErrInvalidLength"))

	return b.String()
}

// renderCNS emits the verify-only sum-zero class with prefix constraint.
func (e javaEmitter) renderCNS(plan KindPlan) string {
	var b strings.Builder
	javaClassOpen(&b, "CNS", "CNS validates and formats a Brazilian CNS (sum mod 11 == 0).",
		"java.util.Random")

	dv := checkDigitNew(plan.Checks[0])
	fmt.Fprintf(&b, `    private static final Mod11.CheckDigit DV = %s;

    /** validate reports whether value is a well-formed CNS (sum %% 11 == 0). */
    public static boolean validate(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 15) {
            return false;
        }
        if (Mod11.allEqual(d)) {
            return false;
        }
        char lead = d.charAt(0);
        if (!(lead == '1' || lead == '2' || lead == '7' || lead == '8' || lead == '9')) {
            return false;
        }
        int[] digits = Mod11.digits(d);
        return Mod11.computeDigit(Mod11.weightedSum(digits, DV.weights), DV) == 0;
    }

    /** format returns the cleaned 15-digit CNS (no separator mask). */
    public static String format(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 15) {
            %s
        }
        return d;
    }

    private static final int[] CNS_PREFIXES = {1, 2, 7, 8, 9};

    /** generate returns a random valid 15-digit CNS. */
    public static String generate() {
        Random rng = new Random();
        while (true) {
            int[] d = new int[15];
            d[0] = CNS_PREFIXES[rng.nextInt(CNS_PREFIXES.length)];
            for (int i = 1; i < 14; i++) {
                d[i] = rng.nextInt(10);
            }
            int partial = 0;
            for (int i = 0; i < 14; i++) {
                partial += d[i] * (15 - i);
            }
            int last = (11 - (partial %% 11)) %% 11;
            if (last == 10) {
                continue;
            }
            d[14] = last;
            StringBuilder sb = new StringBuilder();
            for (int v : d) {
                sb.append(v);
            }
            String out = sb.toString();
            if (!Mod11.allEqual(out)) {
                return out;
            }
        }
    }
}
`, dv, javaThrow("ErrInvalidLength"))

	return b.String()
}

// renderCNPJ emits the alphanumeric CNPJ class (bespoke char-map + RL-cycling
// weights per the Note): two DVs, last two chars numeric, all-equal rejection.
func (e javaEmitter) renderCNPJ(plan KindPlan) string {
	var b strings.Builder
	javaClassOpen(&b, "CNPJ", "CNPJ validates and formats an alphanumeric Brazilian CNPJ.",
		"java.util.Random")

	dv := checkDigitNew(plan.Checks[0])
	fmt.Fprintf(&b, `    private static final Mod11.CheckDigit DV = %s;

    /** cnpjClean uppercases and keeps only [0-9A-Z], capped at 14 chars. */
    private static String cnpjClean(String value) {
        StringBuilder out = new StringBuilder();
        for (int i = 0; i < value.length(); i++) {
            char up = Character.toUpperCase(value.charAt(i));
            if ((up >= '0' && up <= '9') || (up >= 'A' && up <= 'Z')) {
                out.append(up);
                if (out.length() == 14) {
                    break;
                }
            }
        }
        return out.toString();
    }

    /** cnpjDV computes one check digit over the base string (RL-cycling weights). */
    private static int cnpjDV(String base) {
        int[] vals = new int[base.length()];
        for (int i = 0; i < base.length(); i++) {
            vals[i] = Mod11.charValue(base.charAt(i));
        }
        return Mod11.computeDigit(Mod11.weightedSum(vals, DV.weights, true), DV);
    }

    /** validate reports whether value is a valid alphanumeric CNPJ. */
    public static boolean validate(String value) {
        String c = cnpjClean(value);
        if (c.length() != 14) {
            return false;
        }
        if (Mod11.allEqual(c)) {
            return false;
        }
        if (c.charAt(12) < '0' || c.charAt(12) > '9') {
            return false;
        }
        if (c.charAt(13) < '0' || c.charAt(13) > '9') {
            return false;
        }
        String base = c.substring(0, 12);
        int dv1 = cnpjDV(base);
        int dv2 = cnpjDV(base + dv1);
        return dv1 == (c.charAt(12) - '0') && dv2 == (c.charAt(13) - '0');
    }

    /** format renders value as XX.XXX.XXX/XXXX-XX, or throws on bad length. */
    public static String format(String value) {
        String c = cnpjClean(value);
        if (c.length() != 14) {
            %s
        }
        return c.substring(0, 2) + "." + c.substring(2, 5) + "." + c.substring(5, 8)
                + "/" + c.substring(8, 12) + "-" + c.substring(12, 14);
    }

    private static final String CNPJ_ALPHANUM = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ";

    /** generate returns a random valid alphanumeric CNPJ. */
    public static String generate() {
        Random rng = new Random();
        StringBuilder baseSb = new StringBuilder();
        for (int i = 0; i < 12; i++) {
            baseSb.append(CNPJ_ALPHANUM.charAt(rng.nextInt(CNPJ_ALPHANUM.length())));
        }
        String base = baseSb.toString();
        int dv1 = cnpjDV(base);
        int dv2 = cnpjDV(base + dv1);
        return base + dv1 + dv2;
    }
}
`, dv, javaThrow("ErrInvalidLength"))

	return b.String()
}
